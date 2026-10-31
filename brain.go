package main

import (
	"math"
	"math/rand"
)

// Macro actions. A Policy picks one every decisionTime seconds and the
// Brain turns it into unit orders.
const (
	ANoop = iota
	AVillager
	AHouse
	AMill
	ALumber
	AMine
	AFarm
	ABarracks
	ARange
	AStable
	ABlacksmith
	ATower
	ASiegeWorkshop
	AMonastery
	ACastle
	AAgeUp
	AInfantry
	ASpear
	AArcher
	ACavalry
	ACavArcher
	ALongbow
	AMonk
	ARam
	AMangonel
	ATrebuchet
	AResearchEco
	AResearchMil
	AEcoFood
	AEcoWood
	AEcoGold
	AEcoStone
	AAttack
	ARetreat
	numActions
)

var actionNames = [numActions]string{
	"Wait", "Villager", "House", "Mill", "Lumber Camp", "Mining Camp", "Farm", "Barracks",
	"Archery Range", "Stable", "Blacksmith", "Tower", "Siege Workshop", "Monastery", "Castle", "Age Up",
	"Infantry", "Spearman", "Archer", "Cavalry", "Cavalry Archer", "Longbowman", "Monk",
	"Battering Ram", "Mangonel", "Trebuchet", "Economy Tech", "Military Tech",
	"More Food", "More Wood", "More Gold", "More Stone", "Attack", "Retreat",
}

var actionBuilds = map[int]BldKind{
	AHouse: House, AMill: Mill, ALumber: LumberCamp, AMine: MiningCamp, AFarm: Farm,
	ABarracks: Barracks, ARange: ArcheryRange, AStable: Stable, ABlacksmith: Blacksmith, ATower: Tower,
	ASiegeWorkshop: SiegeWorkshop, AMonastery: Monastery, ACastle: Castle,
}

var actionUnits = map[int]UnitKind{
	AVillager: Villager, AInfantry: Militia, ASpear: Spearman, AArcher: Archer, ACavArcher: CavalryArcher,
	ALongbow: Longbowman, AMonk: Monk, ARam: Ram, AMangonel: Mangonel, ATrebuchet: Trebuchet,
}

// Technologies the AI researches, in order of preference.
var (
	ecoTechs = []Tech{Loom, DoubleBitAxe, HorseCollar, GoldMining, Wheelbarrow, BowSaw, HeavyPlow, StoneMining, HandCart}
	milTechs = []Tech{Forging, Fletching, ManAtArms, ScaleMail, PaddedArmor, ScaleBarding, Crossbowman, LightCavalry,
		Pikeman, LongSwordsman, IronCasting, BodkinArrow, ChainMail, ChainBarding, Arbalest, Cavalier, ChampionUp, BlastFurnace}
)

const (
	decisionTime = 3.0
	upkeepTime   = 1.0
	numObs       = 96
)

// Policy chooses a macro action. logp and value are only used by learning
// policies.
type Policy interface {
	Choose(g *Game, team int, obs []float64, mask []bool) (action int, logp, value float64)
	Name() string
}

// Step is one recorded decision, for training.
type Step struct {
	Obs    []float64
	Mask   []bool
	Action int
	LogP   float64
	Value  float64
	Reward float64
}

type Brain struct {
	team      int
	policy    Policy
	record    bool
	steps     []Step
	eco       [numRes]float64
	decideT   float64
	upkeepT   float64
	prevDiff  float64
	attacking bool
	belled    bool
	calmT     float64
	attacks   int
	target    *Building
	counts    [numActions]int
	last      int
	lastProbs []float64
}

func newBrain(team int, p Policy, record bool) *Brain {
	return &Brain{team: team, policy: p, record: record, eco: [numRes]float64{0.5, 0.4, 0.1, 0}, decideT: 0.5}
}

func (g *Game) updateBrains() {
	for _, b := range g.brains {
		if b == nil {
			continue
		}
		if b.upkeepT -= dt; b.upkeepT <= 0 {
			b.upkeepT = upkeepTime
			b.upkeep(g)
		}
		if b.decideT -= dt; b.decideT <= 0 {
			b.decideT = decisionTime
			b.decide(g)
		}
	}
}

func (b *Brain) scoreDiff(g *Game) float64 {
	return g.score(b.team).Total - g.score(1-b.team).Total
}

func (b *Brain) decide(g *Game) {
	diff := b.scoreDiff(g)
	if n := len(b.steps); n > 0 {
		b.steps[n-1].Reward += (diff - b.prevDiff) / 200
	}
	b.prevDiff = diff
	obs := g.observe(b)
	mask := g.actionMask(b)
	a, logp, v := b.policy.Choose(g, b.team, obs, mask)
	if a < 0 || a >= numActions || !mask[a] {
		a = ANoop
	}
	if b.record {
		b.steps = append(b.steps, Step{Obs: obs, Mask: mask, Action: a, LogP: logp, Value: v})
	}
	b.counts[a]++
	b.last = a
	g.execute(b, a)
}

// finish adds the terminal reward once the game is over.
func (b *Brain) finish(g *Game) {
	if len(b.steps) == 0 {
		return
	}
	r := (b.scoreDiff(g) - b.prevDiff) / 200
	switch g.winner {
	case b.team:
		r += 5
	case 1 - b.team:
		r -= 5
	}
	b.steps[len(b.steps)-1].Reward += r
}

// State helpers.

type teamInfo struct {
	vills, idle, builders int
	gatherers             [numRes]int
	army                  [numUnitKinds]int
	armyTotal             int
	built, all            [numBldKinds]int
	tcQueue               int
	garrisoned            int
}

func (g *Game) info(team int) teamInfo {
	var t teamInfo
	for _, u := range g.units {
		if u.Team != team {
			continue
		}
		switch {
		case u.Inside != nil:
			t.vills++
			t.garrisoned++
		case u.Kind == Villager:
			t.vills++
			switch u.Order {
			case OrdIdle, OrdMove:
				t.idle++
			case OrdBuild:
				t.builders++
			case OrdGather, OrdReturn:
				t.gatherers[u.GatherKnd]++
			case OrdAttack:
				t.gatherers[Food]++
			}
		case u.def().Military:
			t.army[u.Kind]++
			t.armyTotal++
		}
	}
	for _, b := range g.buildings {
		if b.Team != team {
			continue
		}
		t.all[b.Kind]++
		if b.Built {
			t.built[b.Kind]++
		}
		if b.Kind == TownCenter {
			t.tcQueue += len(b.Queue)
		}
	}
	return t
}

func (g *Game) homeOf(team int) pt {
	if tc := g.teamTC(team); tc != nil {
		return pt{tc.TX + 1, tc.TY + 1}
	}
	return g.homes[team]
}

func (g *Game) unitsNear(team int, p pt, radius float64, military bool) int {
	n := 0
	x, y := tileCenter(p.X), tileCenter(p.Y)
	g.eachNear(x, y, radius, func(u *Unit) {
		if u.Team == team && (!military || u.def().Military) && math.Hypot(u.X-x, u.Y-y) <= radius {
			n++
		}
	})
	return n
}

func (g *Game) resourceNear(p pt, k ResKind, radius int) int {
	total := 0
	for y := max(p.Y-radius, 0); y <= min(p.Y+radius, MapH-1); y++ {
		for x := max(p.X-radius, 0); x <= min(p.X+radius, MapW-1); x++ {
			if r := g.tile(x, y).Res; r != nil && r.Kind == k {
				total += r.Amount
			}
		}
	}
	return total
}

func clamp01(v float64) float64 { return min(max(v, 0), 1) }

// observe encodes the game state from the brain's side as numObs floats.
func (g *Game) observe(b *Brain) []float64 {
	me, en := b.team, 1-b.team
	p, e := &g.players[me], &g.players[en]
	mi, ei := g.info(me), g.info(en)
	home, enemyHome := g.homeOf(me), g.homeOf(en)
	o := make([]float64, 0, numObs)
	for k := range numRes {
		o = append(o, math.Log1p(float64(p.Res[k]))/7)
	}
	pop, cap := g.pop(me), g.popCap(me)
	o = append(o, float64(pop)/MaxPop, float64(cap)/MaxPop, clamp01(float64(cap-pop)/10), float64(mi.vills)/60, float64(mi.idle)/10, float64(mi.builders)/5)
	for k := range numRes {
		o = append(o, float64(mi.gatherers[k])/30, b.eco[k])
	}
	armyKinds := []UnitKind{Militia, Spearman, Archer, Scout, Knight, CavalryArcher, Longbowman, Monk, Ram, Mangonel, Trebuchet}
	for _, k := range armyKinds {
		o = append(o, float64(mi.army[k])/15)
	}
	for k := range numBldKinds {
		if k != TownCenter && !bldDefs[k].Wall {
			o = append(o, float64(mi.all[k])/5)
		}
	}
	for a := range NumAges {
		o = append(o, boolf(p.Age == a))
	}
	o = append(o, boolf(g.agingUp(me)), float64(mi.tcQueue)/5, g.time/1500)
	tc, etc := g.teamTC(me), g.teamTC(en)
	o = append(o, hpFrac(tc), hpFrac(etc))
	o = append(o, float64(ei.vills)/60, float64(ei.armyTotal)/40, float64(e.Age)/3)
	o = append(o, float64(g.unitsNear(en, home, 14*TileSize, true))/20, float64(g.unitsNear(me, enemyHome, 14*TileSize, true))/20)
	o = append(o, float64(mi.armyTotal)/40, boolf(b.attacking))
	o = append(o, math.Tanh(b.scoreDiff(g)/1000))
	o = append(o, float64(g.resourceNear(home, Food, 12))/2000, float64(g.resourceNear(home, Wood, 12))/5000,
		float64(g.resourceNear(home, Gold, 14))/2000, float64(g.resourceNear(home, Stone, 14))/2000)
	o = append(o, float64(ei.army[Scout]+ei.army[Knight]+ei.army[CavalryArcher])/15,
		float64(ei.army[Archer]+ei.army[Longbowman])/15, float64(ei.army[Militia]+ei.army[Spearman])/15,
		float64(ei.army[Ram]+ei.army[Mangonel]+ei.army[Trebuchet])/5)
	eco, mil := 0, 0
	for _, t := range ecoTechs {
		eco += int(boolf(p.Researched[t]))
	}
	for _, t := range milTechs {
		mil += int(boolf(p.Researched[t]))
	}
	o = append(o, float64(eco)/float64(len(ecoTechs)), float64(mil)/float64(len(milTechs)))
	for _, k := range []UnitKind{Militia, Spearman, Archer, Scout, Knight} {
		o = append(o, float64(p.Tier[k])/3)
	}
	o = append(o, float64(mi.garrisoned)/15, boolf(b.belled))
	for len(o) < numObs {
		o = append(o, 0)
	}
	return o[:numObs]
}

func boolf(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func hpFrac(b *Building) float64 {
	if b == nil {
		return 0
	}
	return b.HP / b.def().HP
}

// trainerFor returns the team's built building with the shortest queue that
// trains k.
func (g *Game) trainerFor(team int, k UnitKind) *Building {
	var best *Building
	for _, b := range g.buildings {
		if b.Team == team && b.Built && b.trains(k) && (best == nil || len(b.Queue) < len(best.Queue)) {
			best = b
		}
	}
	return best
}

// researcherFor returns the team's built building that researches t.
func (g *Game) researcherFor(team int, t Tech) *Building {
	var best *Building
	for _, b := range g.buildings {
		if b.Team == team && b.Built && b.Kind == techDefs[t].At && (best == nil || len(b.Queue) < len(best.Queue)) {
			best = b
		}
	}
	return best
}

// nextTech returns the first technology in list the team can research and
// afford right now, with the building to research it in.
func (g *Game) nextTech(team int, list []Tech) (Tech, *Building) {
	for _, t := range list {
		if ok, _ := g.techAvailable(team, t); !ok || !g.canAfford(team, techDefs[t].Cost) {
			continue
		}
		if b := g.researcherFor(team, t); b != nil && len(b.Queue) < 2 {
			return t, b
		}
	}
	return NoTech, nil
}

func (b *Brain) actionUnit(g *Game, a int) (UnitKind, bool) {
	if a == ACavalry {
		if g.players[b.team].Age >= 2 {
			return Knight, true
		}
		return Scout, true
	}
	k, ok := actionUnits[a]
	return k, ok
}

func (g *Game) actionMask(b *Brain) []bool {
	m := make([]bool, numActions)
	team := b.team
	info := g.info(team)
	m[ANoop] = true
	for a := range numActions {
		if k, ok := actionBuilds[a]; ok {
			avail, _ := g.bldAvailable(team, k)
			m[a] = avail && info.vills > 0 && g.canAfford(team, bldDefs[k].Cost) && info.all[k]-info.built[k] < 2
			continue
		}
		if k, ok := b.actionUnit(g, a); ok {
			avail, _ := g.unitAvailable(team, k)
			t := g.trainerFor(team, k)
			m[a] = avail && t != nil && len(t.Queue) < 3 && g.canAfford(team, unitDefs[k].Cost)
		}
	}
	if ok, _ := g.ageUpAvailable(team); ok {
		m[AAgeUp] = g.canAfford(team, ageCost[g.players[team].Age+1])
	}
	t, _ := g.nextTech(team, ecoTechs)
	m[AResearchEco] = t != NoTech
	t, _ = g.nextTech(team, milTechs)
	m[AResearchMil] = t != NoTech
	for k := range numRes {
		m[AEcoFood+k] = info.vills > 0 && b.eco[k] < 0.8
	}
	m[AAttack] = info.armyTotal > 0
	m[ARetreat] = b.attacking
	return m
}

// Executing actions.

func (g *Game) execute(b *Brain, a int) {
	team := b.team
	switch {
	case a == ANoop:
	case a == AAgeUp:
		if tc := g.teamTC(team); tc != nil {
			g.enqueue(tc, QueueItem{Age: g.players[team].Age + 1})
		}
	case a == AResearchEco || a == AResearchMil:
		list := ecoTechs
		if a == AResearchMil {
			list = milTechs
		}
		if t, at := g.nextTech(team, list); at != nil {
			g.enqueue(at, QueueItem{Tech: t + 1})
		}
	case a >= AEcoFood && a <= AEcoStone:
		b.shiftEco(g, ResKind(a-AEcoFood))
	case a == AAttack:
		b.attacking = true
		b.attacks++
		b.target = nil
		b.sendArmy(g)
	case a == ARetreat:
		b.attacking = false
		home := g.homeOf(team)
		for _, u := range g.units {
			if u.Team == team && u.def().Military {
				g.orderMove(u, tileCenter(home.X+2), tileCenter(home.Y+2))
			}
		}
	default:
		if k, ok := actionBuilds[a]; ok {
			b.build(g, k)
		} else if k, ok := b.actionUnit(g, a); ok {
			if t := g.trainerFor(team, k); t != nil {
				g.enqueue(t, QueueItem{Unit: k})
			}
		}
	}
}

func (b *Brain) shiftEco(g *Game, k ResKind) {
	b.eco[k] += 0.15
	sum := 0.0
	for _, v := range b.eco {
		sum += v
	}
	for i := range b.eco {
		b.eco[i] /= sum
	}
	// Move a couple of villagers from the most over-staffed resource.
	info := g.info(b.team)
	over, worst := ResKind(-1), 0.0
	for r := range numRes {
		if d := float64(info.gatherers[r]) - b.eco[r]*float64(info.vills); ResKind(r) != k && d > worst {
			over, worst = ResKind(r), d
		}
	}
	if over < 0 {
		return
	}
	moved := 0
	for _, u := range g.units {
		if moved >= 2 {
			break
		}
		if u.Team == b.team && u.Kind == Villager && u.GatherKnd == over && (u.Order == OrdGather || u.Order == OrdReturn) {
			b.assign(g, u, k)
			moved++
		}
	}
}

// sendArmy orders every military unit to attack the enemy building nearest
// to the army.
func (b *Brain) sendArmy(g *Game) {
	var army []*Unit
	cx, cy := 0.0, 0.0
	for _, u := range g.units {
		if u.Team == b.team && u.def().Military {
			army = append(army, u)
			cx += u.X
			cy += u.Y
		}
	}
	if len(army) == 0 {
		return
	}
	cx, cy = cx/float64(len(army)), cy/float64(len(army))
	if b.target == nil || b.target.dead {
		b.target = nil
		bd := math.Inf(1)
		for _, o := range g.buildings {
			if o.Team != 1-b.team || o.Kind == Farm {
				continue
			}
			if d := distToRect(cx, cy, o.rect()); d < bd {
				b.target, bd = o, d
			}
		}
	}
	if b.target == nil {
		return
	}
	for _, u := range army {
		if u.def().Heal > 0 {
			// Monks walk with the army and heal whoever is hurt.
			if u.healing == nil && math.Hypot(u.X-cx, u.Y-cy) > 3*TileSize {
				g.orderMove(u, cx, cy)
			}
			continue
		}
		if u.Order == OrdIdle || u.Order == OrdMove || (u.Order == OrdAttack && u.Target == nil && u.TargetB != b.target) {
			g.orderAttackBld(u, b.target)
		}
	}
}

// pickWorker returns an idle villager, or else the nearest wood or food
// gatherer to tile p.
func (g *Game) pickWorker(team int, p pt) *Unit {
	var best *Unit
	bd := math.Inf(1)
	x, y := tileCenter(p.X), tileCenter(p.Y)
	for _, u := range g.units {
		if u.Team != team || u.Kind != Villager || u.Order == OrdBuild || u.Order == OrdAttack {
			continue
		}
		d := math.Hypot(u.X-x, u.Y-y)
		if u.Order != OrdIdle {
			if u.GatherKnd == Gold || u.GatherKnd == Stone || u.FarmB != nil {
				continue
			}
			d += 2000
		}
		if d < bd {
			best, bd = u, d
		}
	}
	return best
}

// siteFree reports whether k fits at (tx, ty) with a free ring around it so
// buildings never wall off paths. Farms may touch other buildings.
func (g *Game) siteFree(k BldKind, team, tx, ty int) bool {
	s := bldDefs[k].Size
	m := 1
	if k == Farm {
		m = 0
	}
	for y := ty - m; y < ty+s+m; y++ {
		for x := tx - m; x < tx+s+m; x++ {
			inside := x >= tx && x < tx+s && y >= ty && y < ty+s
			if inside {
				if g.blocked(x, y) || g.buildingAtTile(x, y) != nil || g.unitOnTile(x, y, 1-team) {
					return false
				}
			} else if b := g.buildingAtTile(x, y); b != nil && b.Kind != Farm {
				return false
			} else if !inBounds(x, y) || g.tile(x, y).Ter == TerWater {
				return false
			}
		}
	}
	return true
}

// findSite searches outward from anchor for a place to put k.
func (g *Game) findSite(k BldKind, team int, anchor pt, radius int) (int, int, bool) {
	s := bldDefs[k].Size
	for d := 0; d <= radius; d++ {
		for dy := -d; dy <= d; dy++ {
			for dx := -d; dx <= d; dx++ {
				if max(abs(dx), abs(dy)) != d {
					continue
				}
				tx, ty := anchor.X+dx-s/2, anchor.Y+dy-s/2
				if g.siteFree(k, team, tx, ty) {
					return tx, ty, true
				}
			}
		}
	}
	return 0, 0, false
}

// nearCamp reports whether a drop site of kind k is within 6 tiles of p.
func (g *Game) nearCamp(team int, k BldKind, p pt) bool {
	for _, b := range g.buildings {
		if b.Team == team && (b.Kind == k || b.Kind == TownCenter) {
			cx, cy := b.center()
			if math.Hypot(cx-tileCenter(p.X), cy-tileCenter(p.Y)) < 6*TileSize {
				return true
			}
		}
	}
	return false
}

func (b *Brain) siteAnchor(g *Game, k BldKind) pt {
	home := g.homeOf(b.team)
	enemy := g.homeOf(1 - b.team)
	toward := func(n int) pt {
		dx, dy := enemy.X-home.X, enemy.Y-home.Y
		l := math.Max(1, math.Hypot(float64(dx), float64(dy)))
		return pt{home.X + int(float64(dx)/l*float64(n)), home.Y + int(float64(dy)/l*float64(n))}
	}
	resAnchor := func(want func(*Resource) bool, camp BldKind) pt {
		r := g.findReachableResource(home.X, home.Y, func(r *Resource) bool {
			return want(r) && !g.nearCamp(b.team, camp, pt{r.TX, r.TY})
		}, home.X, home.Y, 30)
		if r == nil {
			return home
		}
		return pt{r.TX, r.TY}
	}
	switch k {
	case Mill:
		return resAnchor(func(r *Resource) bool { return r.Kind == Food && r.Variant == FoodBerries }, Mill)
	case LumberCamp:
		return resAnchor(kindIs(Wood), LumberCamp)
	case MiningCamp:
		want := Gold
		if b.eco[Stone] > b.eco[Gold] {
			want = Stone
		}
		return resAnchor(kindIs(want), MiningCamp)
	case Farm:
		for _, o := range g.buildings {
			if o.Team == b.team && o.Kind == Mill && o.Built {
				return pt{o.TX + 1, o.TY + 1}
			}
		}
		return home
	case Barracks, ArcheryRange, Stable, Castle:
		return toward(7)
	case Tower:
		return toward(5)
	}
	return home
}

func (b *Brain) build(g *Game, k BldKind) {
	anchor := b.siteAnchor(g, k)
	tx, ty, ok := g.findSite(k, b.team, anchor, 14)
	if !ok {
		return
	}
	w := g.pickWorker(b.team, anchor)
	if w == nil || !g.canAfford(b.team, bldDefs[k].Cost) {
		return
	}
	g.pay(b.team, bldDefs[k].Cost)
	site := g.addBuilding(k, b.team, tx, ty, false)
	g.orderBuild(w, site)
	if bldDefs[k].Size >= 3 {
		if w2 := g.pickWorker(b.team, anchor); w2 != nil && w2 != w {
			g.orderBuild(w2, site)
		}
	}
}

// assign sends a villager to gather k.
func (b *Brain) assign(g *Game, u *Unit, k ResKind) {
	home := g.homeOf(b.team)
	ux, uy := toTile(u.X), toTile(u.Y)
	if k == Food {
		for _, f := range g.buildings {
			if f.Team == b.team && f.Kind == Farm && f.Built && (f.farmer == nil || f.farmer.dead || f.farmer.FarmB != f) {
				g.orderFarm(u, f)
				return
			}
		}
		if r := g.findReachableResource(ux, uy, kindIs(Food), home.X, home.Y, 14); r != nil {
			g.orderGather(u, r)
			return
		}
		if prey := g.nearestAnimal(home, 16); prey != nil {
			g.orderAttackUnit(u, prey)
			return
		}
		if ok, _ := g.bldAvailable(b.team, Farm); ok && g.canAfford(b.team, bldDefs[Farm].Cost) {
			if tx, ty, ok := g.findSite(Farm, b.team, b.siteAnchor(g, Farm), 10); ok {
				g.pay(b.team, bldDefs[Farm].Cost)
				g.orderBuild(u, g.addBuilding(Farm, b.team, tx, ty, false))
				return
			}
		}
		k = Wood
	}
	near := home
	for _, d := range g.buildings {
		if d.Team == b.team && d.Built && d.Kind != TownCenter && d.def().Drops[k] {
			near = pt{d.TX + 1, d.TY + 1}
			break
		}
	}
	if r := g.findReachableResource(ux, uy, kindIs(k), near.X, near.Y, 30); r != nil {
		g.orderGather(u, r)
	}
}

func (g *Game) nearestAnimal(p pt, radius int) *Unit {
	var best *Unit
	bd := float64(radius * TileSize)
	x, y := tileCenter(p.X), tileCenter(p.Y)
	for _, u := range g.units {
		if u.Kind == Deer {
			if d := math.Hypot(u.X-x, u.Y-y); d < bd {
				best, bd = u, d
			}
		}
	}
	return best
}

// upkeep runs the routine work that does not need a decision: idle
// villagers, unattended foundations, defense and keeping attacks going.
func (b *Brain) upkeep(g *Game) {
	team := b.team
	info := g.info(team)
	for _, u := range g.units {
		if u.Team != team || u.Kind != Villager || u.Order != OrdIdle {
			continue
		}
		// Pick the resource furthest below its target share.
		best, bestD := Wood, math.Inf(-1)
		for k := range numRes {
			if d := b.eco[k]*float64(info.vills) - float64(info.gatherers[k]); d > bestD {
				best, bestD = ResKind(k), d
			}
		}
		b.assign(g, u, best)
		info.gatherers[best]++
	}
	for _, site := range g.buildings {
		if site.Team != team || site.Built {
			continue
		}
		staffed := false
		for _, u := range g.units {
			if u.BuildB == site {
				staffed = true
				break
			}
		}
		if !staffed {
			if w := g.pickWorker(team, pt{site.TX, site.TY}); w != nil {
				g.orderBuild(w, site)
			}
		}
	}
	home := g.homeOf(team)
	hx, hy := tileCenter(home.X), tileCenter(home.Y)
	intruder := g.nearestEnemyUnit(team, hx, hy, 14*TileSize)
	if intruder != nil && !b.attacking {
		for _, u := range g.units {
			if u.Team == team && u.def().Military && !u.def().OnlyBuildings && u.def().Heal == 0 && u.Target == nil {
				g.orderAttackUnit(u, intruder)
			}
		}
	}
	// Ring the town bell when raiders outnumber the defenders, and send the
	// villagers back to work once the base has been quiet for a while.
	raiders := g.unitsNear(1-team, home, 12*TileSize, true)
	defenders := g.unitsNear(team, home, 12*TileSize, true)
	switch {
	case !b.belled && raiders >= 4 && raiders > 2*defenders:
		b.belled, b.calmT = true, 0
		g.ringBell(team)
	case b.belled && raiders == 0:
		if b.calmT += upkeepTime; b.calmT >= 8 {
			b.belled = false
			g.ungarrisonAll(team)
		}
	case b.belled:
		b.calmT = 0
	}
	if b.attacking {
		b.sendArmy(g)
	}
	for _, t := range g.buildings {
		if t.Team == team && t.Built && len(t.def().Trains) > 0 && t.Kind != TownCenter && !t.HasRally {
			t.HasRally, t.RallyX, t.RallyY = true, hx+48, hy+48
		}
	}
}

// Scripted policy: a fixed build order and army plan. It is the opponent
// for "Play vs Scripted AI" and the benchmark the learner is measured against.

type scripted struct{ rng *rand.Rand }

func (s *scripted) Name() string { return "Scripted AI" }

func (s *scripted) Choose(g *Game, team int, obs []float64, mask []bool) (int, float64, float64) {
	b := g.brains[team]
	p := &g.players[team]
	info := g.info(team)
	pop, cap := g.pop(team), g.popCap(team)
	age := p.Age
	try := func(a int) bool { return mask[a] }
	need := func(k BldKind, a int, minAge, minVills int) bool {
		return age >= minAge && info.all[k] == 0 && info.vills >= minVills && try(a)
	}

	housing := info.all[House] - info.built[House]
	if cap < MaxPop && pop+3 >= cap && housing == 0 && try(AHouse) {
		return AHouse, 0, 0
	}
	targetVills := []int{20, 30, 40, 50}[age]
	if info.vills < targetVills && info.tcQueue == 0 && try(AVillager) {
		return AVillager, 0, 0
	}
	for _, step := range []struct {
		k             BldKind
		a, age, vills int
	}{
		{LumberCamp, ALumber, 0, 4}, {Mill, AMill, 0, 7}, {MiningCamp, AMine, 0, 10}, {Barracks, ABarracks, 0, 12},
		{Blacksmith, ABlacksmith, 0, 16}, {ArcheryRange, ARange, 1, 0}, {Stable, AStable, 1, 0},
		{Monastery, AMonastery, 2, 0}, {SiegeWorkshop, ASiegeWorkshop, 2, 0}, {Castle, ACastle, 2, 0},
	} {
		if need(step.k, step.a, step.age, step.vills) {
			return step.a, 0, 0
		}
	}
	if info.vills >= 16+6*age && try(AAgeUp) {
		return AAgeUp, 0, 0
	}
	if g.resourceNear(g.homeOf(team), Food, 12) < 300 && info.all[Farm] < info.gatherers[Food]+2 && try(AFarm) {
		return AFarm, 0, 0
	}
	if info.vills >= 8 && s.rng.Intn(3) == 0 && try(AResearchEco) {
		return AResearchEco, 0, 0
	}
	if info.armyTotal >= 5 && s.rng.Intn(3) == 0 && try(AResearchMil) {
		return AResearchMil, 0, 0
	}
	want := [numRes][NumAges]float64{
		Food: {0.5, 0.4, 0.35, 0.35}, Wood: {0.4, 0.35, 0.3, 0.25}, Gold: {0.1, 0.2, 0.25, 0.3}, Stone: {0, 0.05, 0.1, 0.1},
	}
	for k := range numRes {
		if a := AEcoFood + int(k); b.eco[k] < want[k][age]-0.08 && try(a) {
			return a, 0, 0
		}
	}
	armyGoal := 5 + 5*age + int(g.time/90)
	if info.armyTotal < armyGoal {
		ei := g.info(1 - team)
		order := []int{AInfantry, AArcher, ACavalry, ASpear, ACavArcher, ALongbow}
		if ei.army[Scout]+ei.army[Knight] > ei.army[Militia]+ei.army[Archer] {
			order = []int{ASpear, AArcher, AInfantry, ACavalry}
		}
		if info.army[Monk] < 2 {
			order = append(order, AMonk)
		}
		if info.army[Ram]+info.army[Mangonel]+info.army[Trebuchet] < 2+age {
			order = append(order, ARam, AMangonel, ATrebuchet)
		}
		start := s.rng.Intn(len(order))
		for i := range order {
			if a := order[(start+i)%len(order)]; try(a) {
				return a, 0, 0
			}
		}
	}
	if !b.attacking && info.armyTotal >= 8+4*b.attacks && try(AAttack) {
		return AAttack, 0, 0
	}
	if b.attacking && info.armyTotal < 3 && try(ARetreat) {
		return ARetreat, 0, 0
	}
	return ANoop, 0, 0
}
