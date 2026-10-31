package main

import "fmt"

type Stats struct {
	Gathered   [numRes]float64
	BuildValue float64
	ArmyValue  float64
	KillValue  float64
	RazeValue  float64
	TechValue  float64
	Kills      int
	Losses     int
	Razed      int
}

type Player struct {
	Res        [numRes]int
	Age        int
	Researched [numTechs]bool
	Tier       [numUnitKinds]int
	Bonus      Bonuses
	Stats      Stats
	alertT     float64
	alertX     float64
	alertY     float64
}

type Score struct {
	Economy, Military, Tech, Total float64
}

var ageScore = [NumAges]float64{0, 150, 400, 800}

// score is the scoring used for display and as the learning signal:
// resources gathered and buildings completed (economy), army trained,
// enemies killed and buildings razed (military), and ages reached.
func (g *Game) score(team int) Score {
	p := &g.players[team]
	s := &p.Stats
	gathered := 0.0
	for _, v := range s.Gathered {
		gathered += v
	}
	sc := Score{
		Economy:  gathered/10 + s.BuildValue/10,
		Military: s.ArmyValue/10 + s.KillValue/5 + s.RazeValue/5,
		Tech:     ageScore[p.Age] + s.TechValue/10,
	}
	sc.Total = sc.Economy + sc.Military + sc.Tech
	return sc
}

func (g *Game) hasBuilt(team int, k BldKind) bool {
	for _, b := range g.buildings {
		if b.Team == team && b.Kind == k && b.Built {
			return true
		}
	}
	return false
}

func (g *Game) countBuildings(team int, k BldKind) (built, all int) {
	for _, b := range g.buildings {
		if b.Team == team && b.Kind == k {
			all++
			if b.Built {
				built++
			}
		}
	}
	return
}

// bldAvailable reports whether team may build k, and why not.
func (g *Game) bldAvailable(team int, k BldKind) (bool, string) {
	def := &bldDefs[k]
	if k == TownCenter {
		return false, "Cannot be built"
	}
	if g.players[team].Age < def.Age {
		return false, "Requires " + ageNames[def.Age]
	}
	if def.Requires != NoBld && !g.hasBuilt(team, def.Requires) {
		return false, "Requires " + bldDefs[def.Requires].Name
	}
	return true, ""
}

func (g *Game) unitAvailable(team int, k UnitKind) (bool, string) {
	if a := unitDefs[k].Age; g.players[team].Age < a {
		return false, "Requires " + ageNames[a]
	}
	return true, ""
}

func (g *Game) agingUp(team int) bool {
	for _, b := range g.buildings {
		if b.Team == team {
			for _, q := range b.Queue {
				if q.Age > 0 {
					return true
				}
			}
		}
	}
	return false
}

// ageUpAvailable reports whether team may start researching the next age.
func (g *Game) ageUpAvailable(team int) (bool, string) {
	age := g.players[team].Age
	if age >= NumAges-1 {
		return false, "Already in the last age"
	}
	if g.agingUp(team) {
		return false, "Already advancing"
	}
	have := 0
	var names string
	for _, k := range ageBuildings[age] {
		if names != "" {
			names += ", "
		}
		names += bldDefs[k].Name
		if g.hasBuilt(team, k) {
			have++
		}
	}
	if need := ageReq[age+1]; have < need {
		return false, fmt.Sprintf("Requires %d of: %s", need, names)
	}
	return true, ""
}

// enqueue adds a unit or an age to a building's queue and pays for it.
func (g *Game) enqueue(b *Building, q QueueItem) (bool, string) {
	switch {
	case len(b.Queue) >= 5:
		return false, "Queue is full"
	case !b.Built:
		return false, "Not built yet"
	case !g.canAfford(b.Team, q.cost()):
		return false, "Not enough resources (" + q.cost().String() + ")"
	}
	g.pay(b.Team, q.cost())
	b.Queue = append(b.Queue, q)
	return true, ""
}

func (g *Game) cancelQueue(b *Building, i int) {
	if i < 0 || i >= len(b.Queue) {
		return
	}
	g.refund(b.Team, b.Queue[i].cost())
	b.Queue = append(b.Queue[:i], b.Queue[i+1:]...)
	if i == 0 {
		b.TrainT = 0
	}
}

func (g *Game) teamTC(team int) *Building {
	for _, b := range g.buildings {
		if b.Team == team && b.Kind == TownCenter && !b.dead {
			return b
		}
	}
	return nil
}
