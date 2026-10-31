package main

import (
	"math/rand"
	"testing"
	"time"
)

func run(g *Game, seconds float64) {
	for range int(seconds / dt) {
		g.step()
	}
}

func teamUnits(g *Game, team int, k UnitKind) []*Unit {
	var v []*Unit
	for _, u := range g.units {
		if u.Team == team && u.Kind == k {
			v = append(v, u)
		}
	}
	return v
}

func nearestRes(g *Game, k ResKind, u *Unit) *Resource {
	return g.findReachableResource(toTile(u.X), toTile(u.Y), kindIs(k), toTile(u.X), toTile(u.Y), 40)
}

func TestEconomyAndBuilding(t *testing.T) {
	for seed := int64(1); seed <= 4; seed++ {
		g := NewGame(GameOptions{Seed: seed, Human: 0})
		v := teamUnits(g, 0, Villager)
		start := g.players[0].Res
		g.orderGather(v[0], nearestRes(g, Wood, v[0]))
		g.orderGather(v[1], nearestRes(g, Food, v[1]))
		g.orderGather(v[2], nearestRes(g, Gold, v[2]))
		run(g, 60)
		got := g.players[0].Res
		for _, k := range []ResKind{Food, Wood, Gold} {
			if got[k] <= start[k] {
				t.Errorf("seed %d: %s did not increase: %d -> %d", seed, resNames[k], start[k], got[k])
			}
		}

		tc := g.teamTC(0)
		g.enqueue(tc, QueueItem{Unit: Villager})
		g.enqueue(tc, QueueItem{Unit: Villager})
		run(g, 21)
		// 3 villagers and a scout start, and the Town Center supports 5.
		if n := len(teamUnits(g, 0, Villager)); n != 4 {
			t.Errorf("seed %d: want 4 villagers after training, got %d", seed, n)
		}

		g.players[0].Res[Wood] += 500
		for _, u := range teamUnits(g, 0, Villager) {
			u.Selected = true
		}
		hx, hy, ok := g.findSite(House, 0, g.homeOf(0), 10)
		if !ok {
			t.Fatalf("seed %d: no house site", seed)
		}
		g.placeBuilding(House, hx, hy)
		run(g, 40)
		if g.popCap(0) != 10 {
			t.Errorf("seed %d: want pop cap 10, got %d", seed, g.popCap(0))
		}
	}
}

func TestHuntingAndFarming(t *testing.T) {
	g := NewGame(GameOptions{Seed: 3, Human: 0})
	v := teamUnits(g, 0, Villager)
	deer := g.nearestAnimal(g.homeOf(0), 30)
	if deer == nil {
		t.Fatal("no deer")
	}
	g.orderAttackUnit(v[0], deer)
	g.players[0].Res[Wood] += 500
	mill := g.addBuilding(Mill, 0, 14, 14, true)
	fx, fy, ok := g.findSite(Farm, 0, pt{mill.TX, mill.TY}, 8)
	if !ok {
		t.Fatal("no farm site")
	}
	farm := g.addBuilding(Farm, 0, fx, fy, true)
	g.orderFarm(v[1], farm)
	food := g.players[0].Stats.Gathered[Food]
	run(g, 90)
	if !deer.dead {
		t.Error("deer still alive after 90 s of hunting")
	}
	if got := g.players[0].Stats.Gathered[Food]; got < food+30 {
		t.Errorf("food gathered from hunting and farming: %v", got-food)
	}
	if farm.Food >= bldDefs[Farm].Food {
		t.Error("farm was not worked")
	}
}

func TestAgeUp(t *testing.T) {
	g := NewGame(GameOptions{Seed: 2, Human: 0})
	if ok, _ := g.ageUpAvailable(0); ok {
		t.Fatal("age up should need buildings")
	}
	g.addBuilding(Barracks, 0, 15, 15, true)
	g.addBuilding(Mill, 0, 15, 20, true)
	g.players[0].Res[Food] = 600
	if ok, why := g.ageUpAvailable(0); !ok {
		t.Fatal(why)
	}
	g.enqueue(g.teamTC(0), QueueItem{Age: 1})
	run(g, ageTime[1]+1)
	if g.players[0].Age != 1 {
		t.Fatalf("age %d, want 1", g.players[0].Age)
	}
	if ok, _ := g.unitAvailable(0, Archer); !ok {
		t.Error("archers should be available in the Feudal Age")
	}
}

// TestScriptedAIPlays checks that the scripted AI builds an economy and an
// army, ages up, and that one side wins or the game reaches the time limit.
func TestScriptedAIPlays(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	opts := GameOptions{Seed: 5, Human: -1, TimeLimit: 20 * 60}
	opts.Policies[0] = &scripted{rng: rng}
	opts.Policies[1] = &scripted{rng: rng}
	g := NewGame(opts)
	start := time.Now()
	ticks := 0
	maxVills, maxArmy, maxAge := 0, 0, 0
	for !g.over() {
		g.step()
		ticks++
		if ticks%600 == 0 {
			i := g.info(0)
			maxVills, maxArmy, maxAge = max(maxVills, i.vills), max(maxArmy, i.armyTotal), max(maxAge, g.players[0].Age)
		}
	}
	el := time.Since(start)
	t.Logf("game over at %s, winner %d, %d ticks in %v (%.3f ms/tick)", clock(g.time), g.winner, ticks, el, float64(el.Microseconds())/float64(ticks)/1000)
	t.Logf("max villagers %d, max army %d, max age %d, scores %.0f / %.0f", maxVills, maxArmy, maxAge, g.score(0).Total, g.score(1).Total)
	if maxVills < 15 || maxArmy < 5 || maxAge < 1 {
		t.Errorf("scripted AI underperformed: villagers %d, army %d, age %d", maxVills, maxArmy, maxAge)
	}
}

func TestTrainingGameProducesEpisodes(t *testing.T) {
	tr := NewTrainer(t.TempDir() + "/policy.json")
	res, ok := playTrainingGame(7, tr.Net, false, func() bool { return false })
	if !ok || len(res.eps) != 2 {
		t.Fatalf("want 2 episodes, got %d", len(res.eps))
	}
	for _, e := range res.eps {
		if len(e.Steps) < 50 {
			t.Errorf("episode has only %d steps", len(e.Steps))
		}
	}
	t.Logf("training game: %s, %d steps, score %.0f", clock(res.duration), len(res.eps[0].Steps), res.score)
}

func TestMapGeneration(t *testing.T) {
	layouts := map[pt]bool{}
	for style := MapArabia; style < numMapStyles; style++ {
		for seed := int64(1); seed <= 25; seed++ {
			g := NewGame(GameOptions{Seed: seed*7919 + int64(style), Human: 0, Map: style})
			layouts[g.homes[0]] = true
			for y := range MapH {
				for x := range MapW {
					a, m := g.tile(x, y), mirrorPt(pt{x, y})
					b := g.tile(m.X, m.Y)
					if a.Ter != b.Ter || (a.Res == nil) != (b.Res == nil) {
						t.Fatalf("%s seed %d: tile %d,%d is not mirrored", mapNames[style], seed, x, y)
					}
				}
			}
			h0, h1 := g.homeOf(0), g.homeOf(1)
			if p, ok := g.findPath(h0.X, h0.Y+2, rect{h1.X - 2, h1.Y - 2, h1.X + 2, h1.Y + 2}, 1); !ok {
				t.Errorf("%s seed %d: bases are not connected (path %d)", mapNames[style], seed, len(p))
			}
			for _, k := range []ResKind{Food, Wood, Gold, Stone} {
				if g.resourceNear(h0, k, 13) == 0 {
					t.Errorf("%s seed %d: no %s near the base", mapNames[style], seed, resNames[k])
				}
			}
		}
	}
	if len(layouts) < 50 {
		t.Errorf("only %d distinct base positions over 125 maps", len(layouts))
	}
}

func TestMapSizes(t *testing.T) {
	defer setMapSize(96)
	for _, size := range mapSizes {
		setMapSize(size)
		for style := MapArabia; style < numMapStyles; style++ {
			for seed := int64(1); seed <= 4; seed++ {
				g := NewGame(GameOptions{Seed: seed*31 + int64(style), Human: 0, Map: style})
				h0, h1 := g.homeOf(0), g.homeOf(1)
				if _, ok := g.findPath(h0.X, h0.Y+2, rect{h1.X - 2, h1.Y - 2, h1.X + 2, h1.Y + 2}, 1); !ok {
					t.Errorf("%d %s seed %d: bases are not connected", size, mapNames[style], seed)
				}
				for _, k := range []ResKind{Food, Wood, Gold, Stone} {
					if g.resourceNear(h0, k, 13) == 0 {
						t.Errorf("%d %s seed %d: no %s near the base", size, mapNames[style], seed, resNames[k])
					}
				}
			}
		}
	}
	for _, size := range []int{32, 256} {
		setMapSize(size)
		rng := rand.New(rand.NewSource(1))
		opts := GameOptions{Seed: 9, Human: -1, TimeLimit: 6 * 60}
		opts.Policies[0], opts.Policies[1] = &scripted{rng: rng}, &scripted{rng: rng}
		g := NewGame(opts)
		start := time.Now()
		for !g.over() {
			g.step()
		}
		t.Logf("size %d: game over at %s in %v, villagers %d", size, clock(g.time), time.Since(start), g.info(0).vills)
	}
}
