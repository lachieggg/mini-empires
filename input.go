package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const edgeScroll = 24

// edgeFactor returns how far into the scroll zone at one window edge the
// cursor is, from 0 to 1. A cursor that has slipped just past the edge of
// the window still counts, so scrolling does not stop when it leaves.
func edgeFactor(pos, size int) (lo, hi float64) {
	if pos < edgeScroll {
		lo = min(1, float64(edgeScroll-pos)/edgeScroll)
	}
	if pos >= size-edgeScroll {
		hi = min(1, float64(pos-(size-edgeScroll))/edgeScroll)
	}
	return lo, hi
}

// handleCamera scrolls with the arrow keys, the screen edges, dragging with
// the middle mouse button and sideways wheel or trackpad motion, and zooms
// with the mouse wheel.
func (g *Game) handleCamera() {
	mx, my := ebiten.CursorPosition()
	z := g.zoomLevel()
	speed := 1100 * dt / z
	near := ebiten.IsFocused() && mx > -300 && my > -300 && mx < ScreenW+300 && my < ScreenH+300
	if near {
		l, r := edgeFactor(mx, ScreenW)
		u, d := edgeFactor(my, ScreenH)
		g.camX += (r - l) * speed
		g.camY += (d - u) * speed
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.camX -= speed
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.camX += speed
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.camY -= speed
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.camY += speed
	}
	if wx, wy := ebiten.Wheel(); wx != 0 || wy != 0 {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			g.camX -= wx * 30 / z
			g.camY -= wy * 30 / z
		} else {
			g.camX -= wx * 30 / z
			if wy != 0 {
				g.zoomAt(math.Pow(1.15, min(max(wy, -3), 3)), mx, my)
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		g.zoomAt(1.25, ScreenW/2, TopBarH+ViewH/2)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		g.zoomAt(0.8, ScreenW/2, TopBarH+ViewH/2)
	}
	if ebiten.IsKeyPressed(ebiten.KeyAlt) {
		for i, z := range zoomLevels() {
			if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
				if i == 0 {
					g.fitMap()
				} else {
					g.setZoom(z, ScreenW/2, TopBarH+ViewH/2)
				}
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		g.fitMap()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		vx, vy := g.visibleCenter()
		g.zoom = 1
		g.centerOn(vx, vy)
	}
	clickButtons(g.zoomUIButtons())
	// Drag to pan: middle button always, right button in play (a right
	// click without moving still gives an order), and left button when
	// watching, where there is nothing to command.
	buttons := []ebiten.MouseButton{ebiten.MouseButtonMiddle, ebiten.MouseButtonRight}
	if g.human < 0 {
		buttons = append(buttons, ebiten.MouseButtonLeft)
	}
	for _, b := range buttons {
		if inpututil.IsMouseButtonJustPressed(b) {
			ok := !inMini(mx, my) && (my < ScreenH-bottomH() || b == ebiten.MouseButtonMiddle)
			g.pan = panState{on: ok, button: b, x: mx, y: my, x0: mx, y0: my}
		}
	}
	if p := &g.pan; p.on {
		if !ebiten.IsMouseButtonPressed(p.button) {
			p.on = false
		} else {
			if abs(mx-p.x0)+abs(my-p.y0) > 6 {
				p.moved = true
			}
			if p.moved || p.button == ebiten.MouseButtonMiddle {
				g.camX -= float64(mx-p.x) / z
				g.camY -= float64(my-p.y) / z
			}
			p.x, p.y = mx, my
		}
	}
	if inMini(mx, my) && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !g.dragging {
		wx, wy, _ := miniToWorld(mx, my)
		g.centerOn(wx, wy)
	}
	g.clampCam()
}

func inMini(mx, my int) bool {
	_, _, ok := miniToWorld(mx, my)
	return ok && !hudMin["bottom"] && my >= ScreenH-bottomH()
}

func inView(my int) bool { return my >= TopBarH && my < ScreenH-bottomH() }

// handleSelection lets a viewer select and inspect anything, without
// giving orders. Left drag pans the map there, so a click selects.
func (g *Game) handleSelection(team int) {
	mx, my := ebiten.CursorPosition()
	if team < 0 {
		if inView(my) && g.clicked(ebiten.MouseButtonLeft) {
			g.finishSelect(mx, my, mx, my, team)
		}
		return
	}
	if inView(my) && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.dragging, g.dragX0, g.dragY0 = true, mx, my
	}
	if g.dragging && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		g.dragging = false
		g.finishSelect(g.dragX0, g.dragY0, mx, min(max(my, TopBarH), ScreenH-bottomH()), team)
	}
}

func (g *Game) handleInput() {
	g.handleCamera()
	if g.over() {
		return
	}
	mx, my := ebiten.CursorPosition()
	team := g.human

	if inMini(mx, my) && g.clicked(ebiten.MouseButtonRight) {
		wx, wy, _ := miniToWorld(mx, my)
		g.command(wx, wy, drawable{})
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		switch {
		case g.placing != NoBld:
			g.placing = NoBld
		case g.page != 0:
			g.page = 0
		default:
			g.clearSelection()
		}
	}

	if g.placing != NoBld {
		if g.clicked(ebiten.MouseButtonRight) {
			g.placing = NoBld
		} else if bldDefs[g.placing].Wall {
			g.handleWallDrag(mx, my)
		} else if inView(my) && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			wx, wy := g.screenToWorld(mx, my)
			tx, ty := g.ghostTile(wx, wy)
			g.placeBuilding(g.placing, tx, ty)
			if !ebiten.IsKeyPressed(ebiten.KeyShift) {
				g.placing = NoBld
			}
		}
		return
	}

	buttons := g.panelButtons()
	g.hover = nil
	for i := range buttons {
		b := &buttons[i]
		over := !hudMin["bottom"] && float64(mx) >= b.x && float64(mx) < b.x+buttonSize && float64(my) >= b.y && float64(my) < b.y+buttonSize
		if over {
			g.hover = b
		}
		anim := animate("cmd"+keyNames[indexOfKey(b.Key)], over)
		if (over && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)) || inpututil.IsKeyJustPressed(b.Key) {
			anim.press = 1
			if b.Enabled {
				b.Do()
			} else if b.Reason != "" {
				g.flash(b.Reason)
			}
			return
		}
	}
	if i := g.queueSlotAt(mx, my); i >= 0 && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.cancelQueue(g.selBld, i)
		return
	}

	g.handleSelection(team)
	if inView(my) && g.clicked(ebiten.MouseButtonRight) {
		wx, wy := g.screenToWorld(mx, my)
		g.command(wx, wy, g.pick(mx, my))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPeriod) {
		g.selectIdleVillager()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		for _, u := range g.selectedUnits() {
			u.HP = 0
		}
	}
	g.handleGroups()
}

// handleGroups implements control groups: Ctrl+digit stores the selection,
// the digit recalls it and pressing it twice centers the camera on it.
func (g *Game) handleGroups() {
	if g.lastGroupT > 0 {
		g.lastGroupT -= dt
	}
	if ebiten.IsKeyPressed(ebiten.KeyAlt) {
		return // Alt+digit picks a zoom level instead.
	}
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	for d := range 10 {
		if !inpututil.IsKeyJustPressed(ebiten.Key0 + ebiten.Key(d)) {
			continue
		}
		if ctrl {
			g.groups[d] = g.selectedUnits()
			g.flash("Group " + itoa(d) + " set")
			return
		}
		alive := g.groups[d][:0]
		for _, u := range g.groups[d] {
			if !u.dead {
				alive = append(alive, u)
			}
		}
		g.groups[d] = alive
		if len(alive) == 0 {
			return
		}
		g.clearSelection()
		for _, u := range alive {
			u.Selected = true
		}
		if g.lastGroup == d && g.lastGroupT > 0 {
			g.centerOn(alive[0].X, alive[0].Y)
		}
		g.lastGroup, g.lastGroupT = d, 0.4
	}
}

func (g *Game) ghostTile(wx, wy float64) (int, int) {
	s := bldDefs[g.placing].Size
	return toTile(wx) - s/2, toTile(wy) - s/2
}

func (g *Game) startPlacing(k BldKind) {
	if ok, why := g.bldAvailable(g.human, k); !ok {
		g.flash(why)
		return
	}
	if !g.canAfford(g.human, bldDefs[k].Cost) {
		g.flash("Not enough resources for " + bldDefs[k].Name + " (" + bldDefs[k].Cost.String() + ")")
		return
	}
	g.placing = k
}

func (g *Game) canPlace(k BldKind, team, tx, ty int) bool {
	s := bldDefs[k].Size
	for y := ty; y < ty+s; y++ {
		for x := tx; x < tx+s; x++ {
			if g.blocked(x, y) || g.buildingAtTile(x, y) != nil || g.unitOnTile(x, y, 1-team) {
				return false
			}
		}
	}
	return true
}

func (g *Game) placeBuilding(k BldKind, tx, ty int) {
	team := g.human
	def := &bldDefs[k]
	if !g.canAfford(team, def.Cost) {
		g.flash("Not enough resources")
		return
	}
	if !g.canPlace(k, team, tx, ty) {
		g.flash("Can't build there")
		return
	}
	g.pay(team, def.Cost)
	b := g.addBuilding(k, team, tx, ty, false)
	if !def.Walkable {
		for _, u := range g.units {
			if g.buildingAtTile(toTile(u.X), toTile(u.Y)) == b {
				u.X, u.Y = g.spawnSpot(b.rect())
			}
		}
	}
	for _, v := range g.selectedVillagers() {
		g.orderBuild(v, b)
	}
}

func (g *Game) finishSelect(x0, y0, x1, y1, team int) {
	shift := ebiten.IsKeyPressed(ebiten.KeyShift)
	if !shift {
		g.clearSelection()
	}
	if abs(x1-x0) < 5 && abs(y1-y0) < 5 {
		h := g.pick(x1, y1)
		if u := h.unit; u != nil && (team < 0 || u.Team == team) {
			u.Selected = !shift || !u.Selected
			g.selBld = nil
			return
		}
		if h.bld != nil && !shift {
			g.selBld = h.bld
		}
		return
	}
	ax, ay := float64(min(x0, x1)), float64(min(y0, y1))
	bx, by := float64(max(x0, x1)), float64(max(y0, y1))
	for _, u := range g.units {
		if u.Inside != nil {
			continue
		}
		sx, sy := g.toScreen(u.X, u.Y)
		sy -= 14
		if (team < 0 && u.Team < Gaia || u.Team == team) && sx >= ax && sx <= bx && sy >= ay && sy <= by {
			u.Selected = true
			g.selBld = nil
		}
	}
}

// command gives the selection a context order for world point (wx, wy).
// h is the object under the cursor; when it is empty the tile at the point
// is used instead.
func (g *Game) command(wx, wy float64, h drawable) {
	team := g.human
	var sel []*Unit
	for _, u := range g.selectedUnits() {
		if u.Team == team {
			sel = append(sel, u)
		}
	}
	tx, ty := toTile(wx), toTile(wy)
	if h.unit == nil && h.bld == nil && h.res == nil {
		h.bld, h.res = g.buildingAtTile(tx, ty), g.resAt(tx, ty)
	}
	if len(sel) == 0 {
		if b := g.selBld; b != nil && b.Team == team && len(b.def().Trains) > 0 {
			b.HasRally, b.RallyX, b.RallyY, b.RallyRes = true, wx, wy, h.res
			g.marker = Marker{wx, wy, 0.5, false}
		}
		return
	}
	if e := h.unit; e != nil && e.Team != team {
		for _, u := range sel {
			if e.Team != Gaia || u.Kind == Villager || u.def().Military {
				g.orderAttackUnit(u, e)
			}
		}
		g.marker = Marker{e.X, e.Y, 0.5, true}
		return
	}
	b := h.bld
	if b != nil && b.Team != team {
		for _, u := range sel {
			g.orderAttackBld(u, b)
		}
		g.marker = Marker{wx, wy, 0.5, true}
		return
	}
	g.marker = Marker{wx, wy, 0.5, false}
	var movers []*Unit
	for _, u := range sel {
		switch {
		case u.Kind == Villager && h.res != nil:
			g.orderGather(u, h.res)
		case u.Kind == Villager && b != nil && !b.Built:
			g.orderBuild(u, b)
		case u.Kind == Villager && b != nil && b.Kind == Farm:
			g.orderFarm(u, b)
		case u.Kind == Villager && b != nil && u.Carry > 0 && b.def().Drops[u.CarryKind]:
			g.orderReturn(u, b)
		default:
			movers = append(movers, u)
		}
	}
	cols := int(math.Ceil(math.Sqrt(float64(len(movers)))))
	for i, u := range movers {
		ox := float64(i%cols) - float64(cols-1)/2
		oy := float64(i/cols) - float64(cols-1)/2
		g.orderMove(u, wx+ox*20, wy+oy*20)
	}
}

func (g *Game) selectIdleVillager() {
	var idle []*Unit
	for _, u := range g.units {
		if u.Team == g.human && u.Kind == Villager && u.Order == OrdIdle {
			idle = append(idle, u)
		}
	}
	if len(idle) == 0 {
		return
	}
	g.idleIdx = (g.idleIdx + 1) % len(idle)
	g.clearSelection()
	u := idle[g.idleIdx]
	u.Selected = true
	g.centerOn(u.X, u.Y)
}

// panState tracks a mouse button held down to drag the map.
type panState struct {
	on, moved bool
	button    ebiten.MouseButton
	x, y      int
	x0, y0    int
}

// clicked reports whether button b was just released without the map being
// dragged, which counts as a click.
func (g *Game) clicked(b ebiten.MouseButton) bool {
	return inpututil.IsMouseButtonJustReleased(b) && !(g.pan.button == b && g.pan.moved)
}

// wallLine returns the tiles of a straight wall from a to b: horizontal,
// vertical or diagonal, whichever is closest to the drag.
func wallLine(a, b pt) []pt {
	dx, dy := b.X-a.X, b.Y-a.Y
	switch {
	case abs(dx) > 2*abs(dy):
		dy = 0
	case abs(dy) > 2*abs(dx):
		dx = 0
	default:
		n := max(abs(dx), abs(dy))
		dx, dy = n*sign(dx), n*sign(dy)
	}
	n := max(abs(dx), abs(dy))
	pts := make([]pt, 0, n+1)
	for i := 0; i <= n; i++ {
		pts = append(pts, pt{a.X + sign(dx)*i, a.Y + sign(dy)*i})
	}
	return pts
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// handleWallDrag places a line of wall foundations between where the left
// button was pressed and where it was released.
func (g *Game) handleWallDrag(mx, my int) {
	wx, wy := g.screenToWorld(mx, my)
	here := pt{toTile(wx), toTile(wy)}
	if inView(my) && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.wallDrag, g.wallStart = true, here
	}
	if !g.wallDrag || !inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		return
	}
	g.wallDrag = false
	k, team := g.placing, g.human
	placed := 0
	for _, p := range wallLine(g.wallStart, here) {
		if !g.canAfford(team, bldDefs[k].Cost) {
			g.flash("Not enough resources for more wall")
			break
		}
		if g.canPlace(k, team, p.X, p.Y) {
			g.placeBuilding(k, p.X, p.Y)
			placed++
		}
	}
	if placed > 0 && !ebiten.IsKeyPressed(ebiten.KeyShift) {
		g.placing = NoBld
	}
}

// On-screen zoom buttons, to the left of the minimap.

const zoomBtnSize = 28

type zoomButton struct {
	label string
	tip   string
	x, y  float64
	do    func(g *Game)
}

func zoomButtons() []zoomButton {
	ox, oy := miniOrigin()
	x := float64(ox) - zoomBtnSize - 8
	y := float64(oy) - 4
	if hudMin["bottom"] {
		// Lay the buttons along the minimized strip instead.
		x, y = ScreenW-160, float64(ScreenH-zoomBtnSize+2)
		return []zoomButton{
			{"+", "Zoom in (PageUp, wheel, Alt+1-5 for set levels)", x, y, func(g *Game) { g.zoomAt(1.25, ScreenW/2, TopBarH+ViewH/2) }},
			{"-", "Zoom out (PageDown, wheel, Alt+1-5 for set levels)", x + zoomBtnSize + 4, y, func(g *Game) { g.zoomAt(0.8, ScreenW/2, TopBarH+ViewH/2) }},
			{"Fit", "Show the whole map (Home)", x + 2*(zoomBtnSize+4), y, func(g *Game) { g.fitMap() }},
		}
	}
	return []zoomButton{
		{"+", "Zoom in (PageUp, wheel, Alt+1-5 for set levels)", x, y, func(g *Game) { g.zoomAt(1.25, ScreenW/2, TopBarH+ViewH/2) }},
		{"-", "Zoom out (PageDown, wheel, Alt+1-5 for set levels)", x, y + zoomBtnSize + 4, func(g *Game) { g.zoomAt(0.8, ScreenW/2, TopBarH+ViewH/2) }},
		{"Fit", "Show the whole map (Home)", x, y + 2*(zoomBtnSize+4), func(g *Game) { g.fitMap() }},
	}
}

func inRect(mx, my int, x, y, w, h float64) bool {
	return float64(mx) >= x && float64(mx) < x+w && float64(my) >= y && float64(my) < y+h
}

// zoomUIButtons are the zoom buttons as animated buttons.
func (g *Game) zoomUIButtons() []uiButton {
	var bs []uiButton
	for _, b := range zoomButtons() {
		bs = append(bs, uiButton{id: "zoom" + b.label, label: b.label, tip: b.tip, x: b.x, y: b.y, w: zoomBtnSize, h: zoomBtnSize, do: func() { b.do(g) }})
	}
	return bs
}
