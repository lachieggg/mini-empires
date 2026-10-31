package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// Button is one entry of the command panel. Its hotkey is the key at its
// grid position, as in AoE II's grid hotkey layout.
type Button struct {
	Key     ebiten.Key
	Icon    *ebiten.Image
	Name    string
	Cost    Cost
	Desc    string
	Reason  string
	Enabled bool
	Do      func()
	x, y    float64
}

const (
	buttonSize = 44
	buttonGap  = 4
	panelX     = 12
)

var gridKeys = [15]ebiten.Key{
	ebiten.KeyQ, ebiten.KeyW, ebiten.KeyE, ebiten.KeyR, ebiten.KeyT,
	ebiten.KeyA, ebiten.KeyS, ebiten.KeyD, ebiten.KeyF, ebiten.KeyG,
	ebiten.KeyZ, ebiten.KeyX, ebiten.KeyC, ebiten.KeyV, ebiten.KeyB,
}

var keyNames = [15]string{"Q", "W", "E", "R", "T", "A", "S", "D", "F", "G", "Z", "X", "C", "V", "B"}

func panelY() float64 { return float64(ScreenH - bottomH() + 12) }

func slotPos(i int) (float64, float64) {
	return panelX + float64(i%5)*(buttonSize+buttonGap), panelY() + float64(i/5)*(buttonSize+buttonGap)
}

const (
	pageMain = iota
	pageEco
	pageMil
)

// panelButtons lists the commands for the current selection.
func (g *Game) panelButtons() []Button {
	team := g.human
	if team < 0 {
		return nil
	}
	s := sprites()
	var bs []Button
	put := func(slot int, b Button) {
		b.Key = gridKeys[slot]
		b.x, b.y = slotPos(slot)
		bs = append(bs, b)
	}
	if len(g.selectedVillagers()) > 0 && g.selectedVillagers()[0].Team == team {
		switch g.page {
		case pageMain:
			put(0, Button{Icon: s.buttons["eco"], Name: "Build Economic Building", Desc: "Houses, drop sites and farms.", Enabled: true, Do: func() { g.page = pageEco }})
			put(1, Button{Icon: s.buttons["mil"], Name: "Build Military Building", Desc: "Barracks, archery range, stable, towers and castles.", Enabled: true, Do: func() { g.page = pageMil }})
		case pageEco, pageMil:
			list := ecoBuildings
			if g.page == pageMil {
				list = milBuildings
			}
			for i, k := range list {
				def := &bldDefs[k]
				ok, why := g.bldAvailable(team, k)
				if ok && !g.canAfford(team, def.Cost) {
					why = "Not enough resources"
				}
				put(i, Button{Icon: s.bldIcon(k, team, g.players[team].Age), Name: def.Name, Cost: def.Cost, Desc: def.Desc, Reason: why, Enabled: ok && why == "", Do: func() { g.startPlacing(k) }})
			}
			put(14, Button{Icon: s.buttons["back"], Name: "Back", Enabled: true, Do: func() { g.page = pageMain }})
		}
	}
	if len(g.selectedUnits()) > 0 && g.selectedUnits()[0].Team == team {
		put(6, Button{Icon: s.buttons["stop"], Name: "Stop", Desc: "Stop what the selected units are doing.", Enabled: true, Do: func() {
			for _, u := range g.selectedUnits() {
				u.reset(OrdIdle)
				u.Res = nil
			}
		}})
	}
	if b := g.selBld; b != nil && b.Team == team && b.Built {
		for i, k := range b.def().Trains {
			ud := &unitDefs[k]
			ok, why := g.unitAvailable(team, k)
			put(i, Button{Icon: s.unitIcon(k, team, g.tierFor(k, team)), Name: "Train " + unitTierName(k, g.tierFor(k, team)), Cost: ud.Cost, Desc: ud.Desc, Reason: why, Enabled: ok, Do: func() {
				if ok, why := g.enqueue(b, QueueItem{Unit: k}); !ok {
					g.flash(why)
				}
			}})
		}
		// Technologies go on the second row: the next step of each line
		// that has not been researched yet.
		slot := 5
		for _, t := range techsAt(b.Kind) {
			d := &techDefs[t]
			p := &g.players[team]
			if p.Researched[t] || (d.Requires != NoTech && !p.Researched[d.Requires]) || slot > 9 {
				continue
			}
			ok, why := g.techAvailable(team, t)
			put(slot, Button{Icon: s.techIcon(t, team), Name: "Research " + d.Name, Cost: d.Cost, Desc: d.Desc, Reason: why, Enabled: ok, Do: func() {
				if ok, why := g.enqueue(b, QueueItem{Tech: t + 1}); !ok {
					g.flash(why)
				}
			}})
			slot++
		}
		if b.def().Garrison > 0 {
			if b.Kind == TownCenter {
				put(10, Button{Icon: s.buttons["townbell"], Name: "Ring Town Bell", Desc: "Villagers run into the nearest Town Center, tower or castle. Each one inside adds an arrow.", Enabled: true, Do: func() { g.ringBell(team) }})
			}
			if n := g.garrisoned(b); n > 0 {
				put(11, Button{Icon: s.buttons["ungarrison"], Name: fmt.Sprintf("Ungarrison (%d inside)", n), Desc: "Send everyone inside back out to work.", Enabled: true, Do: func() { g.ungarrison(b) }})
			}
		}
		if b.Kind == TownCenter && g.players[team].Age < NumAges-1 {
			next := g.players[team].Age + 1
			ok, why := g.ageUpAvailable(team)
			put(1, Button{Icon: s.buttons["age"+itoa(next)], Name: "Advance to " + ageNames[next], Cost: ageCost[next],
				Desc: "Unlocks new buildings, units and technologies.", Reason: why, Enabled: ok,
				Do: func() {
					if ok, why := g.enqueue(b, QueueItem{Age: next}); !ok {
						g.flash(why)
					}
				}})
		}
	}
	return bs
}

const (
	queueX    = 270
	queueSize = 34
)

func queueSlot(i int) (float64, float64) {
	return queueX + float64(i)*(queueSize+4), ScreenH - 52
}

// queueSlotAt returns the index of the selected building's queue entry
// under the cursor, or -1.
func (g *Game) queueSlotAt(mx, my int) int {
	b := g.selBld
	if b == nil || b.Team != g.human {
		return -1
	}
	for i := range b.Queue {
		x, y := queueSlot(i)
		if float64(mx) >= x && float64(mx) < x+queueSize && float64(my) >= y && float64(my) < y+queueSize {
			return i
		}
	}
	return -1
}
