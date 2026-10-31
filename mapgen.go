package main

import (
	"math"

	"mini-empires/internal/art"
)

// Map styles. MapRandom picks one from the seed.
const (
	MapRandom = iota
	MapArabia
	MapForest
	MapLakes
	MapRivers
	MapOasis
	numMapStyles
)

var mapNames = [numMapStyles]string{"Random", "Arabia", "Black Forest", "Lakes", "Rivers", "Oasis"}

type mapParams struct {
	forest    float64 // fbm threshold for forest; lower means more trees
	forestSc  float64 // noise scale of forests
	border    int     // border forest thickness
	lakes     int
	lakeR     [2]float64
	river     bool
	oasis     bool
	dryBias   float64
	scattered float64 // chance of a lone tree on open ground
}

var mapStyles = [numMapStyles]mapParams{
	MapArabia: {forest: 0.68, forestSc: 9, border: 2, lakes: 1, lakeR: [2]float64{2, 3.5}, dryBias: 0.15, scattered: 0.01},
	MapForest: {forest: 0.5, forestSc: 7, border: 4, lakes: 0, dryBias: -0.2, scattered: 0.03},
	MapLakes:  {forest: 0.7, forestSc: 8, border: 2, lakes: 4, lakeR: [2]float64{3, 6}, dryBias: -0.1, scattered: 0.01},
	MapRivers: {forest: 0.66, forestSc: 8, border: 2, river: true, dryBias: -0.05, scattered: 0.01},
	MapOasis:  {forest: 0.78, forestSc: 9, border: 1, oasis: true, dryBias: 0.45, scattered: 0.005},
}

// fbm is fractal value noise in [0, 1].
func fbm(x, y float64, seed uint32) float64 {
	v, amp, sum := 0.0, 1.0, 0.0
	for o := range 4 {
		v += art.Smooth(x, y, seed+uint32(o)*101) * amp
		sum += amp
		x, y, amp = x*2.03, y*2.03, amp*0.5
	}
	return v / sum
}

func mirrorPt(p pt) pt { return pt{MapW - 1 - p.X, MapH - 1 - p.Y} }

// canonical maps a tile to the representative of its mirror pair, so a
// field evaluated on canonical coordinates is symmetric by construction.
func canonical(x, y int) (int, int) {
	m := mirrorPt(pt{x, y})
	if y*MapW+x <= m.Y*MapW+m.X {
		return x, y
	}
	return m.X, m.Y
}

func (g *Game) genMap(style int) {
	rng := g.rng
	if style <= MapRandom || style >= numMapStyles {
		style = 1 + rng.Intn(numMapStyles-1)
	}
	g.mapStyle = style
	mp := mapStyles[style]
	seed := uint32(rng.Int63())
	ox, oy := rng.Float64()*1000, rng.Float64()*1000

	// Bases: point-symmetric through the map center, as far apart as the
	// map allows.
	margin := min(9, MapW/5)
	var home pt
	best := -1.0
	for range 500 {
		p := pt{margin + rng.Intn(max(1, MapW/2-margin-3)), margin + rng.Intn(max(1, MapH-2*margin))}
		d := math.Hypot(float64(p.X-mirrorPt(p).X), float64(p.Y-mirrorPt(p).Y))
		if d > best {
			home, best = p, d
		}
		if d >= 0.575*float64(MapW) {
			break
		}
	}
	g.homes = [2]pt{home, mirrorPt(home)}
	if rng.Intn(2) == 0 {
		g.homes[0], g.homes[1] = g.homes[1], g.homes[0]
	}
	distHome := func(x, y int) float64 {
		d := math.Inf(1)
		for _, h := range g.homes {
			d = math.Min(d, math.Hypot(float64(x-h.X), float64(y-h.Y)))
		}
		return d
	}

	// Ground.
	for y := range MapH {
		for x := range MapW {
			cx, cy := canonical(x, y)
			fx, fy := float64(cx)+ox, float64(cy)+oy
			t := g.tile(x, y)
			dry := fbm(fx/14, fy/14, seed) + mp.dryBias
			dirt := fbm(fx/6+50, fy/6, seed+7)
			switch {
			case dirt > 0.72:
				t.Ter = TerDirt
			case dry > 0.62:
				t.Ter = TerDryGrass
			default:
				t.Ter = TerGrass
			}
			if mp.oasis && dry > 0.8 {
				t.Ter = TerSand
			}
			if d := distHome(x, y); d < 3.5+1.5*art.Hash(cx, cy, seed) {
				t.Ter = TerDirt
			}
			t.Var = uint8(art.Hash(x, y, seed+3) * 4)
		}
	}

	setWater := func(x, y int) {
		if !inBounds(x, y) || distHome(x, y) < near(13) {
			return
		}
		g.tile(x, y).Ter = TerWater
		m := mirrorPt(pt{x, y})
		g.tile(m.X, m.Y).Ter = TerWater
	}
	blob := func(c pt, r float64) {
		for y := c.Y - int(r) - 3; y <= c.Y+int(r)+3; y++ {
			for x := c.X - int(r) - 3; x <= c.X+int(r)+3; x++ {
				d := math.Hypot(float64(x-c.X), float64(y-c.Y))
				if d < r*(0.7+0.6*fbm(float64(x)/3+ox, float64(y)/3, seed+11)) {
					setWater(x, y)
				}
			}
		}
	}
	randomSpot := func(minHome float64) pt {
		for range 200 {
			p := pt{6 + rng.Intn(MapW-12), 6 + rng.Intn(MapH-12)}
			if distHome(p.X, p.Y) >= minHome {
				return p
			}
		}
		return pt{MapW / 2, MapH / 2}
	}
	for range scaled(mp.lakes) * min(mp.lakes, 1) {
		blob(randomSpot(near(16)), mp.lakeR[0]+rng.Float64()*(mp.lakeR[1]-mp.lakeR[0]))
	}
	if mp.oasis {
		blob(pt{MapW / 2, MapH / 2}, (6+rng.Float64()*2)*math.Sqrt(mapScale()))
	}
	if mp.river {
		g.carveRiver(seed, ox, distHome)
	}
	// Shores.
	for y := range MapH {
		for x := range MapW {
			if g.tile(x, y).Ter == TerWater {
				continue
			}
			for _, d := range dirs8 {
				if inBounds(x+d.X, y+d.Y) && g.tile(x+d.X, y+d.Y).Ter == TerWater {
					g.tile(x, y).Ter = TerSand
					break
				}
			}
		}
	}
	if mp.oasis {
		for y := range MapH {
			for x := range MapW {
				if math.Hypot(float64(x)-float64(MapW-1)/2, float64(y)-float64(MapH-1)/2) < 12*math.Sqrt(mapScale()) && g.tile(x, y).Ter != TerWater {
					g.tile(x, y).Ter = TerSand
				}
			}
		}
	}

	amounts := [numRes]int{Food: 150, Wood: 100, Gold: 400, Stone: 350}
	place := func(x, y int, k ResKind, v uint8) {
		for _, p := range []pt{{x, y}, mirrorPt(pt{x, y})} {
			if !inBounds(p.X, p.Y) || g.blocked(p.X, p.Y) {
				continue
			}
			t := g.tile(p.X, p.Y)
			if k == Wood && t.Ter == TerSand {
				v = 4
			}
			t.Res = &Resource{Kind: k, Variant: v, TX: p.X, TY: p.Y, Amount: amounts[k]}
			if k == Wood && t.Ter != TerSand {
				t.Ter = TerForest
			}
		}
	}
	species := []uint8{0, 1, 2, 3, 5}

	// Forests from a noise field, with a species per region.
	for y := range MapH {
		for x := range MapW {
			cx, cy := canonical(x, y)
			if cx != x || cy != y {
				continue
			}
			fx, fy := float64(x)+ox, float64(y)+oy
			edge := min(x, y, MapW-1-x, MapH-1-y)
			f := fbm(fx/mp.forestSc, fy/mp.forestSc, seed+21)
			border := edge < mp.border+int(2*fbm(fx/4, fy/4, seed+22))
			if distHome(x, y) < 8 || g.tile(x, y).Ter == TerWater {
				continue
			}
			sp := species[int(fbm(fx/20, fy/20, seed+23)*float64(len(species)*2))%len(species)]
			if f > mp.forest || border || rng.Float64() < mp.scattered {
				place(x, y, Wood, sp)
			}
		}
	}

	// Home resources at random bearings, the same for both players.
	clearAround := func(c pt, r int) {
		for y := c.Y - r; y <= c.Y+r; y++ {
			for x := c.X - r; x <= c.X+r; x++ {
				if inBounds(x, y) {
					g.tile(x, y).Res = nil
					if g.tile(x, y).Ter == TerForest {
						g.tile(x, y).Ter = TerGrass
					}
				}
			}
		}
	}
	h := g.homes[0]
	clearAround(h, 4)
	clearAround(g.homes[1], 4)
	bearing := rng.Float64() * 2 * math.Pi
	at := func(dist float64) pt {
		bearing += math.Pi/2 + rng.Float64()*0.7
		return pt{h.X + int(math.Round(math.Cos(bearing)*dist)), h.Y + int(math.Round(math.Sin(bearing)*dist))}
	}
	cluster := func(c pt, k ResKind, v uint8, n int) {
		placed := 0
		for r := 0; placed < n && r < 4; r++ {
			for dy := -r; dy <= r && placed < n; dy++ {
				for dx := -r; dx <= r && placed < n; dx++ {
					if max(abs(dx), abs(dy)) != r || g.blocked(c.X+dx, c.Y+dy) || distHome(c.X+dx, c.Y+dy) < 4 {
						continue
					}
					place(c.X+dx, c.Y+dy, k, v)
					placed++
				}
			}
		}
	}
	cluster(at(5.5+rng.Float64()), Food, FoodBerries, 6)
	cluster(at(6.5+rng.Float64()*1.5), Gold, 0, 5)
	cluster(at(7+rng.Float64()*2), Stone, 0, 4)
	wood := at(8 + rng.Float64()*2)
	for range 24 {
		cluster(pt{wood.X + rng.Intn(5) - 2, wood.Y + rng.Intn(5) - 2}, Wood, species[rng.Intn(len(species))], 1)
	}

	// Contested resources.
	for range scaled(3 + rng.Intn(3)) {
		cluster(randomSpot(near(13)), Gold, 0, 4+rng.Intn(3))
	}
	for range scaled(2 + rng.Intn(2)) {
		cluster(randomSpot(near(13)), Stone, 0, 3+rng.Intn(3))
	}
	for range scaled(1 + rng.Intn(2)) {
		cluster(randomSpot(near(12)), Food, FoodBerries, 5)
	}

	g.ensureConnected()
}

// carveRiver runs a winding river from one map edge to the opposite edge
// through the center, with fords so the two halves stay connected.
func (g *Game) carveRiver(seed uint32, ox float64, distHome func(x, y int) float64) {
	// Half the river, from the center to an edge; the mirror gives the rest.
	horizontal := g.rng.Intn(2) == 0
	var pts []pt
	for i := 0; i <= MapW/2; i++ {
		off := int((fbm(float64(i)/9+ox, 3, seed+31) - 0.5) * 24)
		if horizontal {
			pts = append(pts, pt{MapW/2 - i, MapH/2 + off})
		} else {
			pts = append(pts, pt{MapW/2 + off, MapH/2 - i})
		}
	}
	fords := map[int]bool{}
	for range 2 {
		f := min(6, MapW/8) + g.rng.Intn(max(1, MapW/2-14))
		for d := -1; d <= 1; d++ {
			fords[f+d] = true
		}
	}
	fords[0], fords[1] = true, true
	for i, p := range pts {
		if fords[i] {
			continue
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				x, y := p.X+dx, p.Y+dy
				if inBounds(x, y) && distHome(x, y) >= 10 {
					g.tile(x, y).Ter = TerWater
					m := mirrorPt(pt{x, y})
					g.tile(m.X, m.Y).Ter = TerWater
				}
			}
		}
	}
}

// ensureConnected makes sure each base can walk to the other and to every
// resource near it, clearing trees and water along a straight line where
// it cannot.
func (g *Game) ensureConnected() {
	reach := func(from pt) []bool {
		seen := make([]bool, MapW*MapH)
		queue := []pt{from}
		seen[from.Y*MapW+from.X] = true
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			for _, d := range dirs8 {
				x, y := p.X+d.X, p.Y+d.Y
				if !inBounds(x, y) || seen[y*MapW+x] || g.blocked(x, y) {
					continue
				}
				// Match findPath: no squeezing diagonally between two blocked tiles.
				if d.X != 0 && d.Y != 0 && (g.blocked(p.X+d.X, p.Y) || g.blocked(p.X, p.Y+d.Y)) {
					continue
				}
				seen[y*MapW+x] = true
				queue = append(queue, pt{x, y})
			}
		}
		return seen
	}
	h0, h1 := g.homes[0], g.homes[1]
	// carve clears trees and water along a line. With all set it also clears
	// any other resource in the way.
	carve := func(a, b pt, all bool) {
		steps := max(abs(b.X-a.X), abs(b.Y-a.Y))
		for i := 0; i <= steps; i++ {
			x := a.X + (b.X-a.X)*i/max(steps, 1)
			y := a.Y + (b.Y-a.Y)*i/max(steps, 1)
			for _, p := range []pt{{x, y}, {x + 1, y}, {x, y + 1}} {
				for _, q := range []pt{p, mirrorPt(p)} {
					if !inBounds(q.X, q.Y) {
						continue
					}
					t := g.tile(q.X, q.Y)
					if t.Res != nil && (t.Res.Kind == Wood || all) {
						t.Res = nil
					}
					if t.Ter == TerWater {
						t.Ter = TerSand
					}
				}
			}
		}
	}
	if !reach(h0)[h1.Y*MapW+h1.X] {
		carve(h0, h1, false)
	}
	if !reach(h0)[h1.Y*MapW+h1.X] {
		carve(h0, h1, true)
	}
	seen := reach(h0)
	for y := h0.Y - 12; y <= h0.Y+12; y++ {
		for x := h0.X - 12; x <= h0.X+12; x++ {
			if !inBounds(x, y) || g.tile(x, y).Res == nil || g.tile(x, y).Res.Kind == Wood {
				continue
			}
			ok := false
			for _, d := range dirs8 {
				if inBounds(x+d.X, y+d.Y) && seen[(y+d.Y)*MapW+x+d.X] {
					ok = true
					break
				}
			}
			if !ok {
				carve(h0, pt{x, y}, false)
				seen = reach(h0)
			}
		}
	}
}

// spawnAnimals places mirrored deer herds and boar near each base, and more
// deer around the map.
func (g *Game) spawnAnimals() {
	rng := g.rng
	add := func(k UnitKind, p pt) {
		for _, q := range []pt{p, mirrorPt(p)} {
			if g.blocked(q.X, q.Y) {
				continue
			}
			u := g.newUnit(k, Gaia, tileCenter(q.X), tileCenter(q.Y))
			u.Home = q
		}
	}
	h := g.homes[0]
	herd := func(c pt, n int) {
		for i := range n {
			add(Deer, pt{c.X + i%2, c.Y + i/2})
		}
	}
	a := rng.Float64() * 2 * math.Pi
	herd(pt{h.X + int(math.Cos(a)*10), h.Y + int(math.Sin(a)*10)}, 3+rng.Intn(2))
	a += 2 + rng.Float64()
	add(Boar, pt{h.X + int(math.Cos(a)*11), h.Y + int(math.Sin(a)*11)})
	for range scaled(3 + rng.Intn(3)) {
		herd(pt{8 + rng.Intn(MapW-16), 8 + rng.Intn(MapH-16)}, 2+rng.Intn(3))
	}
}

// scaled scales a count tuned for an 80x80 map to the current map area.
func scaled(n int) int { return max(1, int(math.Round(float64(n)*mapScale()))) }

// near limits a distance tuned for an 80x80 map so it still fits on small maps.
func near(d float64) float64 { return math.Min(d, float64(MapW)/5) }
