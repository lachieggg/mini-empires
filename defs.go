package main

import "image/color"

const (
	TileSize   = 32
	ScreenW    = 1280
	ScreenH    = 720
	TopBarH    = 24
	BottomBarH = 168
	ViewH      = ScreenH - TopBarH - BottomBarH
	MiniSize   = 150
	dt         = 1.0 / 60
	CarryCap   = 10
	GatherTime = 0.55
	Reach      = 24.0
	MaxPop     = 125
	Gaia       = 2
	numTeams   = 3
)

type ResKind int

const (
	Food ResKind = iota
	Wood
	Gold
	Stone
	numRes
)

var resNames = [numRes]string{"Food", "Wood", "Gold", "Stone"}

type Cost [numRes]int

func (c Cost) total() int { return c[Food] + c[Wood] + c[Gold] + c[Stone] }

func (c Cost) String() string {
	s := ""
	for k, v := range c {
		if v > 0 {
			if s != "" {
				s += " "
			}
			s += itoa(v) + resNames[k][:1]
		}
	}
	if s == "" {
		return "free"
	}
	return s
}

// Ages.

const NumAges = 4

var (
	ageNames = [NumAges]string{"Dark Age", "Feudal Age", "Castle Age", "Imperial Age"}
	ageCost  = [NumAges]Cost{{}, {Food: 500}, {Food: 800, Gold: 200}, {Food: 1000, Gold: 800}}
	ageTime  = [NumAges]float64{0, 35, 50, 65}
	// ageReq is how many different buildings of the previous age a player
	// needs before researching the next one.
	ageReq = [NumAges]int{0, 2, 2, 2}
)

// Units.

type UnitKind int

const (
	Villager UnitKind = iota
	Militia
	Spearman
	Archer
	Scout
	Knight
	CavalryArcher
	Longbowman
	Monk
	Ram
	Mangonel
	Trebuchet
	Deer
	Boar
	numUnitKinds
)

type UnitDef struct {
	Name          string
	Desc          string
	HP            float64
	Speed         float64
	Attack        float64
	Range         float64 // pixels; 0 is melee
	MinRange      float64
	Cooldown      float64
	MeleeArmor    float64
	PierceArmor   float64
	BonusVsCav    float64
	BonusVsBld    float64
	Splash        float64 // radius of area damage, in pixels
	OnlyBuildings bool
	Heal          float64 // hit points per second healed for nearby allies
	Infantry      bool
	Archer        bool
	Cavalry       bool
	Siege         bool
	Military      bool
	Animal        bool
	Food          int // carcass food for animals
	Cost          Cost
	TrainTime     float64
	Sight         float64
	Radius        float64
	Age           int
}

const T = TileSize

var unitDefs = [numUnitKinds]UnitDef{
	Villager:      {Name: "Villager", Desc: "Gathers resources and constructs buildings.", HP: 25, Speed: 64, Attack: 3, Cooldown: 2, Cost: Cost{Food: 50}, TrainTime: 10, Sight: 4 * T, Radius: 7},
	Militia:       {Name: "Militia", Desc: "Infantry swordsman. Upgrades to Man-at-Arms, Long Swordsman and Champion.", HP: 45, Speed: 64, Attack: 5, Cooldown: 1.6, PierceArmor: 1, Infantry: true, Military: true, Cost: Cost{Food: 60, Gold: 20}, TrainTime: 14, Sight: 6 * T, Radius: 8},
	Spearman:      {Name: "Spearman", Desc: "Anti-cavalry infantry. Strong against Scouts and Knights.", HP: 45, Speed: 66, Attack: 3, Cooldown: 2, BonusVsCav: 15, Infantry: true, Military: true, Cost: Cost{Food: 35, Wood: 25}, TrainTime: 14, Sight: 6 * T, Radius: 8, Age: 1},
	Archer:        {Name: "Archer", Desc: "Ranged unit. Strong against infantry.", HP: 30, Speed: 64, Attack: 4, Range: 4.5 * T, Cooldown: 2, Archer: true, Military: true, Cost: Cost{Wood: 25, Gold: 45}, TrainTime: 14, Sight: 7 * T, Radius: 8, Age: 1},
	Scout:         {Name: "Scout Cavalry", Desc: "Fast light cavalry. Good at raiding and scouting.", HP: 45, Speed: 112, Attack: 3, Cooldown: 2, PierceArmor: 2, Cavalry: true, Military: true, Cost: Cost{Food: 80}, TrainTime: 14, Sight: 8 * T, Radius: 10, Age: 1},
	Knight:        {Name: "Knight", Desc: "Heavy cavalry with strong armor.", HP: 100, Speed: 98, Attack: 10, Cooldown: 1.8, MeleeArmor: 2, PierceArmor: 2, Cavalry: true, Military: true, Cost: Cost{Food: 60, Gold: 75}, TrainTime: 20, Sight: 6 * T, Radius: 10, Age: 2},
	CavalryArcher: {Name: "Cavalry Archer", Desc: "Fast mounted archer.", HP: 50, Speed: 98, Attack: 6, Range: 4 * T, Cooldown: 2, Archer: true, Cavalry: true, Military: true, Cost: Cost{Wood: 40, Gold: 70}, TrainTime: 24, Sight: 7 * T, Radius: 10, Age: 2},
	Longbowman:    {Name: "Longbowman", Desc: "Castle unique unit. Archer with very long range.", HP: 35, Speed: 62, Attack: 6, Range: 6.5 * T, Cooldown: 2, Archer: true, Military: true, Cost: Cost{Wood: 35, Gold: 40}, TrainTime: 18, Sight: 8 * T, Radius: 8, Age: 2},
	Monk:          {Name: "Monk", Desc: "Heals nearby wounded units.", HP: 30, Speed: 50, Heal: 2.5, Military: true, Cost: Cost{Gold: 100}, TrainTime: 30, Sight: 8 * T, Radius: 8, Age: 2},
	Ram:           {Name: "Battering Ram", Desc: "Siege. Destroys buildings, nearly immune to arrows. Cannot attack units.", HP: 175, Speed: 40, Attack: 2, Cooldown: 4, PierceArmor: 180, BonusVsBld: 125, OnlyBuildings: true, Siege: true, Military: true, Cost: Cost{Wood: 160, Gold: 75}, TrainTime: 30, Sight: 3 * T, Radius: 13, Age: 2},
	Mangonel:      {Name: "Mangonel", Desc: "Siege. Hurls stones that damage everything near where they land.", HP: 50, Speed: 40, Attack: 40, Range: 7 * T, MinRange: 3 * T, Cooldown: 6, PierceArmor: 6, BonusVsBld: 35, Splash: 0.9 * T, Siege: true, Military: true, Cost: Cost{Wood: 160, Gold: 135}, TrainTime: 32, Sight: 9 * T, Radius: 13, Age: 2},
	Trebuchet:     {Name: "Trebuchet", Desc: "Siege. Destroys buildings from very long range. Cannot attack units.", HP: 150, Speed: 30, Attack: 0, Range: 15 * T, MinRange: 4 * T, Cooldown: 10, MeleeArmor: 1, PierceArmor: 150, BonusVsBld: 250, Splash: 0.5 * T, OnlyBuildings: true, Siege: true, Military: true, Cost: Cost{Wood: 200, Gold: 200}, TrainTime: 40, Sight: 16 * T, Radius: 15, Age: 3},
	Deer:          {Name: "Deer", Desc: "Hunt it for food.", HP: 5, Speed: 70, Animal: true, Food: 140, Sight: 4 * T, Radius: 9},
	Boar:          {Name: "Wild Boar", Desc: "Dangerous. Hunt it for a lot of food.", HP: 75, Speed: 60, Attack: 8, Cooldown: 2, Animal: true, Food: 340, Sight: 4 * T, Radius: 9},
}

// Buildings.

type BldKind int

const (
	NoBld      BldKind = -1
	TownCenter BldKind = iota - 1
	House
	Mill
	LumberCamp
	MiningCamp
	Farm
	Barracks
	ArcheryRange
	Stable
	Blacksmith
	Tower
	SiegeWorkshop
	Monastery
	Castle
	PalisadeWall
	StoneWall
	numBldKinds
)

type BldDef struct {
	Name      string
	Desc      string
	Size      int
	HP        float64
	Cost      Cost
	BuildTime float64
	Pop       int
	Drops     [numRes]bool
	Walkable  bool
	Wall      bool
	Food      int // farms
	Trains    []UnitKind
	Attack    float64
	Range     float64
	Cooldown  float64
	Arrows    int
	Garrison  int // villagers that can shelter inside
	Age       int
	Requires  BldKind // NoBld when there is no prerequisite
	Height    float64 // sprite height above the footprint center, for HP bars
}

var allDrops = [numRes]bool{true, true, true, true}

var bldDefs = [numBldKinds]BldDef{
	TownCenter:    {Name: "Town Center", Desc: "Trains villagers, researches ages, accepts all resources.", Size: 3, HP: 2400, BuildTime: 90, Pop: 5, Drops: allDrops, Trains: []UnitKind{Villager}, Attack: 5, Range: 6 * T, Cooldown: 2, Arrows: 1, Garrison: 15, Requires: NoBld, Height: 110},
	House:         {Name: "House", Desc: "Supports 5 population.", Size: 2, HP: 550, Cost: Cost{Wood: 25}, BuildTime: 12, Pop: 5, Requires: NoBld, Height: 62},
	Mill:          {Name: "Mill", Desc: "Drop site for food. Required for farms. Researches farming upgrades.", Size: 2, HP: 600, Cost: Cost{Wood: 100}, BuildTime: 20, Drops: [numRes]bool{Food: true}, Requires: NoBld, Height: 80},
	LumberCamp:    {Name: "Lumber Camp", Desc: "Drop site for wood. Researches woodcutting upgrades.", Size: 2, HP: 600, Cost: Cost{Wood: 100}, BuildTime: 20, Drops: [numRes]bool{Wood: true}, Requires: NoBld, Height: 55},
	MiningCamp:    {Name: "Mining Camp", Desc: "Drop site for gold and stone. Researches mining upgrades.", Size: 2, HP: 600, Cost: Cost{Wood: 100}, BuildTime: 20, Drops: [numRes]bool{Gold: true, Stone: true}, Requires: NoBld, Height: 55},
	Farm:          {Name: "Farm", Desc: "Renewable food. One villager farms it.", Size: 2, HP: 300, Cost: Cost{Wood: 60}, BuildTime: 10, Walkable: true, Food: 175, Requires: Mill, Height: 20},
	Barracks:      {Name: "Barracks", Desc: "Trains infantry and researches infantry upgrades.", Size: 3, HP: 1200, Cost: Cost{Wood: 175}, BuildTime: 30, Trains: []UnitKind{Militia, Spearman}, Requires: NoBld, Height: 70},
	ArcheryRange:  {Name: "Archery Range", Desc: "Trains archers and researches archer upgrades.", Size: 3, HP: 1200, Cost: Cost{Wood: 175}, BuildTime: 30, Trains: []UnitKind{Archer, CavalryArcher}, Age: 1, Requires: Barracks, Height: 70},
	Stable:        {Name: "Stable", Desc: "Trains cavalry and researches cavalry upgrades.", Size: 3, HP: 1200, Cost: Cost{Wood: 175}, BuildTime: 30, Trains: []UnitKind{Scout, Knight}, Age: 1, Requires: Barracks, Height: 70},
	Blacksmith:    {Name: "Blacksmith", Desc: "Researches attack and armor upgrades.", Size: 3, HP: 1200, Cost: Cost{Wood: 150}, BuildTime: 30, Requires: NoBld, Height: 75},
	Tower:         {Name: "Watch Tower", Desc: "Fires arrows at nearby enemies.", Size: 1, HP: 1000, Cost: Cost{Wood: 25, Stone: 125}, BuildTime: 40, Attack: 5, Range: 7 * T, Cooldown: 2, Arrows: 1, Garrison: 5, Age: 1, Requires: NoBld, Height: 125},
	SiegeWorkshop: {Name: "Siege Workshop", Desc: "Builds battering rams and mangonels.", Size: 3, HP: 1500, Cost: Cost{Wood: 200}, BuildTime: 40, Trains: []UnitKind{Ram, Mangonel}, Age: 2, Requires: Blacksmith, Height: 75},
	Monastery:     {Name: "Monastery", Desc: "Trains monks, who heal your units.", Size: 3, HP: 2100, Cost: Cost{Stone: 175}, BuildTime: 40, Trains: []UnitKind{Monk}, Age: 2, Requires: NoBld, Height: 95},
	Castle:        {Name: "Castle", Desc: "Fortress. Fires many arrows, supports 20 population, trains Longbowmen and Trebuchets.", Size: 4, HP: 4800, Cost: Cost{Stone: 650}, BuildTime: 90, Pop: 20, Trains: []UnitKind{Longbowman, Trebuchet}, Attack: 11, Range: 8 * T, Cooldown: 2, Arrows: 3, Garrison: 20, Age: 2, Requires: NoBld, Height: 150},
	PalisadeWall:  {Name: "Palisade Wall", Desc: "Cheap wooden wall. Drag to place a line.", Size: 1, HP: 250, Cost: Cost{Wood: 2}, BuildTime: 5, Wall: true, Requires: NoBld, Height: 40},
	StoneWall:     {Name: "Stone Wall", Desc: "Strong stone wall. Drag to place a line.", Size: 1, HP: 1800, Cost: Cost{Stone: 5}, BuildTime: 8, Wall: true, Age: 1, Requires: NoBld, Height: 50},
}

// buildValue is the score value of a building. The Town Center has no
// cost, so it gets a fixed value.
func buildValue(k BldKind) int {
	if k == TownCenter {
		return 500
	}
	return bldDefs[k].Cost.total()
}

// Buildings a villager can construct, by menu page.
var (
	ecoBuildings = []BldKind{House, Mill, LumberCamp, MiningCamp, Farm, Blacksmith, Monastery, PalisadeWall, StoneWall}
	milBuildings = []BldKind{Barracks, ArcheryRange, Stable, SiegeWorkshop, Tower, Castle}
)

// ageBuildings lists the buildings that count towards ageReq for advancing
// out of each age.
var ageBuildings = [NumAges][]BldKind{
	{Barracks, Mill, LumberCamp, MiningCamp, Blacksmith},
	{ArcheryRange, Stable, Tower, Blacksmith},
	{Castle, Monastery, SiegeWorkshop},
	nil,
}

var teamColors = [numTeams]color.RGBA{{50, 100, 230, 255}, {220, 50, 50, 255}, {200, 200, 200, 255}}

var teamNames = [numTeams]string{"Blue", "Red", "Gaia"}

const arrowTime = 0.3

// Map size in tiles. It is set from the menu before games start and must
// not change while any game, including a training game, is running.
var (
	MapW = 96
	MapH = 96
)

var mapSizes = []int{32, 64, 96, 128, 192, 256}

func setMapSize(n int) {
	MapW, MapH = n, n
	cellsW, cellsH = (n+cellTiles-1)/cellTiles, (n+cellTiles-1)/cellTiles
}

// mapScale is the map's area relative to the 80x80 map the generator's
// counts were tuned on.
func mapScale() float64 { return float64(MapW*MapH) / (80 * 80) }

// worldW and worldH are the map's size in world units.
func worldW() float64 { return float64(MapW * TileSize) }
func worldH() float64 { return float64(MapH * TileSize) }
