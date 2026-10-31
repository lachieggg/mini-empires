package main

import "math"

type Order int

const (
	OrdIdle Order = iota
	OrdMove
	OrdGather
	OrdReturn
	OrdBuild
	OrdAttack
	OrdGarrison
)

var orderNames = [...]string{"Idle", "Moving", "Gathering", "Returning", "Building", "Attacking", "Garrisoning"}

type Unit struct {
	Kind      UnitKind
	Team      int
	X, Y      float64
	Facing    float64
	HP, MaxHP float64
	Order     Order
	TX, TY    float64
	Path      []pt
	Res       *Resource
	FarmB     *Building
	GatherKnd ResKind
	LastRes   pt
	Carry     int
	CarryKind ResKind
	Drop      *Building
	BuildB    *Building
	Target    *Unit
	TargetB   *Building
	Shelter   *Building // the building to garrison in, while OrdGarrison
	Inside    *Building // the building the unit is garrisoned in
	Home      pt
	Selected  bool
	Female    bool

	dead, killed                    bool
	carcass                         *Resource
	work, cool, repath, scan, stuck float64
	anim, wander                    float64
	fails                           int
	chase                           pt
	moved, acting                   bool
	healing                         *Unit
}

func (u *Unit) def() *UnitDef { return &unitDefs[u.Kind] }

func (u *Unit) moving() bool { return len(u.Path) > 0 || u.Order == OrdMove }

// QueueItem is a unit, an age or a technology waiting in a building.
type QueueItem struct {
	Unit UnitKind
	Age  int  // the age being researched, or 0
	Tech Tech // the technology being researched plus one, or 0
}

func (q QueueItem) time() float64 {
	switch {
	case q.Age > 0:
		return ageTime[q.Age]
	case q.Tech > 0:
		return techDefs[q.Tech-1].Time
	}
	return unitDefs[q.Unit].TrainTime
}

func (q QueueItem) cost() Cost {
	switch {
	case q.Age > 0:
		return ageCost[q.Age]
	case q.Tech > 0:
		return techDefs[q.Tech-1].Cost
	}
	return unitDefs[q.Unit].Cost
}

type Building struct {
	Kind     BldKind
	Team     int
	TX, TY   int
	HP       float64
	Built    bool
	Progress float64
	Queue    []QueueItem
	TrainT   float64
	Food     int

	HasRally       bool
	RallyX, RallyY float64
	RallyRes       *Resource

	dead, razed bool
	cool        float64
	housedMsg   float64
	farmer      *Unit
}

func (b *Building) def() *BldDef { return &bldDefs[b.Kind] }

func (b *Building) rect() rect {
	s := b.def().Size
	return rect{b.TX, b.TY, b.TX + s - 1, b.TY + s - 1}
}

func (b *Building) center() (float64, float64) {
	s := float64(b.def().Size * TileSize)
	return float64(b.TX*TileSize) + s/2, float64(b.TY*TileSize) + s/2
}

func (b *Building) trains(k UnitKind) bool {
	for _, t := range b.def().Trains {
		if t == k {
			return true
		}
	}
	return false
}

func (g *Game) addBuilding(k BldKind, team, tx, ty int, built bool) *Building {
	b := &Building{Kind: k, Team: team, TX: tx, TY: ty, Built: built, HP: 1, Food: bldDefs[k].Food}
	if k == Farm {
		b.Food += g.bonus(team).FarmFood
	}
	if built {
		b.HP, b.Progress = b.def().HP, 1
	}
	r := b.rect()
	for y := r.y0; y <= r.y1; y++ {
		for x := r.x0; x <= r.x1; x++ {
			t := g.tile(x, y)
			t.Res, t.Bld = nil, b
			if t.Ter == TerWater || t.Ter == TerForest {
				t.Ter = TerDirt
				g.markGroundDirty(r)
			}
		}
	}
	g.buildings = append(g.buildings, b)
	return b
}

// spawnSpot returns the center of the closest free tile ringing r.
func (g *Game) spawnSpot(r rect) (float64, float64) {
	for d := 1; d < 8; d++ {
		for y := r.y0 - d; y <= r.y1+d; y++ {
			for x := r.x0 - d; x <= r.x1+d; x++ {
				if x != r.x0-d && x != r.x1+d && y != r.y0-d && y != r.y1+d {
					continue
				}
				if g.blocked(x, y) || g.unitOnTile(x, y, -1) {
					continue
				}
				return tileCenter(x), tileCenter(y)
			}
		}
	}
	return tileCenter(r.x0), tileCenter(r.y1 + 1)
}

func (g *Game) newUnit(k UnitKind, team int, x, y float64) *Unit {
	u := &Unit{Kind: k, Team: team, X: x, Y: y, Facing: math.Pi / 4}
	u.Female = k == Villager && g.rng.Intn(2) == 0
	u.MaxHP = g.maxHP(u)
	u.HP = u.MaxHP
	g.units = append(g.units, u)
	if tx, ty := toTile(x), toTile(y); inBounds(tx, ty) {
		if o := &g.grid.occ[team][ty*MapW+tx]; *o < 255 {
			*o++
		}
	}
	return u
}

func (g *Game) spawnUnit(k UnitKind, team int, from *Building) *Unit {
	x, y := g.spawnSpot(from.rect())
	u := g.newUnit(k, team, x, y)
	if from.HasRally {
		if r := from.RallyRes; k == Villager && r != nil && r.Amount > 0 {
			g.orderGather(u, r)
		} else {
			g.orderMove(u, from.RallyX, from.RallyY)
		}
	}
	return u
}

func (g *Game) pop(team int) int {
	n := 0
	for _, u := range g.units {
		if u.Team == team {
			n++
		}
	}
	return n
}

func (g *Game) popCap(team int) int {
	n := 0
	for _, b := range g.buildings {
		if b.Team == team && b.Built {
			n += b.def().Pop
		}
	}
	return min(n, MaxPop)
}

func (g *Game) canAfford(team int, c Cost) bool {
	for k, v := range c {
		if g.players[team].Res[k] < v {
			return false
		}
	}
	return true
}

func (g *Game) pay(team int, c Cost) {
	for k, v := range c {
		g.players[team].Res[k] -= v
	}
}

func (g *Game) refund(team int, c Cost) {
	for k, v := range c {
		g.players[team].Res[k] += v
	}
}

// Orders.

func (u *Unit) reset(o Order) {
	u.Order = o
	u.Path, u.Target, u.TargetB, u.BuildB, u.Drop = nil, nil, nil, nil, nil
	u.repath, u.fails = 0, 0
	if u.FarmB != nil && o != OrdGather && o != OrdReturn {
		if u.FarmB.farmer == u {
			u.FarmB.farmer = nil
		}
		u.FarmB = nil
	}
}

func (g *Game) orderMove(u *Unit, x, y float64) {
	x = min(max(x, 1), worldW()-1)
	y = min(max(y, 1), worldH()-1)
	u.reset(OrdMove)
	u.Res = nil
	path, ok := g.findPath(toTile(u.X), toTile(u.Y), tileRect(toTile(x), toTile(y)), 0)
	u.Path = path
	switch {
	case ok:
		u.TX, u.TY = x, y
	case len(path) > 0:
		l := path[len(path)-1]
		u.TX, u.TY = tileCenter(l.X), tileCenter(l.Y)
	default:
		u.TX, u.TY = u.X, u.Y
	}
}

func (g *Game) orderGather(u *Unit, r *Resource) {
	if u.Res != r {
		u.work = 0
	}
	u.reset(OrdGather)
	u.Res, u.GatherKnd = r, r.Kind
}

func (g *Game) orderFarm(u *Unit, b *Building) {
	u.reset(OrdGather)
	if b.farmer != nil && b.farmer != u && !b.farmer.dead && b.farmer.FarmB == b {
		b.farmer.reset(OrdIdle)
	}
	u.Res, u.FarmB, u.GatherKnd = nil, b, Food
	b.farmer = u
}

func (g *Game) orderReturn(u *Unit, b *Building) {
	res, farm := u.Res, u.FarmB
	u.reset(OrdReturn)
	u.Res, u.FarmB, u.Drop = res, farm, b
}

func (g *Game) orderBuild(u *Unit, b *Building) {
	u.reset(OrdBuild)
	u.Res = nil
	u.BuildB = b
}

func (g *Game) orderAttackUnit(u *Unit, e *Unit) {
	u.reset(OrdAttack)
	u.Target = e
}

func (g *Game) orderAttackBld(u *Unit, b *Building) {
	u.reset(OrdAttack)
	u.TargetB = b
}

// Movement.

func (g *Game) tryMove(u *Unit, dx, dy float64) {
	if g.blockedAt(u.X, u.Y) || !g.blockedAt(u.X+dx, u.Y+dy) {
		u.X += dx
		u.Y += dy
		return
	}
	if !g.blockedAt(u.X+dx, u.Y) {
		u.X += dx
	} else if !g.blockedAt(u.X, u.Y+dy) {
		u.Y += dy
	}
}

// moveToward steps u toward (x, y) and reports whether it got there.
func (g *Game) moveToward(u *Unit, x, y float64) bool {
	dx, dy := x-u.X, y-u.Y
	d := math.Hypot(dx, dy)
	step := g.speedOf(u) * dt
	u.moved = true
	if d <= step {
		g.tryMove(u, dx, dy)
		return true
	}
	u.Facing = math.Atan2(dy, dx)
	g.tryMove(u, dx/d*step, dy/d*step)
	if d-math.Hypot(x-u.X, y-u.Y) < step*0.25 {
		u.stuck += dt
	} else {
		u.stuck = max(0, u.stuck-dt)
	}
	return false
}

func (g *Game) followPath(u *Unit) {
	p := u.Path[0]
	if g.blocked(p.X, p.Y) || u.stuck > 0.6 {
		u.Path, u.stuck, u.repath = nil, 0, 0
		return
	}
	cx, cy := tileCenter(p.X), tileCenter(p.Y)
	if math.Hypot(cx-u.X, cy-u.Y) < 12 || g.moveToward(u, cx, cy) {
		u.Path = u.Path[1:]
	}
}

const (
	Moving = iota
	Arrived
	Unreachable
)

// approach walks u toward the tile rectangle r until it is within reach pixels.
func (g *Game) approach(u *Unit, r rect, reach float64) int {
	if distToRect(u.X, u.Y, r) <= reach {
		u.Path, u.fails = nil, 0
		return Arrived
	}
	if len(u.Path) == 0 && u.repath <= 0 {
		path, ok := g.findPath(toTile(u.X), toTile(u.Y), r, 1)
		u.repath = 0.5
		if ok {
			u.fails = 0
		} else if len(path) == 0 {
			if u.fails++; u.fails > 3 {
				u.fails = 0
				return Unreachable
			}
		}
		u.Path = path
	}
	if len(u.Path) > 0 {
		g.followPath(u)
		return Moving
	}
	px, py := nearestOnRect(u.X, u.Y, r)
	g.moveToward(u, px, py)
	return Moving
}

// Per-tick unit logic.

func (g *Game) updateUnit(u *Unit) {
	if u.cool > 0 {
		u.cool -= dt
	}
	u.repath -= dt
	u.anim += dt
	u.moved, u.acting = false, false
	if u.def().Heal > 0 && (u.Order == OrdIdle || u.Order == OrdMove && u.healing != nil) {
		g.updateHeal(u)
		return
	}
	switch u.Order {
	case OrdIdle:
		d := u.def()
		switch {
		case d.Animal:
			g.wanderAnimal(u)
		case d.OnlyBuildings:
			if b := g.nearestEnemyBuilding(u.Team, u.X, u.Y, d.Sight+g.rangeOf(u)); b != nil {
				g.orderAttackBld(u, b)
			}
		case d.Military:
			if e := g.nearestEnemyUnit(u.Team, u.X, u.Y, d.Sight); e != nil {
				g.orderAttackUnit(u, e)
			}
		}
	case OrdGarrison:
		g.updateGarrison(u)
	case OrdMove:
		switch {
		case len(u.Path) > 0:
			g.followPath(u)
		case u.stuck > 1:
			u.stuck = 0
			if u.fails++; u.fails > 3 {
				u.Order = OrdIdle
			} else {
				u.Path, _ = g.findPath(toTile(u.X), toTile(u.Y), tileRect(toTile(u.TX), toTile(u.TY)), 0)
			}
		case g.moveToward(u, u.TX, u.TY):
			u.Order = OrdIdle
		}
	case OrdGather:
		g.updateGather(u)
	case OrdReturn:
		g.updateReturn(u)
	case OrdBuild:
		g.updateBuild(u)
	case OrdAttack:
		g.updateAttack(u)
	}
}

func (g *Game) wanderAnimal(u *Unit) {
	if u.wander -= dt; u.wander > 0 {
		return
	}
	u.wander = 4 + g.rng.Float64()*6
	x, y := u.Home.X+g.rng.Intn(7)-3, u.Home.Y+g.rng.Intn(7)-3
	if !g.blocked(x, y) {
		g.orderMove(u, tileCenter(x), tileCenter(y))
	}
}

func (g *Game) gatherTime(u *Unit) float64 {
	return GatherTime / (1 + g.bonus(u.Team).Gather[u.GatherKnd])
}

func (g *Game) updateGather(u *Unit) {
	if b := u.FarmB; b != nil {
		g.updateFarm(u, b)
		return
	}
	r := u.Res
	if r == nil || r.Amount <= 0 {
		r = g.findReachableResource(toTile(u.X), toTile(u.Y), kindIs(u.GatherKnd), u.LastRes.X, u.LastRes.Y, 8)
		if r == nil {
			if u.Carry > 0 {
				g.orderReturn(u, nil)
			} else {
				u.Order = OrdIdle
			}
			return
		}
		u.Res, u.Path, u.fails = r, nil, 0
	}
	if u.Carry > 0 && u.CarryKind != r.Kind {
		u.Carry = 0
	}
	if u.Carry >= g.carryCap(u) {
		g.orderReturn(u, nil)
		return
	}
	switch g.approach(u, tileRect(r.TX, r.TY), Reach) {
	case Arrived:
		u.acting = true
		u.LastRes = pt{r.TX, r.TY}
		u.Facing = math.Atan2(tileCenter(r.TY)-u.Y, tileCenter(r.TX)-u.X)
		if u.work += dt; u.work >= g.gatherTime(u) {
			u.work = 0
			r.Amount--
			u.Carry++
			u.CarryKind = r.Kind
			if r.Amount <= 0 {
				g.removeResource(r)
			}
		}
	case Unreachable:
		if alt := g.findReachableResource(toTile(u.X), toTile(u.Y), kindIs(r.Kind), r.TX, r.TY, 8); alt != nil && alt != r {
			u.Res = alt
		} else {
			u.Order = OrdIdle
		}
	}
}

func (g *Game) updateFarm(u *Unit, b *Building) {
	if b.dead || !b.Built || b.Food <= 0 {
		u.FarmB = nil
		if u.Carry > 0 {
			g.orderReturn(u, nil)
		} else {
			u.Order = OrdIdle
		}
		return
	}
	if u.Carry > 0 && u.CarryKind != Food {
		u.Carry = 0
	}
	if u.Carry >= g.carryCap(u) {
		g.orderReturn(u, nil)
		return
	}
	if g.approach(u, b.rect(), 0) != Arrived {
		return
	}
	u.acting = true
	if u.work += dt; u.work >= g.gatherTime(u)*1.3 {
		u.work = 0
		b.Food--
		u.Carry++
		u.CarryKind = Food
		if b.Food <= 0 {
			b.HP = 0
			b.razed = true // exhausted, not destroyed by an enemy
		}
	}
}

func (g *Game) nearestDropSite(u *Unit, k ResKind) *Building {
	var best *Building
	bd := math.Inf(1)
	for _, b := range g.buildings {
		if b.Team == u.Team && b.Built && b.def().Drops[k] {
			if d := distToRect(u.X, u.Y, b.rect()); d < bd {
				best, bd = b, d
			}
		}
	}
	return best
}

func (g *Game) updateReturn(u *Unit) {
	b := u.Drop
	if b == nil || b.dead || !b.Built || !b.def().Drops[u.CarryKind] {
		b = g.nearestDropSite(u, u.CarryKind)
		u.Drop, u.Path = b, nil
	}
	if b == nil {
		u.Order = OrdIdle
		return
	}
	switch g.approach(u, b.rect(), Reach) {
	case Arrived:
		p := &g.players[u.Team]
		p.Res[u.CarryKind] += u.Carry
		p.Stats.Gathered[u.CarryKind] += float64(u.Carry)
		u.Carry = 0
		res, farm := u.Res, u.FarmB
		u.reset(OrdGather)
		u.Res, u.FarmB = res, farm
		if farm == nil && (res == nil || res.Amount <= 0) && u.LastRes == (pt{}) {
			u.Order = OrdIdle
		}
	case Unreachable:
		u.Order = OrdIdle
	}
}

func (g *Game) updateBuild(u *Unit) {
	b := u.BuildB
	if b == nil || b.dead || b.Built {
		u.Order = OrdIdle
		for _, o := range g.buildings {
			if o.Team == u.Team && !o.Built && !o.dead && distToRect(u.X, u.Y, o.rect()) < 10*TileSize {
				g.orderBuild(u, o)
				return
			}
		}
		if b != nil && b.Built && !b.dead {
			g.afterBuild(u, b)
		}
		return
	}
	switch g.approach(u, b.rect(), Reach) {
	case Arrived:
		u.acting = true
		bx, by := b.center()
		u.Facing = math.Atan2(by-u.Y, bx-u.X)
		def := b.def()
		inc := dt / def.BuildTime
		b.Progress += inc
		b.HP += def.HP * inc
		if b.Progress >= 1 {
			b.Progress, b.Built = 1, true
			b.HP = min(b.HP, def.HP)
			g.players[b.Team].Stats.BuildValue += float64(buildValue(b.Kind))
			g.afterBuild(u, b)
		}
	case Unreachable:
		u.Order = OrdIdle
	}
}

// afterBuild sends a builder to work at what it just built, the way AoE
// villagers start farming a farm or gathering next to a new camp.
func (g *Game) afterBuild(u *Unit, b *Building) {
	switch b.Kind {
	case Farm:
		if b.farmer == nil || b.farmer.dead || b.farmer.FarmB != b {
			g.orderFarm(u, b)
		}
	case LumberCamp, MiningCamp, Mill:
		kinds := map[BldKind][]ResKind{LumberCamp: {Wood}, MiningCamp: {Gold, Stone}, Mill: {Food}}[b.Kind]
		for _, k := range kinds {
			if r := g.findReachableResource(toTile(u.X), toTile(u.Y), kindIs(k), b.TX, b.TY, 8); r != nil {
				g.orderGather(u, r)
				return
			}
		}
	}
}

func (g *Game) attackRange(u *Unit, e *Unit) float64 {
	r := g.rangeOf(u)
	if u.Kind == Villager && e != nil && e.def().Animal {
		r = 3 * TileSize
	}
	return r
}

// Projectile kinds for effects.
const (
	ShotArrow = iota
	ShotStone
	ShotBoulder
)

func shotOf(u *Unit) int {
	switch u.Kind {
	case Mangonel:
		return ShotStone
	case Trebuchet:
		return ShotBoulder
	}
	return ShotArrow
}

// fire launches u's attack at a point. Ranged attacks show a projectile.
// Splash attacks also hit enemies near the point.
func (g *Game) fire(u *Unit, tx, ty float64, target *Unit, bld *Building) {
	def := u.def()
	ranged := def.Range > 0
	if ranged {
		g.effects = append(g.effects, Effect{u.X, u.Y, tx, ty, arrowTime + def.Range/TileSize*0.03, 12, shotOf(u), arrowTime + def.Range/TileSize*0.03})
	}
	atk := g.attackOf(u)
	pierce := def.Archer
	if target != nil {
		g.damageUnit(target, atk, pierce, u.Team, u)
	}
	if bld != nil {
		amt := atk + def.BonusVsBld
		if def.Archer {
			amt *= 0.25
		}
		g.damageBld(bld, amt, u.Team)
	}
	if def.Splash > 0 {
		g.eachNear(tx, ty, def.Splash, func(o *Unit) {
			if o != target && o.Team != u.Team && o.Team != Gaia && o.Inside == nil && math.Hypot(o.X-tx, o.Y-ty) <= def.Splash {
				g.damageUnit(o, atk*0.5, false, u.Team, u)
			}
		})
	}
}

func (g *Game) updateAttack(u *Unit) {
	def := u.def()
	if e := u.Target; e != nil {
		if e.dead || e.HP <= 0 || e.Inside != nil {
			u.Target = nil
			if u.Kind == Villager && e.carcass != nil && e.carcass.Amount > 0 {
				g.orderGather(u, e.carcass)
				return
			}
			if def.Military {
				if n := g.nearestEnemyUnit(u.Team, u.X, u.Y, def.Sight); n != nil {
					u.Target = n
					return
				}
			}
			if u.TargetB == nil {
				u.Order = OrdIdle
			}
			return
		}
		d := math.Hypot(e.X-u.X, e.Y-u.Y)
		rng := g.attackRange(u, e)
		switch {
		case def.MinRange > 0 && d < def.MinRange:
			// Too close to fire: back away from the target.
			u.Path = nil
			g.moveToward(u, u.X+(u.X-e.X), u.Y+(u.Y-e.Y))
		case d <= def.Radius+e.def().Radius+5+rng:
			u.Path = nil
			u.acting = true
			u.Facing = math.Atan2(e.Y-u.Y, e.X-u.X)
			if u.cool <= 0 {
				u.cool = def.Cooldown
				g.fire(u, e.X, e.Y, e, nil)
			}
		case d < 2.5*TileSize:
			u.Path = nil
			g.moveToward(u, e.X, e.Y)
		default:
			et := pt{toTile(e.X), toTile(e.Y)}
			if (len(u.Path) == 0 || et != u.chase) && u.repath <= 0 {
				u.chase = et
				u.Path, _ = g.findPath(toTile(u.X), toTile(u.Y), tileRect(et.X, et.Y), 1)
				u.repath = 0.6
			}
			if len(u.Path) > 0 {
				g.followPath(u)
			} else {
				g.moveToward(u, e.X, e.Y)
			}
		}
		return
	}
	b := u.TargetB
	if b == nil || b.dead {
		u.Order = OrdIdle
		return
	}
	if def.Military && !def.OnlyBuildings {
		if u.scan -= dt; u.scan <= 0 {
			u.scan = 0.5
			if n := g.nearestEnemyUnit(u.Team, u.X, u.Y, 4*TileSize); n != nil {
				u.Target, u.Path = n, nil
				return
			}
		}
	}
	switch g.approach(u, b.rect(), Reach+g.rangeOf(u)) {
	case Arrived:
		u.acting = true
		bx, by := b.center()
		u.Facing = math.Atan2(by-u.Y, bx-u.X)
		if u.cool <= 0 {
			u.cool = def.Cooldown
			g.fire(u, bx, by, nil, b)
		}
	case Unreachable:
		u.Order = OrdIdle
	}
}

func (g *Game) nearestEnemyUnit(team int, x, y, radius float64) *Unit {
	var best *Unit
	bd := radius
	g.eachNear(x, y, radius, func(e *Unit) {
		if e.Team != team && e.Team != Gaia && !e.dead && e.HP > 0 && e.Inside == nil {
			if d := math.Hypot(e.X-x, e.Y-y); d <= bd {
				best, bd = e, d
			}
		}
	})
	return best
}

func (g *Game) nearestEnemyBuilding(team int, x, y, radius float64) *Building {
	var best *Building
	bd := radius
	for _, b := range g.buildings {
		if b.Team != team && b.Team != Gaia && !b.dead {
			if d := distToRect(x, y, b.rect()); d <= bd {
				best, bd = b, d
			}
		}
	}
	return best
}

// damageUnit applies an attack to e from team, and by when a unit made it.
func (g *Game) damageUnit(e *Unit, atk float64, pierce bool, team int, by *Unit) {
	ed := e.def()
	d := atk - g.armorOf(e, pierce)
	if by != nil && ed.Cavalry {
		d += g.bonusVsCav(by)
	}
	e.HP -= max(1, d)
	if e.HP <= 0 {
		if !e.killed {
			e.killed = true
			if team < Gaia && e.Team < Gaia && team != e.Team {
				s := &g.players[team].Stats
				s.Kills++
				s.KillValue += float64(ed.Cost.total())
				g.players[e.Team].Stats.Losses++
			}
		}
		return
	}
	if e.Team < Gaia && team < Gaia {
		g.players[e.Team].alertT = 3
		g.players[e.Team].alertX, g.players[e.Team].alertY = e.X, e.Y
	}
	if by == nil {
		return
	}
	switch {
	case e.Kind == Deer:
		dx, dy := e.X-by.X, e.Y-by.Y
		l := math.Max(1, math.Hypot(dx, dy))
		g.orderMove(e, e.X+dx/l*96, e.Y+dy/l*96)
	case e.Kind == Boar && e.Order != OrdAttack:
		g.orderAttackUnit(e, by)
	case ed.Military && !ed.OnlyBuildings && ed.Heal == 0 && e.Order == OrdIdle:
		g.orderAttackUnit(e, by)
	}
}

func (g *Game) damageBld(b *Building, amt float64, team int) {
	if b.dead || b.HP <= 0 {
		return
	}
	b.HP -= amt
	if b.Team < Gaia {
		cx, cy := b.center()
		p := &g.players[b.Team]
		p.alertT, p.alertX, p.alertY = 3, cx, cy
	}
	if b.HP <= 0 && !b.razed && team < Gaia && team != b.Team {
		b.razed = true
		s := &g.players[team].Stats
		s.Razed++
		s.RazeValue += float64(buildValue(b.Kind))
	}
}

// updateHeal moves a monk to the most hurt friendly unit nearby and heals it.
func (g *Game) updateHeal(u *Unit) {
	def := u.def()
	h := u.healing
	if h == nil || h.dead || h.HP >= h.MaxHP || h.Inside != nil || math.Hypot(h.X-u.X, h.Y-u.Y) > def.Sight {
		h, u.healing = nil, nil
		worst := 1.0
		g.eachNear(u.X, u.Y, def.Sight, func(o *Unit) {
			if o != u && o.Team == u.Team && !o.dead && o.Inside == nil && o.HP < o.MaxHP {
				if f := o.HP / o.MaxHP; f < worst {
					h, worst = o, f
				}
			}
		})
		u.healing = h
	}
	if h == nil {
		if u.Order == OrdMove {
			u.Order = OrdIdle
		}
		return
	}
	if math.Hypot(h.X-u.X, h.Y-u.Y) > 2.5*TileSize {
		g.moveToward(u, h.X, h.Y)
		return
	}
	u.acting = true
	u.Facing = math.Atan2(h.Y-u.Y, h.X-u.X)
	h.HP = min(h.MaxHP, h.HP+def.Heal*dt)
}

// Garrisoning: villagers shelter inside Town Centers, towers and castles.
// Each one inside adds an arrow.

func (g *Game) garrisoned(b *Building) int {
	n := 0
	for _, u := range g.units {
		if u.Inside == b {
			n++
		}
	}
	return n
}

func (g *Game) orderGarrison(u *Unit, b *Building) {
	res := u.Res
	u.reset(OrdGarrison)
	u.Res, u.Shelter = res, b
}

func (g *Game) updateGarrison(u *Unit) {
	b := u.Shelter
	if b == nil || b.dead || !b.Built || g.garrisoned(b) >= b.def().Garrison {
		u.Shelter, u.Order = nil, OrdIdle
		return
	}
	switch g.approach(u, b.rect(), Reach) {
	case Arrived:
		u.Inside, u.Shelter, u.Selected = b, nil, false
		u.Order, u.Path = OrdIdle, nil
	case Unreachable:
		u.Order = OrdIdle
	}
}

// ungarrison puts everyone inside b back outside, back to work.
func (g *Game) ungarrison(b *Building) {
	for _, u := range g.units {
		if u.Inside != b {
			continue
		}
		u.Inside = nil
		u.X, u.Y = g.spawnSpot(b.rect())
		if r := u.Res; r != nil && r.Amount > 0 {
			g.orderGather(u, r)
		}
	}
}

// ringBell sends the team's villagers near b into the nearest shelters.
func (g *Game) ringBell(team int) {
	for _, u := range g.units {
		if u.Team != team || u.Kind != Villager || u.Inside != nil {
			continue
		}
		var best *Building
		bd := 30.0 * TileSize
		for _, b := range g.buildings {
			if b.Team == team && b.Built && b.def().Garrison > 0 {
				if d := distToRect(u.X, u.Y, b.rect()); d < bd {
					best, bd = b, d
				}
			}
		}
		if best != nil {
			g.orderGarrison(u, best)
		}
	}
}

func (g *Game) ungarrisonAll(team int) {
	for _, b := range g.buildings {
		if b.Team == team && b.def().Garrison > 0 {
			g.ungarrison(b)
		}
	}
}

func (g *Game) updateBuilding(b *Building) {
	if !b.Built || b.dead {
		return
	}
	def := b.def()
	if b.housedMsg > 0 {
		b.housedMsg -= dt
	}
	if len(b.Queue) > 0 {
		q := b.Queue[0]
		if b.TrainT < q.time() {
			b.TrainT += dt
		}
		if b.TrainT >= q.time() {
			p := &g.players[b.Team]
			switch {
			case q.Age > 0:
				p.Age = q.Age
				b.Queue, b.TrainT = b.Queue[1:], 0
				if b.Team == g.human {
					g.flash("You have advanced to the " + ageNames[q.Age] + "!")
				}
			case q.Tech > 0:
				g.research(b.Team, q.Tech-1)
				b.Queue, b.TrainT = b.Queue[1:], 0
				if b.Team == g.human {
					g.flash(techDefs[q.Tech-1].Name + " researched")
				}
			case g.pop(b.Team) < g.popCap(b.Team):
				g.spawnUnit(q.Unit, b.Team, b)
				if unitDefs[q.Unit].Military {
					p.Stats.ArmyValue += float64(unitDefs[q.Unit].Cost.total())
				}
				b.Queue, b.TrainT = b.Queue[1:], 0
			case b.Team == g.human && b.housedMsg <= 0:
				b.housedMsg = 5
				g.flash("Need more houses!")
			}
		}
	}
	if def.Arrows > 0 {
		if b.cool > 0 {
			b.cool -= dt
		}
		if b.cool <= 0 {
			cx, cy := b.center()
			bon := g.bonus(b.Team)
			rng := def.Range + bon.ArcherRange
			shot := 0
			for range def.Arrows + g.garrisoned(b) {
				e := g.nearestEnemyUnit(b.Team, cx, cy, rng)
				if e == nil {
					break
				}
				g.damageUnit(e, def.Attack+bon.ArcherAtk, true, b.Team, nil)
				g.effects = append(g.effects, Effect{cx, cy, e.X, e.Y, arrowTime, 70, ShotArrow, arrowTime})
				shot++
			}
			if shot > 0 {
				b.cool = def.Cooldown
			}
		}
	}
}

// separate pushes overlapping units apart.
func (g *Game) separate() {
	for _, a := range g.units {
		if a.Inside != nil {
			continue
		}
		g.eachNear(a.X, a.Y, 24, func(b *Unit) {
			if b == a || b.X < a.X || (b.X == a.X && b.Y <= a.Y) {
				return
			}
			dx, dy := b.X-a.X, b.Y-a.Y
			minD := a.def().Radius + b.def().Radius
			d2 := dx*dx + dy*dy
			if d2 >= minD*minD {
				return
			}
			d := math.Sqrt(d2)
			if d < 0.01 {
				dx, dy = g.rng.Float64()-0.5, g.rng.Float64()-0.5
				d = math.Hypot(dx, dy)
			}
			push := (minD - d) / 4
			nx, ny := dx/d*push, dy/d*push
			// Moving units also step sideways, so two units walking into each
			// other slide past instead of pushing each other straight back.
			ax, ay, bx, by := -nx, -ny, nx, ny
			ux, uy := dx/d, dy/d
			if a.moving() && math.Cos(a.Facing)*ux+math.Sin(a.Facing)*uy > 0.7 {
				ax, ay = ax+ny, ay-nx
			}
			if b.moving() && math.Cos(b.Facing)*-ux+math.Sin(b.Facing)*-uy > 0.7 {
				bx, by = bx-ny, by+nx
			}
			g.tryMove(a, ax, ay)
			g.tryMove(b, bx, by)
		})
	}
}

// placeCarcass leaves food where an animal died.
func (g *Game) placeCarcass(u *Unit) {
	cx, cy := toTile(u.X), toTile(u.Y)
	variant := uint8(FoodDeer)
	if u.Kind == Boar {
		variant = FoodBoar
	}
	for d := 0; d < 4; d++ {
		for y := cy - d; y <= cy+d; y++ {
			for x := cx - d; x <= cx+d; x++ {
				if g.blocked(x, y) || g.tile(x, y).Bld != nil {
					continue
				}
				r := &Resource{Kind: Food, Variant: variant, TX: x, TY: y, Amount: u.def().Food}
				g.tile(x, y).Res = r
				u.carcass = r
				return
			}
		}
	}
}

func (g *Game) cleanup() {
	alive := g.units[:0]
	for _, u := range g.units {
		if u.HP > 0 {
			alive = append(alive, u)
			continue
		}
		u.dead, u.Selected = true, false
		if u.FarmB != nil && u.FarmB.farmer == u {
			u.FarmB.farmer = nil
		}
		if u.def().Animal {
			g.placeCarcass(u)
		} else if g.visual {
			g.corpses = append(g.corpses, Corpse{u.Kind, u.Team, u.X, u.Y, u.Facing, corpseTime})
		}
	}
	clear(g.units[len(alive):])
	g.units = alive

	bs := g.buildings[:0]
	for _, b := range g.buildings {
		if b.HP > 0 {
			bs = append(bs, b)
			continue
		}
		b.dead = true
		r := b.rect()
		for y := r.y0; y <= r.y1; y++ {
			for x := r.x0; x <= r.x1; x++ {
				g.tile(x, y).Bld = nil
			}
		}
		g.ungarrison(b)
		if g.selBld == b {
			g.selBld = nil
		}
		if b.Kind == TownCenter && g.winner < 0 && !g.hasTC(b.Team) {
			g.winner = 1 - b.Team
		}
	}
	clear(g.buildings[len(bs):])
	g.buildings = bs
}

func (g *Game) hasTC(team int) bool {
	for _, b := range g.buildings {
		if b.Team == team && b.Kind == TownCenter && !b.dead && b.HP > 0 {
			return true
		}
	}
	return false
}
