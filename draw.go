package main

import (
	"bytes"
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"mini-empires/internal/art"
)

var (
	white = color.RGBA{255, 255, 255, 255}
	black = color.RGBA{0, 0, 0, 255}
	gold  = color.RGBA{240, 205, 110, 255}
	grey  = color.RGBA{170, 165, 155, 255}
	red   = color.RGBA{235, 90, 70, 255}
	green = color.RGBA{110, 220, 110, 255}
	// Panels are translucent so the map shows through. Colors are
	// premultiplied by alpha.
	panel  = color.RGBA{24, 19, 14, 165}
	panel2 = color.RGBA{42, 33, 23, 185}
	trim   = color.RGBA{150, 115, 60, 255}
)

var regularFont, boldFont *text.GoTextFaceSource

func init() {
	var err error
	if regularFont, err = text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF)); err != nil {
		panic(err)
	}
	if boldFont, err = text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF)); err != nil {
		panic(err)
	}
}

func face(size float64, bold bool) *text.GoTextFace {
	if bold {
		return &text.GoTextFace{Source: boldFont, Size: size}
	}
	return &text.GoTextFace{Source: regularFont, Size: size}
}

func drawText(dst *ebiten.Image, s string, x, y, size float64, c color.Color, bold bool) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face(size, bold), op)
}

// drawTextShadow draws text with a dark drop shadow, for text over the map.
func drawTextShadow(dst *ebiten.Image, s string, x, y, size float64, c color.Color, bold bool) {
	drawText(dst, s, x+1, y+1, size, black, bold)
	drawText(dst, s, x, y, size, c, bold)
}

func textWidth(s string, size float64, bold bool) float64 {
	w, _ := text.Measure(s, face(size, bold), 0)
	return w
}

func fillRect(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}

func strokeRect(dst *ebiten.Image, x, y, w, h, sw float64, c color.Color) {
	vector.StrokeRect(dst, float32(x), float32(y), float32(w), float32(h), float32(sw), c, false)
}

func line(dst *ebiten.Image, x0, y0, x1, y1, w float64, c color.Color) {
	vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), float32(w), c, true)
}

// box draws a bevelled AoE-style panel.
func box(dst *ebiten.Image, x, y, w, h float64) {
	fillRect(dst, x, y, w, h, panel)
	strokeRect(dst, x, y, w, h, 2, trim)
	strokeRect(dst, x+3, y+3, w-6, h-6, 1, color.RGBA{90, 70, 40, 255})
}

func hpBar(dst *ebiten.Image, cx, y, w, frac float64) {
	fillRect(dst, cx-w/2-1, y-1, w+2, 6, black)
	c := color.RGBA{60, 220, 60, 255}
	if frac < 0.35 {
		c = color.RGBA{230, 60, 40, 255}
	}
	fillRect(dst, cx-w/2, y, w*max(frac, 0), 4, c)
}

func drawAt(dst, img *ebiten.Image, x, y float64, cs *ebiten.ColorScale) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(x), math.Round(y))
	if cs != nil {
		op.ColorScale = *cs
	}
	dst.DrawImage(img, op)
}

// drawFit draws img scaled down to fit inside the w x h box at (x, y).
func drawFit(dst, img *ebiten.Image, x, y, w, h float64, cs *ebiten.ColorScale) {
	b := img.Bounds()
	k := min(w/float64(b.Dx()), h/float64(b.Dy()), 2)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(k, k)
	op.GeoM.Translate(x+(w-float64(b.Dx())*k)/2, y+(h-float64(b.Dy())*k)/2)
	if cs != nil {
		op.ColorScale = *cs
	}
	dst.DrawImage(img, op)
}

// diamond outlines a footprint of tiles on the ground.
func (g *Game) diamond(dst *ebiten.Image, r rect, c color.Color) {
	x0, y0, x1, y1 := r.pixels()
	pts := [][2]float64{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}, {x0, y0}}
	for i := range 4 {
		ax, ay := g.toView(pts[i][0], pts[i][1])
		bx, by := g.toView(pts[i+1][0], pts[i+1][1])
		line(dst, ax, ay, bx, by, 1.5, c)
	}
}

var frameCount int

// DrawWorld draws the map and everything on it.
// blit draws a world sprite with its top-left at (x, y), scaled by the zoom.
func (g *Game) blit(dst, img *ebiten.Image, x, y float64, cs *ebiten.ColorScale) {
	z := g.zoomLevel()
	op := &ebiten.DrawImageOptions{}
	if z < 0.99 {
		op.Filter = ebiten.FilterLinear
	}
	op.GeoM.Scale(z, z)
	op.GeoM.Translate(math.Round(x), math.Round(y))
	if cs != nil {
		op.ColorScale = *cs
	}
	dst.DrawImage(img, op)
}

// DrawWorld draws the map and everything on it.
func (g *Game) DrawWorld(screen *ebiten.Image) {
	frameCount++
	s := sprites()
	z := g.zoomLevel()
	screen.Fill(black)

	g.drawGround(screen)

	if b := g.selBld; b != nil {
		g.diamond(screen, b.rect(), white)
	}
	for _, u := range g.units {
		if u.Selected && u.Inside == nil {
			sx, sy := g.toView(u.X, u.Y)
			var cs ebiten.ColorScale
			if u.Team != g.human && g.human >= 0 {
				cs.Scale(1, 0.4, 0.4, 1)
			}
			g.blit(screen, s.ring, sx-17*z, sy-9*z, &cs)
		}
	}
	if m := g.marker; m.t > 0 {
		var cs ebiten.ColorScale
		if m.attack {
			cs.Scale(1, 0.25, 0.25, 1)
		} else {
			cs.Scale(0.35, 1, 0.35, 1)
		}
		sx, sy := g.toView(m.x, m.y)
		op := &ebiten.DrawImageOptions{ColorScale: cs}
		k := (0.4 + m.t) * z
		op.GeoM.Translate(-17, -9)
		op.GeoM.Scale(k, k)
		op.GeoM.Translate(sx, sy)
		screen.DrawImage(s.ring, op)
	}

	scene := g.scene()
	for _, d := range scene {
		switch {
		case d.corpse != nil:
			c := d.corpse
			var cs ebiten.ColorScale
			cs.ScaleAlpha(float32(min(1, c.t/3)))
			g.blit(screen, s.frames(c.Kind, c.Team, g.tierFor(c.Kind, c.Team))[unitDir(c.Facing)][art.UnitFrames-1], d.x, d.y, &cs)
		case d.unit != nil:
			g.drawUnit(screen, d.unit, d.x, d.y)
		case d.bld != nil:
			g.drawBuilding(screen, d.bld, d.sp, d.x, d.y)
		default:
			g.blit(screen, d.sp.img, d.x, d.y, nil)
		}
	}

	for _, e := range g.effects {
		dur := max(e.dur, arrowTime)
		f := 1 - e.t/dur
		wx, wy := e.x0+(e.x1-e.x0)*f, e.y0+(e.y1-e.y0)*f
		sx, sy := g.toView(wx, wy)
		arc := 24 + 6*math.Hypot(e.x1-e.x0, e.y1-e.y0)/TileSize
		sy -= (e.h*(1-f) + 16 + arc*math.Sin(f*math.Pi)) * z
		switch e.kind {
		case ShotStone, ShotBoulder:
			r := 3.0
			if e.kind == ShotBoulder {
				r = 5
			}
			r = max(1.5, r*z)
			vector.DrawFilledCircle(screen, float32(sx), float32(sy), float32(r), color.RGBA{105, 100, 92, 255}, true)
			vector.StrokeCircle(screen, float32(sx), float32(sy), float32(r), 1, color.RGBA{40, 36, 30, 255}, true)
		default:
			tx, ty := g.toView(e.x1, e.y1)
			dx, dy := tx-sx, ty-16*z-sy
			if l := math.Hypot(dx, dy); l > 0 {
				n := max(4, 9*z)
				line(screen, sx, sy, sx+dx/l*n, sy+dy/l*n, 1.5, color.RGBA{50, 35, 20, 255})
			}
		}
	}

	for _, d := range scene {
		if u := d.unit; u != nil && (u.Selected || u.HP < u.MaxHP) && u.Team != Gaia {
			hpBar(screen, d.x+ax(u.Kind)*z, d.y+(ay(u.Kind)-40)*z, max(10, 20*z), u.HP/u.MaxHP)
		}
		if b := d.bld; b != nil && (b == g.selBld || b.HP < b.def().HP && b.Built) {
			cx, cy := b.center()
			sx, sy := g.toView(cx, cy)
			hpBar(screen, sx, sy-b.def().Height*z, float64(b.def().Size)*24*z, b.HP/b.def().HP)
		}
	}
	if b := g.selBld; b != nil && b.HasRally {
		rx, ry := g.toView(b.RallyX, b.RallyY)
		line(screen, rx, ry, rx, ry-18, 2, color.RGBA{70, 50, 30, 255})
		fillRect(screen, rx+1, ry-18, 10, 7, teamColors[b.Team])
	}

	mx, my := ebiten.CursorPosition()
	if g.placing != NoBld && g.human >= 0 {
		wx, wy := g.screenToWorld(mx, my)
		tx, ty := g.ghostTile(wx, wy)
		def := &bldDefs[g.placing]
		sp := s.finished(g.placing, g.human, g.players[g.human].Age)
		spots := []pt{{tx, ty}}
		if def.Wall && g.wallDrag {
			spots = wallLine(g.wallStart, pt{toTile(wx), toTile(wy)})
		}
		half := float64(def.Size*TileSize) / 2
		for _, p := range spots {
			sx, sy := g.toView(float64(p.X*TileSize)+half, float64(p.Y*TileSize)+half)
			var cs ebiten.ColorScale
			if g.canPlace(g.placing, g.human, p.X, p.Y) && g.canAfford(g.human, def.Cost) {
				cs.Scale(0.6, 1, 0.6, 0.75)
			} else {
				cs.Scale(1, 0.35, 0.35, 0.75)
			}
			g.blit(screen, sp.img, sx-sp.ax*z, sy-sp.ay*z, &cs)
			g.diamond(screen, rect{p.X, p.Y, p.X + def.Size - 1, p.Y + def.Size - 1}, white)
		}
	}
	if g.dragging {
		x0, y0 := float64(min(g.dragX0, mx)), float64(min(g.dragY0, my))
		strokeRect(screen, x0, y0, math.Abs(float64(mx-g.dragX0)), math.Abs(float64(my-g.dragY0)), 1, white)
	}
}

// unitDir maps a world facing angle to one of the 8 sheet rows.
func unitDir(facing float64) int {
	d := int(math.Round((facing + math.Pi/4) / (math.Pi / 4)))
	return ((d % 8) + 8) % 8
}

func (g *Game) unitFrame(u *Unit) int {
	switch {
	case u.acting && u.def().Military:
		if u.cool > u.def().Cooldown-0.35 {
			return 6
		}
		return 5
	case u.acting:
		return 5 + int(u.anim*3)%2
	case u.moved:
		return 1 + int(u.anim*8)%4
	}
	return 0
}

func (g *Game) drawUnit(dst *ebiten.Image, u *Unit, x, y float64) {
	s := sprites()
	z := g.zoomLevel()
	frames := s.frames(u.Kind, u.Team, g.tierFor(u.Kind, u.Team))
	if u.Kind == Villager && u.Team < Gaia {
		frames = &s.vills[u.Team][boolIdx(u.Female)][villagerTask(u)]
	}
	g.blit(dst, frames[unitDir(u.Facing)][g.unitFrame(u)], x, y, nil)
	if u.Carry > 0 {
		g.blit(dst, s.icons[u.CarryKind], x+(ax(u.Kind)+2)*z, y+(ay(u.Kind)-28)*z, nil)
	}
}

func (g *Game) drawBuilding(dst *ebiten.Image, b *Building, sp *Sprite, x, y float64) {
	g.blit(dst, sp.img, x, y, nil)
	if b.Built || b.Kind == Farm || b.def().Wall {
		return
	}
	// Show the finished building rising out of its foundation.
	z := g.zoomLevel()
	full := sprites().finished(b.Kind, min(b.Team, 1), g.players[b.Team].Age)
	cx, cy := b.center()
	sx, sy := g.toView(cx, cy)
	w, h := full.rgba.Rect.Dx(), full.rgba.Rect.Dy()
	top := int(float64(h) * (1 - b.Progress))
	if top >= h {
		return
	}
	part := full.img.SubImage(imageRect(0, top, w, h)).(*ebiten.Image)
	var cs ebiten.ColorScale
	cs.ScaleAlpha(0.9)
	g.blit(dst, part, sx-full.ax*z, sy-(full.ay-float64(top))*z, &cs)
}

func clock(t float64) string {
	s := int(t)
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// resourceBar draws one team's stockpile, population and age.
func (g *Game) resourceBar(dst *ebiten.Image, team int, x, y float64) float64 {
	s := sprites()
	p := &g.players[team]
	for k := range numRes {
		drawAt(dst, s.icons[k], x, y+3, nil)
		drawText(dst, itoa(p.Res[k]), x+16, y, 14, white, false)
		x += 70
	}
	pop := fmt.Sprintf("%d/%d", g.pop(team), g.popCap(team))
	c := white
	if g.pop(team) >= g.popCap(team) {
		c = red
	}
	drawText(dst, "Pop "+pop, x, y, 14, c, false)
	x += 80
	drawText(dst, ageNames[p.Age], x, y, 14, gold, true)
	return x + 100
}

func (g *Game) drawTopBar(dst *ebiten.Image) {
	fillRect(dst, 0, 0, ScreenW, TopBarH, panel)
	line(dst, 0, TopBarH, ScreenW, TopBarH, 2, trim)
	if g.human >= 0 {
		g.resourceBar(dst, g.human, 10, 3)
	} else {
		fillRect(dst, 0, 0, 6, TopBarH, teamColors[0])
		g.resourceBar(dst, 0, 12, 3)
		fillRect(dst, ScreenW/2, 0, 6, TopBarH, teamColors[1])
		g.resourceBar(dst, 1, ScreenW/2+12, 3)
	}
	t := clock(g.time)
	drawText(dst, t, ScreenW-textWidth(t, 14, true)-10, 3, 14, white, true)
	m := fmt.Sprintf("%s  #%d", mapNames[g.mapStyle], uint32(g.seed)%100000)
	drawText(dst, m, ScreenW-textWidth(t, 14, true)-textWidth(m, 12, false)-24, 5, 12, grey, false)
}

// drawScores draws both teams' scores, broken down like AoE's score screen.
func (g *Game) drawScores(dst *ebiten.Image, x, y float64) {
	if !window(dst, "score", "Score", x, y, 250, 70) {
		return
	}
	drawText(dst, "Total   Eco   Mil   Tech", x+78, y+6, 12, grey, false)
	for t := range 2 {
		sc := g.score(t)
		yy := y + 26 + float64(t)*20
		fillRect(dst, x+10, yy+3, 10, 10, teamColors[t])
		drawText(dst, teamNames[t], x+26, yy, 13, white, false)
		drawText(dst, fmt.Sprintf("%5.0f %5.0f %5.0f %5.0f", sc.Total, sc.Economy, sc.Military, sc.Tech), x+88, yy, 13, white, false)
	}
}

func (g *Game) drawBottomPanel(dst *ebiten.Image) {
	by := float64(ScreenH - bottomH())
	fillRect(dst, 0, by, ScreenW, float64(bottomH()), panel)
	line(dst, 0, by, ScreenW, by, 2, trim)
	bx, bty := float64(ScreenW-titleH-6), by+4
	minButton(dst, bx, bty, hudMin["bottom"])
	hudButtons = append(hudButtons, hudButton{"bottom", bx, bty, titleH - 4, titleH - 4})
	if hudMin["bottom"] {
		drawText(dst, "Commands and minimap (minimized)", 12, by+5, 13, grey, false)
		g.drawZoomButtons(dst)
		if g.msgT > 0 {
			drawText(dst, g.msg, 270, by-24, 15, gold, true)
		}
		return
	}

	for _, b := range g.panelButtons() {
		a := animOf("cmd" + keyNames[indexOfKey(b.Key)])
		k := 1 - 0.1*a.press
		s := buttonSize * k
		x, y := b.x+(buttonSize-s)/2, b.y+(buttonSize-s)/2
		fillRect(dst, x, y, s, s, lerpColor(panel2, color.RGBA{90, 72, 44, 230}, a.hover))
		var cs *ebiten.ColorScale
		if !b.Enabled {
			cs = &ebiten.ColorScale{}
			cs.Scale(0.35, 0.35, 0.35, 1)
		} else if a.hover > 0.05 {
			cs = &ebiten.ColorScale{}
			f := float32(1 + 0.25*a.hover)
			cs.Scale(f, f, f, 1)
		}
		if b.Icon != nil {
			drawFit(dst, b.Icon, x+3, y+3, s-6, s-6, cs)
		}
		strokeRect(dst, x, y, s, s, 1.5, lerpColor(trim, gold, a.hover))
		drawTextShadow(dst, keyNames[indexOfKey(b.Key)], x+3, y+1, 10, white, true)
	}

	g.drawSelectionInfo(dst, 270, by+10)

	if h := g.hover; h != nil {
		lines := []string{h.Name}
		if h.Cost.total() > 0 {
			lines = append(lines, "Cost: "+h.Cost.String())
		}
		if h.Desc != "" {
			lines = append(lines, h.Desc)
		}
		w := 0.0
		for _, l := range lines {
			w = max(w, textWidth(l, 13, false))
		}
		if h.Reason != "" {
			w = max(w, textWidth(h.Reason, 13, false))
		}
		hgt := float64(len(lines))*17 + 12
		if h.Reason != "" {
			hgt += 17
		}
		x, y := 12.0, by-hgt-6
		box(dst, x, y, w+20, hgt)
		for i, l := range lines {
			c := white
			if i == 0 {
				c = gold
			}
			drawText(dst, l, x+10, y+6+float64(i)*17, 13, c, i == 0)
		}
		if h.Reason != "" {
			drawText(dst, h.Reason, x+10, y+6+float64(len(lines))*17, 13, red, false)
		}
	}

	if g.minimap == nil {
		g.minimap = ebiten.NewImage(MiniSize, MiniSize)
	}
	if g.minimapT -= 1.0 / 60; g.minimapT <= 0 {
		g.minimapT = 0.25
		g.renderMinimap()
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.ColorScale.ScaleAlpha(0.85)
	op.GeoM = miniGeo()
	dst.DrawImage(g.minimap, op)
	geo := miniGeo()
	corner := func(x, y float64) (float64, float64) {
		return geo.Apply(x*MiniSize/worldW(), y*MiniSize/worldH())
	}
	var ms [4][2]float64
	for i, c := range [][2]int{{0, TopBarH}, {ScreenW, TopBarH}, {ScreenW, ScreenH - bottomH()}, {0, ScreenH - bottomH()}} {
		wx, wy := g.screenToWorld(c[0], c[1])
		ms[i][0], ms[i][1] = corner(wx, wy)
	}
	for i := range 4 {
		j := (i + 1) % 4
		line(dst, ms[i][0], ms[i][1], ms[j][0], ms[j][1], 1, white)
	}
	edge := [][2]float64{{0, 0}, {worldW(), 0}, {worldW(), worldH()}, {0, worldH()}}
	for i := range 4 {
		ax, ay := corner(edge[i][0], edge[i][1])
		bx, bz := corner(edge[(i+1)%4][0], edge[(i+1)%4][1])
		line(dst, ax, ay, bx, bz, 1.5, trim)
	}
	g.drawZoomButtons(dst)
	if g.human >= 0 && g.players[g.human].alertT > 0 && frameCount/20%2 == 0 {
		p := &g.players[g.human]
		x, y := corner(p.alertX, p.alertY)
		strokeRect(dst, x-6, y-6, 12, 12, 2, red)
	}
}

func indexOfKey(k ebiten.Key) int {
	for i, g := range gridKeys {
		if g == k {
			return i
		}
	}
	return 0
}

func (g *Game) drawSelectionInfo(dst *ebiten.Image, x, y float64) {
	s := sprites()
	if g.msgT > 0 {
		drawText(dst, g.msg, x, float64(ScreenH-bottomH()-26), 15, gold, true)
	}
	if b := g.selBld; b != nil {
		def := b.def()
		drawFit(dst, g.bldSprite(b).img, x, y, 64, 64, nil)
		drawText(dst, def.Name, x+74, y, 16, gold, true)
		drawText(dst, fmt.Sprintf("%s    HP %.0f/%.0f", teamNames[b.Team], b.HP, def.HP), x+74, y+22, 13, white, false)
		line2 := def.Desc
		switch {
		case !b.Built:
			line2 = fmt.Sprintf("Under construction: %.0f%%", b.Progress*100)
		case b.Kind == Farm:
			line2 = fmt.Sprintf("Food left: %d", b.Food)
		}
		drawText(dst, line2, x+74, y+40, 13, grey, false)
		for i, q := range b.Queue {
			qx, qy := queueSlot(i)
			fillRect(dst, qx, qy, queueSize, queueSize, panel2)
			var icon *ebiten.Image
			if q.Age > 0 {
				icon = s.buttons["age"+itoa(q.Age)]
			} else {
				icon = s.unitIcon(q.Unit, b.Team, g.tierFor(q.Unit, b.Team))
			}
			drawFit(dst, icon, qx+2, qy+2, queueSize-4, queueSize-4, nil)
			strokeRect(dst, qx, qy, queueSize, queueSize, 1, trim)
			if i == 0 {
				f := min(b.TrainT/q.time(), 1)
				fillRect(dst, qx, qy+queueSize+2, queueSize*f, 4, green)
			}
		}
		if len(b.Queue) > 0 {
			drawText(dst, "Click to cancel", queueX, ScreenH-70, 11, grey, false)
		}
		return
	}
	sel := g.selectedUnits()
	if len(sel) == 0 {
		hint := "Select units with left click or drag. Right click to give orders."
		if g.human < 0 {
			hint = "Click a unit or building to inspect it."
		}
		drawText(dst, hint, x, y+4, 13, grey, false)
		return
	}
	if len(sel) == 1 {
		u := sel[0]
		drawFit(dst, s.unitIcon(u.Kind, u.Team, g.tierFor(u.Kind, u.Team)), x, y, 56, 56, nil)
		drawText(dst, g.unitName(u), x+66, y, 16, gold, true)
		drawText(dst, fmt.Sprintf("%s    HP %.0f/%.0f    Attack %.0f", teamNames[u.Team], u.HP, u.MaxHP, g.attackOf(u)), x+66, y+22, 13, white, false)
		status := orderNames[u.Order]
		if u.Order == OrdGather || u.Order == OrdReturn {
			status += " " + resNames[u.GatherKnd]
		}
		if u.Carry > 0 {
			status += fmt.Sprintf("    Carrying %d %s", u.Carry, resNames[u.CarryKind])
		}
		drawText(dst, status, x+66, y+40, 13, grey, false)
		return
	}
	drawText(dst, fmt.Sprintf("%d units selected", len(sel)), x, y, 15, gold, true)
	for i, u := range sel {
		if i >= 24 {
			break
		}
		ix, iy := x+float64(i%12)*34, y+24+float64(i/12)*40
		fillRect(dst, ix, iy, 30, 36, panel2)
		drawFit(dst, s.unitIcon(u.Kind, u.Team, g.tierFor(u.Kind, u.Team)), ix, iy, 30, 32, nil)
		fillRect(dst, ix, iy+34, 30*u.HP/u.MaxHP, 2, green)
	}
}

func (g *Game) renderMinimap() {
	m := g.minimap
	sc := float64(MiniSize) / float64(MapW)
	terr := [...]color.RGBA{TerGrass: {80, 130, 55, 255}, TerDryGrass: {140, 140, 70, 255}, TerDirt: {130, 105, 70, 255},
		TerSand: {200, 185, 130, 255}, TerWater: {50, 90, 160, 255}, TerForest: {30, 75, 35, 255}}
	mini := [numRes]color.RGBA{Food: {200, 60, 90, 255}, Wood: {25, 70, 30, 255}, Gold: {240, 200, 40, 255}, Stone: {160, 160, 160, 255}}
	for y := range MapH {
		for x := range MapW {
			t := g.tile(x, y)
			c := terr[t.Ter]
			if t.Res != nil {
				c = mini[t.Res.Kind]
			}
			switch {
			case !g.seenTile(x, y):
				c = black
			case !g.canSee(x, y):
				c = color.RGBA{c.R / 2, c.G / 2, c.B / 2, 255}
			}
			fillRect(m, float64(x)*sc, float64(y)*sc, sc+0.5, sc+0.5, c)
		}
	}
	for _, b := range g.buildings {
		if !g.seenTile(b.TX, b.TY) && b.Team != g.human {
			continue
		}
		sz := float64(b.def().Size)
		fillRect(m, float64(b.TX)*sc, float64(b.TY)*sc, sz*sc, sz*sc, teamColors[b.Team])
	}
	ps := float64(MiniSize) / float64(MapW*TileSize)
	for _, u := range g.units {
		if u.Inside != nil || !g.unitShown(u) {
			continue
		}
		c := teamColors[u.Team]
		fillRect(m, u.X*ps-1.5, u.Y*ps-1.5, 3, 3, color.RGBA{min(255, c.R+60), min(255, c.G+60), min(255, c.B+60), 255})
	}
}

func boolIdx(b bool) int {
	if b {
		return 1
	}
	return 0
}

// villagerTask picks the sheet that shows the right tool for what a
// villager is doing.
func villagerTask(u *Unit) art.VillagerTask {
	switch u.Order {
	case OrdBuild:
		return art.TaskBuild
	case OrdAttack:
		return art.TaskHunt
	case OrdGather, OrdReturn:
		switch {
		case u.GatherKnd == Wood:
			return art.TaskWood
		case u.GatherKnd == Gold || u.GatherKnd == Stone:
			return art.TaskMine
		case u.FarmB != nil:
			return art.TaskFarm
		case u.GatherKnd == Food:
			return art.TaskForage
		}
	}
	return art.TaskIdle
}

func (g *Game) drawZoomButtons(dst *ebiten.Image) {
	drawButtons(dst, g.zoomUIButtons())
}
