package main

import (
	"container/heap"
	"image"
	"math"
	"strconv"
)

type pt struct{ X, Y int }

// rect is an inclusive tile rectangle.
type rect struct{ x0, y0, x1, y1 int }

func tileRect(x, y int) rect { return rect{x, y, x, y} }

func itoa(v int) string { return strconv.Itoa(v) }

func toTile(v float64) int { return int(math.Floor(v / TileSize)) }

func tileCenter(t int) float64 { return float64(t)*TileSize + TileSize/2 }

func imageRect(x0, y0, x1, y1 int) image.Rectangle { return image.Rect(x0, y0, x1, y1) }

func (r rect) pixels() (x0, y0, x1, y1 float64) {
	return float64(r.x0 * TileSize), float64(r.y0 * TileSize), float64((r.x1 + 1) * TileSize), float64((r.y1 + 1) * TileSize)
}

func distToRect(x, y float64, r rect) float64 {
	x0, y0, x1, y1 := r.pixels()
	return math.Hypot(max(x0-x, 0, x-x1), max(y0-y, 0, y-y1))
}

func nearestOnRect(x, y float64, r rect) (float64, float64) {
	x0, y0, x1, y1 := r.pixels()
	return min(max(x, x0), x1), min(max(y, y0), y1)
}

type Terrain uint8

const (
	TerGrass Terrain = iota
	TerDryGrass
	TerDirt
	TerSand
	TerWater
	TerForest
)

type Tile struct {
	Res *Resource
	Bld *Building
	Ter Terrain
	Var uint8
}

// Resource variants select the sprite: tree species for wood, and berries
// or a carcass for food.
const (
	FoodBerries = iota
	FoodDeer
	FoodBoar
)

type Resource struct {
	Kind    ResKind
	Variant uint8
	TX, TY  int
	Amount  int
}

func inBounds(x, y int) bool { return x >= 0 && y >= 0 && x < MapW && y < MapH }

func (g *Game) tile(x, y int) *Tile { return &g.tiles[y*MapW+x] }

func (g *Game) blocked(x, y int) bool {
	if !inBounds(x, y) {
		return true
	}
	t := &g.tiles[y*MapW+x]
	return t.Res != nil || t.Ter == TerWater || (t.Bld != nil && !t.Bld.def().Walkable)
}

func (g *Game) blockedAt(x, y float64) bool { return g.blocked(toTile(x), toTile(y)) }

func (g *Game) resAt(x, y int) *Resource {
	if !inBounds(x, y) {
		return nil
	}
	return g.tile(x, y).Res
}

func (g *Game) buildingAtTile(x, y int) *Building {
	if !inBounds(x, y) {
		return nil
	}
	return g.tile(x, y).Bld
}

func (g *Game) removeResource(r *Resource) {
	t := g.tile(r.TX, r.TY)
	if t.Res == r {
		t.Res = nil
	}
}

var dirs8 = [8]pt{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}

type node struct {
	i int32
	f float32
}

type nodeHeap []node

func (h nodeHeap) Len() int           { return len(h) }
func (h nodeHeap) Less(a, b int) bool { return h[a].f < h[b].f }
func (h nodeHeap) Swap(a, b int)      { h[a], h[b] = h[b], h[a] }
func (h *nodeHeap) Push(x any)        { *h = append(*h, x.(node)) }
func (h *nodeHeap) Pop() any {
	old := *h
	x := old[len(old)-1]
	*h = old[:len(old)-1]
	return x
}

// pathScratch holds reusable search buffers. A tile's entry is valid only
// when its gen matches the current search.
type pathScratch struct {
	gs     []float32
	came   []int32
	gen    []uint32
	closed []uint32
	cur    uint32
	open   nodeHeap
	queue  []pt
}

func newPathScratch() *pathScratch {
	n := MapW * MapH
	return &pathScratch{gs: make([]float32, n), came: make([]int32, n), gen: make([]uint32, n), closed: make([]uint32, n)}
}

func octile(dx, dy int) float32 {
	return float32(max(dx, dy)) + (math.Sqrt2-1)*float32(min(dx, dy))
}

// findPath runs A* from (sx, sy) to any walkable tile within adj tiles of r.
// If the goal cannot be reached, it returns a path to the closest tile found
// and ok is false.
func (g *Game) findPath(sx, sy int, r rect, adj int) (path []pt, ok bool) {
	gx0, gy0, gx1, gy1 := r.x0-adj, r.y0-adj, r.x1+adj, r.y1+adj
	inGoal := func(x, y int) bool { return x >= gx0 && x <= gx1 && y >= gy0 && y <= gy1 }
	h := func(x, y int) float32 { return octile(max(gx0-x, 0, x-gx1), max(gy0-y, 0, y-gy1)) }
	if !inBounds(sx, sy) {
		return nil, false
	}
	if inGoal(sx, sy) {
		return nil, true
	}

	ps := g.ps
	ps.cur++
	gen := ps.cur
	start := int32(sy*MapW + sx)
	ps.gen[start], ps.gs[start] = gen, 0
	ps.open = append(ps.open[:0], node{start, h(sx, sy)})
	best, bestH := start, h(sx, sy)
	expanded := 0

	for ps.open.Len() > 0 && expanded < MapW*MapH {
		cur := heap.Pop(&ps.open).(node)
		if ps.closed[cur.i] == gen {
			continue
		}
		ps.closed[cur.i] = gen
		expanded++
		cx, cy := int(cur.i)%MapW, int(cur.i)/MapW
		if inGoal(cx, cy) {
			best, ok = cur.i, true
			break
		}
		if hh := h(cx, cy); hh < bestH {
			best, bestH = cur.i, hh
		}
		for _, d := range dirs8 {
			nx, ny := cx+d.X, cy+d.Y
			if g.blocked(nx, ny) {
				continue
			}
			cost := float32(1)
			if d.X != 0 && d.Y != 0 {
				if g.blocked(cx+d.X, cy) || g.blocked(cx, cy+d.Y) {
					continue
				}
				cost = math.Sqrt2
			}
			ni := int32(ny*MapW + nx)
			ng := ps.gs[cur.i] + cost
			if ps.gen[ni] != gen || ng < ps.gs[ni] {
				ps.gen[ni], ps.gs[ni], ps.came[ni] = gen, ng, cur.i
				heap.Push(&ps.open, node{ni, ng + h(nx, ny)})
			}
		}
	}

	for i := best; i != start; i = ps.came[i] {
		path = append(path, pt{int(i) % MapW, int(i) / MapW})
	}
	for a, b := 0, len(path)-1; a < b; a, b = a+1, b-1 {
		path[a], path[b] = path[b], path[a]
	}
	return path, ok
}

// findReachableResource walks outward from tile (sx, sy) and returns the
// first reachable resource accepted by want within maxD tiles of (nx, ny).
func (g *Game) findReachableResource(sx, sy int, want func(*Resource) bool, nx, ny, maxD int) *Resource {
	if !inBounds(sx, sy) {
		return nil
	}
	ps := g.ps
	ps.cur++
	gen := ps.cur
	ps.gen[sy*MapW+sx] = gen
	ps.queue = append(ps.queue[:0], pt{sx, sy})
	for head := 0; head < len(ps.queue) && head < 4000; head++ {
		p := ps.queue[head]
		for _, d := range dirs8 {
			x, y := p.X+d.X, p.Y+d.Y
			if !inBounds(x, y) {
				continue
			}
			i := y*MapW + x
			if r := g.tiles[i].Res; r != nil && r.Amount > 0 && want(r) {
				if dx, dy := x-nx, y-ny; dx*dx+dy*dy <= maxD*maxD {
					return r
				}
			}
			if ps.gen[i] == gen || g.blocked(x, y) {
				continue
			}
			ps.gen[i] = gen
			ps.queue = append(ps.queue, pt{x, y})
		}
	}
	return nil
}

func kindIs(k ResKind) func(*Resource) bool {
	return func(r *Resource) bool { return r.Kind == k }
}

// Spatial grid, rebuilt every tick.

const (
	cellTiles = 4
)

type spatial struct {
	cells [][]*Unit
	occ   [numTeams][]uint8
}

func (g *Game) rebuildGrid() {
	s := g.grid
	for i := range s.cells {
		s.cells[i] = s.cells[i][:0]
	}
	for t := range s.occ {
		clear(s.occ[t])
	}
	for _, u := range g.units {
		if u.Inside != nil {
			continue
		}
		tx, ty := toTile(u.X), toTile(u.Y)
		if !inBounds(tx, ty) {
			continue
		}
		c := (ty/cellTiles)*cellsW + tx/cellTiles
		s.cells[c] = append(s.cells[c], u)
		if o := &s.occ[u.Team][ty*MapW+tx]; *o < 255 {
			*o++
		}
	}
}

// eachNear calls fn for every unit in the grid cells within radius of (x, y).
func (g *Game) eachNear(x, y, radius float64, fn func(*Unit)) {
	c0x, c0y := toTile(x-radius)/cellTiles, toTile(y-radius)/cellTiles
	c1x, c1y := toTile(x+radius)/cellTiles, toTile(y+radius)/cellTiles
	for cy := max(c0y, 0); cy <= min(c1y, cellsH-1); cy++ {
		for cx := max(c0x, 0); cx <= min(c1x, cellsW-1); cx++ {
			for _, u := range g.grid.cells[cy*cellsW+cx] {
				fn(u)
			}
		}
	}
}

func (g *Game) unitOnTile(x, y int, team int) bool {
	if !inBounds(x, y) {
		return false
	}
	i := y*MapW + x
	if team >= 0 {
		return g.grid.occ[team][i] > 0
	}
	return g.grid.occ[0][i]+g.grid.occ[1][i]+g.grid.occ[2][i] > 0
}

var cellsW, cellsH = (96 + cellTiles - 1) / cellTiles, (96 + cellTiles - 1) / cellTiles

func newSpatial() *spatial {
	s := &spatial{cells: make([][]*Unit, cellsW*cellsH)}
	for t := range s.occ {
		s.occ[t] = make([]uint8, MapW*MapH)
	}
	return s
}
