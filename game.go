package main

import (
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

type Effect struct {
	x0, y0, x1, y1, t float64
	h                 float64 // launch height in pixels
	kind              int
	dur               float64 // total flight time
}

const corpseTime = 10.0

type Corpse struct {
	Kind   UnitKind
	Team   int
	X, Y   float64
	Facing float64
	t      float64
}

type Marker struct {
	x, y, t float64
	attack  bool
}

type GameOptions struct {
	Seed      int64
	Human     int // the team a person controls, or -1
	Policies  [2]Policy
	Record    bool    // keep decision steps for training
	Visual    bool    // keep effects that only matter on screen
	TimeLimit float64 // seconds; 0 for none
	Map       int     // a map style, or MapRandom
	Fog       bool    // fog of war for the human player
}

type Game struct {
	rng       *rand.Rand
	tiles     []Tile
	units     []*Unit
	buildings []*Building
	players   [numTeams]Player
	brains    [2]*Brain
	time      float64
	timeLimit float64
	winner    int // -1 while playing, a team, or 2 for a draw
	human     int
	visual    bool
	effects   []Effect
	corpses   []Corpse
	marker    Marker
	ps        *pathScratch
	grid      *spatial
	homes     [2]pt
	mapStyle  int
	ground    *groundMaps
	fog       *fogMap
	seed      int64

	camX, camY     float64
	zoom           float64
	view           *ebiten.Image
	selBld         *Building
	dragging       bool
	dragX0, dragY0 int
	pan            panState
	wallDrag       bool
	wallStart      pt
	placing        BldKind
	page           int
	hover          *Button
	msg            string
	msgT           float64
	idleIdx        int
	groups         [10][]*Unit
	lastGroup      int
	lastGroupT     float64

	minimap  *ebiten.Image
	minimapT float64
}

func NewGame(o GameOptions) *Game {
	g := &Game{
		rng:       rand.New(rand.NewSource(o.Seed)),
		tiles:     make([]Tile, MapW*MapH),
		placing:   NoBld,
		winner:    -1,
		human:     o.Human,
		visual:    o.Visual,
		timeLimit: o.TimeLimit,
		ps:        newPathScratch(),
		grid:      newSpatial(),
		seed:      o.Seed,
	}
	g.genMap(o.Map)
	for team := range 2 {
		g.players[team].Res = [numRes]int{Food: 200, Wood: 200, Gold: 100, Stone: 200}
		h := g.homes[team]
		tc := g.addBuilding(TownCenter, team, h.X-1, h.Y-1, true)
		for range 3 {
			g.spawnUnit(Villager, team, tc)
			g.rebuildGrid()
		}
		sx, sy := g.spawnSpot(tc.rect())
		g.newUnit(Scout, team, sx, sy)
		if p := o.Policies[team]; p != nil {
			g.brains[team] = newBrain(team, p, o.Record)
		}
	}
	g.spawnAnimals()
	if o.Fog && o.Human >= 0 {
		g.fog = newFog(o.Human)
	}
	g.rebuildGrid()
	// Start zoomed out over most of the map, leaning toward your own base.
	g.zoom = defaultZoom()
	mx, my := worldW()/2, worldH()/2
	if o.Human >= 0 {
		cx, cy := g.teamTC(o.Human).center()
		mx, my = (mx+2*cx)/3, (my+2*cy)/3
	}
	g.centerOn(mx, my)
	return g
}

func (g *Game) over() bool { return g.winner >= 0 }

func (g *Game) step() {
	if g.over() {
		return
	}
	g.time += dt
	if g.msgT > 0 {
		g.msgT -= dt
	}
	if g.marker.t > 0 {
		g.marker.t -= dt
	}
	for i := range g.players {
		if g.players[i].alertT > 0 {
			g.players[i].alertT -= dt
		}
	}
	g.rebuildGrid()
	g.updateBrains()
	g.updateFog()
	for _, b := range g.buildings {
		g.updateBuilding(b)
	}
	for _, u := range g.units {
		if !u.dead && u.Inside == nil {
			g.updateUnit(u)
		}
	}
	g.separate()
	g.cleanup()

	fx := g.effects[:0]
	for _, e := range g.effects {
		if e.t -= dt; e.t > 0 {
			fx = append(fx, e)
		}
	}
	g.effects = fx
	cs := g.corpses[:0]
	for _, c := range g.corpses {
		if c.t -= dt; c.t > 0 {
			cs = append(cs, c)
		}
	}
	g.corpses = cs

	if g.timeLimit > 0 && g.time >= g.timeLimit && !g.over() {
		a, b := g.score(0).Total, g.score(1).Total
		switch {
		case a > b:
			g.winner = 0
		case b > a:
			g.winner = 1
		default:
			g.winner = 2
		}
	}
	if g.over() {
		for _, b := range g.brains {
			if b != nil {
				b.finish(g)
			}
		}
	}
}

func (g *Game) flash(s string) {
	g.msg, g.msgT = s, 4
}

// Camera and coordinates. The simulation runs on a square world grid, and
// iso projects it to a 2:1 isometric view where one tile is a 64x32 diamond
// at zoom 1. The camera position is in unzoomed isometric pixels, and the
// map is drawn straight to the screen with every sprite scaled by the zoom.

func iso(wx, wy float64) (float64, float64) { return wx - wy, (wx + wy) / 2 }

const maxZoom = 2.0

// fitZoom is the zoom at which the whole map fits between the HUD bars.
func fitZoom() float64 {
	w := float64(MapW+MapH) * TileSize
	return min(ScreenW/w, float64(ScreenH-TopBarH-bottomH())/(w/2))
}

func minZoom() float64 { return fitZoom() * 0.9 }

// defaultZoom shows most of the map while units stay recognizable.
func defaultZoom() float64 { return min(1, fitZoom()*1.7) }

// zoomLevels are the magnifications on Alt+1 to Alt+5: the whole map, the
// default overview, then close, normal and near.
func zoomLevels() [5]float64 {
	return [5]float64{fitZoom(), defaultZoom(), 0.6, 1, 1.6}
}

func (g *Game) zoomLevel() float64 {
	if g.zoom == 0 {
		return 1
	}
	return g.zoom
}

// viewSize is the size of the visible map area in unzoomed isometric pixels.
func (g *Game) viewSize() (float64, float64) {
	z := g.zoomLevel()
	return ScreenW / z, ScreenH / z
}

// toView maps a world point to the screen.
func (g *Game) toView(wx, wy float64) (float64, float64) {
	ix, iy := iso(wx, wy)
	z := g.zoomLevel()
	return (ix - g.camX) * z, (iy - g.camY) * z
}

func (g *Game) toScreen(wx, wy float64) (float64, float64) { return g.toView(wx, wy) }

func (g *Game) screenToView(sx, sy int) (float64, float64) { return float64(sx), float64(sy) }

func (g *Game) screenToWorld(sx, sy int) (float64, float64) {
	z := g.zoomLevel()
	ix, iy := float64(sx)/z+g.camX, float64(sy)/z+g.camY
	return iy + ix/2, iy - ix/2
}

func (g *Game) clampCam() {
	z := g.zoomLevel()
	vw, vh := g.viewSize()
	vh -= float64(bottomH()) / z
	lo, hi := float64(-MapH*TileSize), worldW()-vw
	if hi < lo {
		lo, hi = (lo+hi)/2, (lo+hi)/2
	}
	g.camX = min(max(g.camX, lo), hi)
	lo, hi = -120-TopBarH/z, (worldW()+worldH())/2-vh
	if hi < lo {
		lo, hi = (lo+hi)/2, (lo+hi)/2
	}
	g.camY = min(max(g.camY, lo), hi)
}

func (g *Game) centerOn(x, y float64) {
	ix, iy := iso(x, y)
	z := g.zoomLevel()
	g.camX, g.camY = ix-ScreenW/2/z, iy-(TopBarH+ViewH/2)/z
	g.clampCam()
}

// setZoom changes the zoom, keeping the world point under screen pixel
// (sx, sy) in place.
func (g *Game) setZoom(z float64, sx, sy int) {
	z0 := g.zoomLevel()
	z = min(max(z, minZoom()), maxZoom)
	g.camX += float64(sx)/z0 - float64(sx)/z
	g.camY += float64(sy)/z0 - float64(sy)/z
	g.zoom = z
	g.clampCam()
}

func (g *Game) zoomAt(factor float64, sx, sy int) { g.setZoom(g.zoomLevel()*factor, sx, sy) }

// fitMap zooms out to show the whole map.
func (g *Game) fitMap() {
	g.zoom = fitZoom()
	g.centerOn(worldW()/2, worldH()/2)
}

const MiniScale = 0.62

func miniOrigin() (int, int) {
	return ScreenW - int(2*MiniSize*MiniScale) - 12, ScreenH - BottomBarH + 32
}

// miniGeo maps square minimap pixels to the diamond drawn on screen.
func miniGeo() ebiten.GeoM {
	var m ebiten.GeoM
	m.SetElement(0, 0, MiniScale)
	m.SetElement(0, 1, -MiniScale)
	m.SetElement(1, 0, MiniScale/2)
	m.SetElement(1, 1, MiniScale/2)
	ox, oy := miniOrigin()
	m.Translate(float64(ox)+MiniSize*MiniScale, float64(oy))
	return m
}

// miniToWorld returns the world point under screen pixel (mx, my) and
// whether it lies on the minimap.
func miniToWorld(mx, my int) (float64, float64, bool) {
	m := miniGeo()
	m.Invert()
	qx, qy := m.Apply(float64(mx), float64(my))
	sc := float64(MapW*TileSize) / MiniSize
	return qx * sc, qy * sc, qx >= 0 && qy >= 0 && qx < MiniSize && qy < MiniSize
}

// Scene and picking.

type drawable struct {
	depth  float64
	x, y   float64 // sprite top-left on screen
	sp     *Sprite
	unit   *Unit
	bld    *Building
	res    *Resource
	corpse *Corpse
}

func (g *Game) visibleTiles() (tx0, ty0, tx1, ty1 int) {
	vw, vh := g.viewSize()
	tx0, ty0, tx1, ty1 = MapW, MapH, -1, -1
	for _, c := range [][2]float64{{-64, -16}, {vw + 64, -16}, {-64, vh + 220}, {vw + 64, vh + 220}} {
		ix, iy := c[0]+g.camX, c[1]+g.camY
		x, y := toTile(iy+ix/2), toTile(iy-ix/2)
		tx0, ty0, tx1, ty1 = min(tx0, x), min(ty0, y), max(tx1, x), max(ty1, y)
	}
	return max(tx0, 0), max(ty0, 0), min(tx1, MapW-1), min(ty1, MapH-1)
}

// visible reports whether a rectangle in screen coordinates is on screen.
func (g *Game) visible(x, y, w, h float64) bool {
	return x < ScreenW && y < ScreenH && x+w > 0 && y+h > 0
}

// scene returns everything standing on the map, sorted back to front.
func (g *Game) scene() []drawable {
	s := sprites()
	z := g.zoomLevel()
	var ds []drawable
	tx0, ty0, tx1, ty1 := g.visibleTiles()
	for ty := ty0; ty <= ty1; ty++ {
		for tx := tx0; tx <= tx1; tx++ {
			r := g.tile(tx, ty).Res
			if r == nil || !g.seenTile(tx, ty) {
				continue
			}
			sp := s.resource(r)
			sx, sy := g.toView(tileCenter(tx), tileCenter(ty))
			x, y := sx-sp.ax*z, sy-sp.ay*z
			if g.visible(x, y, float64(sp.rgba.Rect.Dx())*z, float64(sp.rgba.Rect.Dy())*z) {
				ds = append(ds, drawable{depth: float64(tx+ty) + 1, x: x, y: y, sp: sp, res: r})
			}
		}
	}
	for _, b := range g.buildings {
		if !g.seenTile(b.TX, b.TY) && b.Team != g.human {
			continue
		}
		sp := g.bldSprite(b)
		cx, cy := b.center()
		sx, sy := g.toView(cx, cy)
		x, y := sx-sp.ax*z, sy-sp.ay*z
		if g.visible(x, y-120*z, float64(sp.rgba.Rect.Dx())*z, (float64(sp.rgba.Rect.Dy())+120)*z) {
			depth := (cx + cy) / TileSize
			if b.Kind == Farm {
				depth = float64(b.TX + b.TY)
			}
			ds = append(ds, drawable{depth: depth, x: x, y: y, sp: sp, bld: b})
		}
	}
	for i := range g.corpses {
		c := &g.corpses[i]
		if !g.canSee(toTile(c.X), toTile(c.Y)) {
			continue
		}
		sx, sy := g.toView(c.X, c.Y)
		ds = append(ds, drawable{depth: (c.X+c.Y)/TileSize - 0.5, x: sx - ax(c.Kind)*z, y: sy - ay(c.Kind)*z, corpse: c})
	}
	for _, u := range g.units {
		if u.Inside != nil || !g.unitShown(u) {
			continue
		}
		sx, sy := g.toView(u.X, u.Y)
		fw, fh, uax, uay := unitAnchor(u.Kind)
		x, y := sx-uax*z, sy-uay*z
		if g.visible(x, y, fw*z, fh*z) {
			ds = append(ds, drawable{depth: (u.X + u.Y) / TileSize, x: x, y: y, unit: u})
		}
	}
	sort.SliceStable(ds, func(i, j int) bool { return ds[i].depth < ds[j].depth })
	return ds
}

// pick returns the front-most object under screen point (mx, my).
func (g *Game) pick(mx, my int) drawable {
	ds := g.scene()
	z := g.zoomLevel()
	fx, fy := float64(mx), float64(my)
	for i := len(ds) - 1; i >= 0; i-- {
		d := ds[i]
		switch {
		case d.corpse != nil:
			continue
		case d.unit != nil:
			// Units are small, so the click area never shrinks below a
			// comfortable size when zoomed out.
			fw, fh, uax, uay := unitAnchor(d.unit.Kind)
			cx, top, bottom := d.x+uax*z, d.y+(uay-30)*z, d.y+fh*z
			half := max(8, (fw/2-12)*z)
			if fx >= cx-half && fx < cx+half && fy >= min(top, bottom-16) && fy < bottom {
				return d
			}
		case d.sp.opaque(int((fx-d.x)/z), int((fy-d.y)/z)):
			return d
		}
	}
	return drawable{}
}
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (g *Game) selectedUnits() []*Unit {
	var s []*Unit
	for _, u := range g.units {
		if u.Selected {
			s = append(s, u)
		}
	}
	return s
}

func (g *Game) selectedVillagers() []*Unit {
	var s []*Unit
	for _, u := range g.units {
		if u.Selected && u.Kind == Villager {
			s = append(s, u)
		}
	}
	return s
}

func (g *Game) clearSelection() {
	for _, u := range g.units {
		u.Selected = false
	}
	g.selBld = nil
	g.page = 0
}

func (g *Game) visibleCenter() (float64, float64) {
	return g.screenToWorld(ScreenW/2, TopBarH+ViewH/2)
}

func dist(ax, ay, bx, by float64) float64 { return math.Hypot(ax-bx, ay-by) }
