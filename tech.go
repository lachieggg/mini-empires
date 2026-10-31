package main

// Technologies are researched at buildings, through the same queue as
// units. Unit upgrades (Man-at-Arms, Crossbowman and so on) are
// technologies that raise a unit line's tier.

type Tech int

const (
	Loom Tech = iota
	Wheelbarrow
	HandCart
	DoubleBitAxe
	BowSaw
	GoldMining
	StoneMining
	HorseCollar
	HeavyPlow
	Forging
	IronCasting
	BlastFurnace
	Fletching
	BodkinArrow
	ScaleMail
	ChainMail
	ScaleBarding
	ChainBarding
	PaddedArmor
	ManAtArms
	LongSwordsman
	ChampionUp
	Pikeman
	Crossbowman
	Arbalest
	LightCavalry
	Cavalier
	numTechs
	NoTech Tech = -1
)

type TechDef struct {
	Name     string
	Desc     string
	At       BldKind
	Age      int
	Cost     Cost
	Time     float64
	Requires Tech
	Icon     string   // button icon name, or "" to use the line's unit icon
	Line     UnitKind // for unit upgrades
	Tier     int      // tier reached, for unit upgrades; 0 otherwise
}

var techDefs = [numTechs]TechDef{
	Loom:          {Name: "Loom", Desc: "Villagers +15 HP, +1/+2 armor.", At: TownCenter, Cost: Cost{Gold: 50}, Time: 20, Requires: NoTech, Icon: "loom"},
	Wheelbarrow:   {Name: "Wheelbarrow", Desc: "Villagers move 10% faster and carry 3 more.", At: TownCenter, Age: 1, Cost: Cost{Food: 175, Wood: 50}, Time: 30, Requires: NoTech, Icon: "wheelbarrow"},
	HandCart:      {Name: "Hand Cart", Desc: "Villagers move 10% faster and carry 4 more.", At: TownCenter, Age: 2, Cost: Cost{Food: 300, Wood: 200}, Time: 40, Requires: Wheelbarrow, Icon: "handcart"},
	DoubleBitAxe:  {Name: "Double-Bit Axe", Desc: "Villagers chop wood 20% faster.", At: LumberCamp, Cost: Cost{Food: 100, Wood: 50}, Time: 20, Requires: NoTech, Icon: "doublebitaxe"},
	BowSaw:        {Name: "Bow Saw", Desc: "Villagers chop wood 20% faster.", At: LumberCamp, Age: 1, Cost: Cost{Food: 150, Wood: 100}, Time: 30, Requires: DoubleBitAxe, Icon: "bowsaw"},
	GoldMining:    {Name: "Gold Mining", Desc: "Villagers mine gold 15% faster.", At: MiningCamp, Cost: Cost{Food: 100, Wood: 75}, Time: 25, Requires: NoTech, Icon: "goldmining"},
	StoneMining:   {Name: "Stone Mining", Desc: "Villagers mine stone 15% faster.", At: MiningCamp, Cost: Cost{Food: 100, Wood: 75}, Time: 25, Requires: NoTech, Icon: "stonemining"},
	HorseCollar:   {Name: "Horse Collar", Desc: "New farms hold 75 more food.", At: Mill, Cost: Cost{Food: 75, Wood: 75}, Time: 20, Requires: NoTech, Icon: "horsecollar"},
	HeavyPlow:     {Name: "Heavy Plow", Desc: "New farms hold 125 more food. Farmers carry 1 more.", At: Mill, Age: 1, Cost: Cost{Food: 125, Wood: 125}, Time: 30, Requires: HorseCollar, Icon: "heavyplow"},
	Forging:       {Name: "Forging", Desc: "Infantry and cavalry +1 attack.", At: Blacksmith, Age: 1, Cost: Cost{Food: 150}, Time: 30, Requires: NoTech, Icon: "forging"},
	IronCasting:   {Name: "Iron Casting", Desc: "Infantry and cavalry +1 attack.", At: Blacksmith, Age: 2, Cost: Cost{Food: 220, Gold: 120}, Time: 40, Requires: Forging, Icon: "ironcasting"},
	BlastFurnace:  {Name: "Blast Furnace", Desc: "Infantry and cavalry +2 attack.", At: Blacksmith, Age: 3, Cost: Cost{Food: 275, Gold: 225}, Time: 50, Requires: IronCasting, Icon: "blastfurnace"},
	Fletching:     {Name: "Fletching", Desc: "Archers, towers and Town Centers +1 attack and +1 range.", At: Blacksmith, Age: 1, Cost: Cost{Food: 100, Gold: 50}, Time: 30, Requires: NoTech, Icon: "fletching"},
	BodkinArrow:   {Name: "Bodkin Arrow", Desc: "Archers, towers and Town Centers +1 attack and +1 range.", At: Blacksmith, Age: 2, Cost: Cost{Food: 200, Gold: 100}, Time: 40, Requires: Fletching, Icon: "bodkinarrow"},
	ScaleMail:     {Name: "Scale Mail Armor", Desc: "Infantry +1/+1 armor.", At: Blacksmith, Age: 1, Cost: Cost{Food: 100}, Time: 30, Requires: NoTech, Icon: "scalemail"},
	ChainMail:     {Name: "Chain Mail Armor", Desc: "Infantry +1/+1 armor.", At: Blacksmith, Age: 2, Cost: Cost{Food: 200, Gold: 100}, Time: 40, Requires: ScaleMail, Icon: "chainmail"},
	ScaleBarding:  {Name: "Scale Barding Armor", Desc: "Cavalry +1/+1 armor.", At: Blacksmith, Age: 1, Cost: Cost{Food: 150}, Time: 30, Requires: NoTech, Icon: "scalebarding"},
	ChainBarding:  {Name: "Chain Barding Armor", Desc: "Cavalry +1/+1 armor.", At: Blacksmith, Age: 2, Cost: Cost{Food: 250, Gold: 150}, Time: 40, Requires: ScaleBarding, Icon: "chainbarding"},
	PaddedArmor:   {Name: "Padded Archer Armor", Desc: "Archers +1/+1 armor.", At: Blacksmith, Age: 1, Cost: Cost{Food: 100}, Time: 30, Requires: NoTech, Icon: "paddedarmor"},
	ManAtArms:     {Name: "Man-at-Arms", Desc: "Upgrades Militia to Man-at-Arms.", At: Barracks, Age: 1, Cost: Cost{Food: 100, Gold: 40}, Time: 25, Requires: NoTech, Line: Militia, Tier: 1},
	LongSwordsman: {Name: "Long Swordsman", Desc: "Upgrades Men-at-Arms to Long Swordsmen.", At: Barracks, Age: 2, Cost: Cost{Food: 200, Gold: 65}, Time: 35, Requires: ManAtArms, Line: Militia, Tier: 2},
	ChampionUp:    {Name: "Champion", Desc: "Upgrades Long Swordsmen to Champions.", At: Barracks, Age: 3, Cost: Cost{Food: 750, Gold: 350}, Time: 50, Requires: LongSwordsman, Line: Militia, Tier: 3},
	Pikeman:       {Name: "Pikeman", Desc: "Upgrades Spearmen to Pikemen.", At: Barracks, Age: 2, Cost: Cost{Food: 215, Gold: 90}, Time: 35, Requires: NoTech, Line: Spearman, Tier: 1},
	Crossbowman:   {Name: "Crossbowman", Desc: "Upgrades Archers to Crossbowmen.", At: ArcheryRange, Age: 2, Cost: Cost{Food: 125, Gold: 75}, Time: 35, Requires: NoTech, Line: Archer, Tier: 1},
	Arbalest:      {Name: "Arbalest", Desc: "Upgrades Crossbowmen to Arbalests.", At: ArcheryRange, Age: 3, Cost: Cost{Food: 350, Gold: 300}, Time: 50, Requires: Crossbowman, Line: Archer, Tier: 2},
	LightCavalry:  {Name: "Light Cavalry", Desc: "Upgrades Scout Cavalry to Light Cavalry.", At: Stable, Age: 2, Cost: Cost{Food: 150, Gold: 50}, Time: 35, Requires: NoTech, Line: Scout, Tier: 1},
	Cavalier:      {Name: "Cavalier", Desc: "Upgrades Knights to Cavaliers.", At: Stable, Age: 3, Cost: Cost{Food: 300, Gold: 300}, Time: 50, Requires: NoTech, Line: Knight, Tier: 1},
}

// A tier is one step of a unit line. Tier 0 uses the unit's own stats.
type tierDef struct {
	Name                   string
	File                   string
	HP, Attack, MArm, PArm float64
	Range, BonusVsCav      float64
}

var unitTiers = map[UnitKind][]tierDef{
	Militia: {
		{Name: "Militia", File: "militia", HP: 45, Attack: 5, PArm: 1},
		{Name: "Man-at-Arms", File: "manatarms", HP: 55, Attack: 7, PArm: 1},
		{Name: "Long Swordsman", File: "longswordsman", HP: 60, Attack: 10, MArm: 1, PArm: 1},
		{Name: "Champion", File: "champion", HP: 70, Attack: 13, MArm: 1, PArm: 1},
	},
	Spearman: {
		{Name: "Spearman", File: "spearman", HP: 45, Attack: 3, BonusVsCav: 15},
		{Name: "Pikeman", File: "pikeman", HP: 55, Attack: 4, BonusVsCav: 22},
	},
	Archer: {
		{Name: "Archer", File: "archer", HP: 30, Attack: 4, Range: 4.5 * T},
		{Name: "Crossbowman", File: "crossbowman", HP: 35, Attack: 5, Range: 5 * T},
		{Name: "Arbalest", File: "arbalest", HP: 40, Attack: 6, Range: 5.5 * T},
	},
	Scout: {
		{Name: "Scout Cavalry", File: "scout", HP: 45, Attack: 3, PArm: 2},
		{Name: "Light Cavalry", File: "lightcavalry", HP: 60, Attack: 7, PArm: 2},
	},
	Knight: {
		{Name: "Knight", File: "knight", HP: 100, Attack: 10, MArm: 2, PArm: 2},
		{Name: "Cavalier", File: "cavalier", HP: 120, Attack: 12, MArm: 2, PArm: 2},
	},
}

// Bonuses holds everything a player's research changes.
type Bonuses struct {
	MeleeAtk                   float64 // infantry and cavalry
	ArcherAtk, ArcherRange     float64 // also towers and Town Centers
	InfArmor, CavArmor         float64 // added to both melee and pierce
	ArcherArmor                float64
	VillHP, VillMArm, VillPArm float64
	VillSpeed                  float64 // fraction
	Carry                      int
	FarmFood                   int
	Gather                     [numRes]float64 // fraction faster
}

func (p *Player) tier(k UnitKind) int { return p.Tier[k] }

// techAvailable reports whether team may research t now, and why not.
func (g *Game) techAvailable(team int, t Tech) (bool, string) {
	p := &g.players[team]
	d := &techDefs[t]
	switch {
	case p.Researched[t]:
		return false, "Already researched"
	case g.researching(team, t):
		return false, "Already researching"
	case p.Age < d.Age:
		return false, "Requires " + ageNames[d.Age]
	case d.Requires != NoTech && !p.Researched[d.Requires]:
		return false, "Requires " + techDefs[d.Requires].Name
	}
	return true, ""
}

func (g *Game) researching(team int, t Tech) bool {
	for _, b := range g.buildings {
		if b.Team == team {
			for _, q := range b.Queue {
				if q.Tech == t+1 {
					return true
				}
			}
		}
	}
	return false
}

// techsAt lists the technologies researched at building kind k.
func techsAt(k BldKind) []Tech {
	var ts []Tech
	for t := range numTechs {
		if techDefs[t].At == k {
			ts = append(ts, Tech(t))
		}
	}
	return ts
}

// research applies a finished technology.
func (g *Game) research(team int, t Tech) {
	p := &g.players[team]
	p.Researched[t] = true
	b := &p.Bonus
	switch t {
	case Loom:
		b.VillHP += 15
		b.VillMArm++
		b.VillPArm += 2
	case Wheelbarrow:
		b.VillSpeed += 0.1
		b.Carry += 3
	case HandCart:
		b.VillSpeed += 0.1
		b.Carry += 4
	case DoubleBitAxe, BowSaw:
		b.Gather[Wood] += 0.2
	case GoldMining:
		b.Gather[Gold] += 0.15
	case StoneMining:
		b.Gather[Stone] += 0.15
	case HorseCollar:
		b.FarmFood += 75
	case HeavyPlow:
		b.FarmFood += 125
	case Forging, IronCasting:
		b.MeleeAtk++
	case BlastFurnace:
		b.MeleeAtk += 2
	case Fletching, BodkinArrow:
		b.ArcherAtk++
		b.ArcherRange += T
	case ScaleMail, ChainMail:
		b.InfArmor++
	case ScaleBarding, ChainBarding:
		b.CavArmor++
	case PaddedArmor:
		b.ArcherArmor++
	}
	if d := &techDefs[t]; d.Tier > 0 {
		p.Tier[d.Line] = max(p.Tier[d.Line], d.Tier)
	}
	g.player0Stats(team)
	for _, u := range g.units {
		if u.Team == team {
			g.refreshMaxHP(u)
		}
	}
}

// player0Stats records research in the score: each technology is worth its
// cost, like a building.
func (g *Game) player0Stats(team int) {
	total := 0.0
	for t, done := range g.players[team].Researched {
		if done {
			total += float64(techDefs[t].Cost.total())
		}
	}
	g.players[team].Stats.TechValue = total
}

// refreshMaxHP updates a unit's maximum hit points after research, keeping
// its share of health.
func (g *Game) refreshMaxHP(u *Unit) {
	m := g.maxHP(u)
	if u.MaxHP > 0 && m != u.MaxHP {
		u.HP *= m / u.MaxHP
	}
	u.MaxHP = m
}

func (g *Game) tierOf(u *Unit) *tierDef {
	if u.Team >= Gaia {
		return nil
	}
	ts, ok := unitTiers[u.Kind]
	if !ok {
		return nil
	}
	return &ts[min(g.players[u.Team].Tier[u.Kind], len(ts)-1)]
}

func (g *Game) unitName(u *Unit) string {
	if t := g.tierOf(u); t != nil {
		return t.Name
	}
	return u.def().Name
}

func (g *Game) bonus(team int) *Bonuses {
	if team >= Gaia {
		return &Bonuses{}
	}
	return &g.players[team].Bonus
}

func (g *Game) maxHP(u *Unit) float64 {
	hp := u.def().HP
	if t := g.tierOf(u); t != nil {
		hp = t.HP
	}
	if u.Kind == Villager {
		hp += g.bonus(u.Team).VillHP
	}
	return hp
}

func (g *Game) attackOf(u *Unit) float64 {
	d := u.def()
	a := d.Attack
	if t := g.tierOf(u); t != nil {
		a = t.Attack
	}
	b := g.bonus(u.Team)
	switch {
	case d.Archer:
		a += b.ArcherAtk
	case d.Infantry || d.Cavalry:
		a += b.MeleeAtk
	}
	return a
}

func (g *Game) bonusVsCav(u *Unit) float64 {
	if t := g.tierOf(u); t != nil {
		return t.BonusVsCav
	}
	return u.def().BonusVsCav
}

func (g *Game) armorOf(u *Unit, pierce bool) float64 {
	d := u.def()
	m, p := d.MeleeArmor, d.PierceArmor
	if t := g.tierOf(u); t != nil {
		m, p = t.MArm, t.PArm
	}
	b := g.bonus(u.Team)
	switch {
	case u.Kind == Villager:
		m, p = m+b.VillMArm, p+b.VillPArm
	case d.Archer && !d.Cavalry:
		m, p = m+b.ArcherArmor, p+b.ArcherArmor
	case d.Cavalry:
		m, p = m+b.CavArmor, p+b.CavArmor
	case d.Infantry:
		m, p = m+b.InfArmor, p+b.InfArmor
	}
	if pierce {
		return p
	}
	return m
}

func (g *Game) rangeOf(u *Unit) float64 {
	r := u.def().Range
	if t := g.tierOf(u); t != nil && t.Range > 0 {
		r = t.Range
	}
	if u.def().Archer {
		r += g.bonus(u.Team).ArcherRange
	}
	return r
}

func (g *Game) speedOf(u *Unit) float64 {
	s := u.def().Speed
	if u.Kind == Villager {
		s *= 1 + g.bonus(u.Team).VillSpeed
	}
	return s
}

func (g *Game) carryCap(u *Unit) int {
	c := CarryCap + g.bonus(u.Team).Carry
	if u.FarmB != nil && g.players[u.Team].Researched[HeavyPlow] {
		c++
	}
	return c
}

// tierFor is the upgrade tier team has reached for unit kind k.
func (g *Game) tierFor(k UnitKind, team int) int {
	if team >= Gaia {
		return 0
	}
	return g.players[team].Tier[k]
}

func unitTierName(k UnitKind, tier int) string {
	if ts, ok := unitTiers[k]; ok {
		return ts[min(tier, len(ts)-1)].Name
	}
	return unitDefs[k].Name
}
