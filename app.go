package main

import (
	"fmt"
	"image/color"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type mode int

const (
	modeMenu mode = iota
	modePlay
	modeWatch
)

type menuItem struct {
	label string
	do    func()
	// A dropdown has options, the index shown, and choose to pick one.
	options  []string
	selected int
	choose   func(int)
}

var sizeNames = map[int]string{32: "tiny", 64: "small", 96: "medium", 128: "large", 192: "huge", 256: "giant"}

func sizeLabel(n int) string { return fmt.Sprintf("%d x %d  (%s)", n, n, sizeNames[n]) }

type App struct {
	mode       mode
	game       *Game
	trainer    *Trainer
	paused     bool
	speed      int
	overT      float64
	showScores bool
	follow     bool
	watchPol   [2]*netPolicy
	vsName     string
	vsPolicy   Policy
	stepAcc    float64
	mapStyle   int
	sizeIdx    int
	fogIdx     int
	dropOpen   int
	dropHover  int

	music     *soundtrack
	menuHover int
}

func NewApp() *App {
	a := &App{trainer: NewTrainer(policyPath()), showScores: true, menuHover: -1, dropOpen: -1, sizeIdx: 2, fogIdx: 1}
	a.music = newSoundtrack()
	return a
}

func (a *App) Layout(int, int) (int, int) { return ScreenW, ScreenH }

func (a *App) startPlay(p Policy, name string) {
	a.trainer.Stop()
	setMapSize(mapSizes[a.sizeIdx])
	opts := GameOptions{Seed: time.Now().UnixNano(), Human: 0, Visual: true, Map: a.mapStyle, Fog: a.fogIdx == 0}
	opts.Policies[1] = p
	a.game = NewGame(opts)
	a.mode, a.paused, a.overT, a.vsName, a.vsPolicy = modePlay, false, 0, name, p
	a.speed = 0
}

func (a *App) startWatch() {
	a.trainer.Stop()
	setMapSize(mapSizes[a.sizeIdx])
	a.trainer.Start()
	a.newWatchGame()
	a.mode, a.paused = modeWatch, false
	a.speed = 4
}

func (a *App) newWatchGame() {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	opts := GameOptions{Seed: rng.Int63(), Human: -1, Visual: true, TimeLimit: trainTimeLimit(), Map: a.mapStyle}
	for t := range 2 {
		a.watchPol[t] = &netPolicy{get: a.trainer.Net, rng: rng, show: true}
		opts.Policies[t] = a.watchPol[t]
	}
	a.game = NewGame(opts)
	a.overT = 0
	cx, cy := float64(MapW*TileSize)/2, float64(MapH*TileSize)/2
	a.game.centerOn(cx, cy)
}

func (a *App) menuItems() []menuItem {
	learned := a.trainer.Net()
	var styles, sizes []string
	for _, n := range mapNames {
		styles = append(styles, n)
	}
	for _, n := range mapSizes {
		sizes = append(sizes, sizeLabel(n))
	}
	return []menuItem{
		{label: "Play vs Scripted AI", do: func() { a.startPlay(&scripted{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}, "Scripted AI") }},
		{label: fmt.Sprintf("Play vs Learned AI  (%d updates)", learned.Updates()), do: func() {
			a.startPlay(&netPolicy{get: a.trainer.Net, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}, "Learned AI")
		}},
		{label: "Watch AI Self-Play  (trains live)", do: a.startWatch},
		{label: "Map style", options: styles, selected: a.mapStyle, choose: func(i int) { a.mapStyle = i }},
		{label: "Map size", options: sizes, selected: a.sizeIdx, choose: func(i int) { a.sizeIdx = i }},
		{label: "Fog of war", options: []string{"On", "Off"}, selected: a.fogIdx, choose: func(i int) { a.fogIdx = i }},
		{label: "Reset Learned AI", do: func() { a.trainer.Reset() }},
		{label: "Quit", do: func() {
			a.trainer.Stop()
			os.Exit(0)
		}},
	}
}

func menuRect(i int) (x, y, w, h float64) {
	return ScreenW/2 - 200, 258 + float64(i)*48, 400, 40
}

// dropRect is option j of the dropdown opened from menu item i. The list
// opens upward when it would run off the bottom of the screen.
func dropRect(i, j, n int) (x, y, w, h float64) {
	x, y, w, _ = menuRect(i)
	h = 32
	top := y + 44
	if top+float64(n)*h > ScreenH-8 {
		top = y - float64(n)*h - 2
	}
	return x + 20, top + float64(j)*h, w - 40, h
}

func (a *App) updateMenu() {
	mx, my := ebiten.CursorPosition()
	click := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	items := a.menuItems()
	if a.dropOpen >= 0 {
		it := items[a.dropOpen]
		a.dropHover = -1
		for j := range it.options {
			if x, y, w, h := dropRect(a.dropOpen, j, len(it.options)); inRect(mx, my, x, y, w, h) {
				a.dropHover = j
				if click {
					it.choose(j)
					a.dropOpen = -1
					return
				}
			}
		}
		if click || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			a.dropOpen = -1
		}
		return
	}
	a.menuHover = -1
	for i, it := range items {
		x, y, w, h := menuRect(i)
		hit := inRect(mx, my, x, y, w, h)
		if hit {
			a.menuHover = i
		}
		if (hit && click) || inpututil.IsKeyJustPressed(ebiten.Key1+ebiten.Key(i)) {
			animOf(fmt.Sprint("menu", i)).press = 1
			if it.options != nil {
				a.dropOpen, a.dropHover = i, it.selected
			} else {
				it.do()
			}
			return
		}
	}
}
func (a *App) Update() error {
	a.music.update()
	if a.mode != modeMenu {
		if handleHUDClick() || clickButtons(a.overlayButtons()) || clickButtons(a.toolbar()) {
			return nil
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		a.music.toggle()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) {
		a.music.skip()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if a.mode != modeMenu {
		if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
			a.faster()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
			a.slower()
		}
	}

	switch a.mode {
	case modeMenu:
		a.updateMenu()
	case modePlay:
		g := a.game
		if inpututil.IsKeyJustPressed(ebiten.KeyF10) || inpututil.IsKeyJustPressed(ebiten.KeyP) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			a.paused = !a.paused
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
			a.showScores = !a.showScores
		}
		if a.paused {
			g.handleCamera()
			return nil
		}
		if g.over() {
			g.handleCamera()
			if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				a.toMenu()
			}
			return nil
		}
		g.handleInput()
		a.advance(g)
	case modeWatch:
		g := a.game
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && len(g.selectedUnits()) == 0 && g.selBld == nil {
			a.toMenu()
			return nil
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.clearSelection()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			a.paused = !a.paused
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF) {
			a.follow = !a.follow
		}
		g.handleCamera()
		g.handleSelection(-1)
		if g.over() {
			if a.overT += dt; a.overT > 4 {
				a.newWatchGame()
			}
			return nil
		}
		if !a.paused {
			a.advance(g)
		}
		if a.follow {
			a.followAction()
		}
	}
	return nil
}

// advance runs as many simulation steps as the game speed asks for this
// frame, carrying fractions over so 1.5x alternates one and two steps.
func (a *App) advance(g *Game) {
	a.stepAcc += speeds[a.speed]
	for a.stepAcc >= 1 && !g.over() {
		g.step()
		a.stepAcc--
	}
}

// followAction moves the camera toward the biggest group of soldiers.
func (a *App) followAction() {
	g := a.game
	bx, by, best := 0.0, 0.0, 0
	for _, u := range g.units {
		if !u.def().Military {
			continue
		}
		n := 0
		g.eachNear(u.X, u.Y, 6*TileSize, func(o *Unit) {
			if o.def().Military {
				n++
			}
		})
		if u.Order == OrdAttack {
			n *= 2
		}
		if n > best {
			bx, by, best = u.X, u.Y, n
		}
	}
	if best == 0 {
		return
	}
	cx, cy := g.visibleCenter()
	g.centerOn(cx+(bx-cx)*0.05, cy+(by-cy)*0.05)
}

func (a *App) Draw(screen *ebiten.Image) {
	hudButtons = hudButtons[:0]
	switch a.mode {
	case modeMenu:
		a.drawMenu(screen)
		return
	}
	g := a.game
	g.DrawWorld(screen)
	g.drawTopBar(screen)
	g.drawBottomPanel(screen)
	if a.mode == modePlay {
		if a.showScores {
			g.drawScores(screen, ScreenW-262, TopBarH+40)
		}
		drawButtons(screen, a.toolbar())
		if g.over() {
			msg := "VICTORY!"
			if g.winner != g.human {
				msg = "DEFEAT"
			}
			a.overlay(screen, msg, "vs "+a.vsName)
		} else if a.paused {
			a.overlay(screen, "Paused", "vs "+a.vsName)
		}
		drawButtons(screen, a.overlayButtons())
		return
	}
	g.drawScores(screen, ScreenW-262, TopBarH+40)
	dashY := float64(TopBarH + 118)
	if hudMin["score"] {
		dashY = TopBarH + 40 + titleH + 6
	}
	a.drawDashboard(screen, ScreenW-330, dashY)
	a.drawThinking(screen, 10, TopBarH+40)
	drawButtons(screen, a.toolbar())
	if g.over() {
		msg := "Draw"
		if g.winner < 2 {
			msg = teamNames[g.winner] + " wins"
		}
		a.overlay(screen, msg, "Next game starting...")
	}
}

func (a *App) overlay(screen *ebiten.Image, title, sub string) {
	fillRect(screen, 0, ScreenH/2-60, ScreenW, 140, color.RGBA{0, 0, 0, 170})
	drawText(screen, title, ScreenW/2-textWidth(title, 40, true)/2, ScreenH/2-50, 40, gold, true)
	drawText(screen, sub, ScreenW/2-textWidth(sub, 16, false)/2, ScreenH/2+8, 16, white, false)
}

func (a *App) drawMenu(screen *ebiten.Image) {
	screen.Fill(color.RGBA{24, 20, 16, 255})
	s := sprites()
	for i, k := range []BldKind{House, TownCenter, Castle, Barracks} {
		sp := s.finished(k, i%2, min(i+1, 3))
		drawFit(screen, sp.img, 40+float64(i)*310, 80, 280, 200, func() *ebiten.ColorScale {
			cs := &ebiten.ColorScale{}
			cs.Scale(0.35, 0.35, 0.35, 1)
			return cs
		}())
	}
	title := "Mini Empires"
	drawText(screen, title, ScreenW/2-textWidth(title, 64, true)/2+3, 120+3, 64, black, true)
	drawText(screen, title, ScreenW/2-textWidth(title, 64, true)/2, 120, 64, gold, true)
	sub := "An Age of Empires II style RTS with a self-taught AI"
	drawText(screen, sub, ScreenW/2-textWidth(sub, 18, false)/2, 205, 18, grey, false)
	items := a.menuItems()
	for i, it := range items {
		x, y, w, h := menuRect(i)
		an := animate(fmt.Sprint("menu", i), i == a.menuHover)
		y += an.press * 2
		box(screen, x, y, w, h)
		fillRect(screen, x+2, y+2, w-4, h-4, color.RGBA{uint8(40 * an.hover), uint8(30 * an.hover), uint8(12 * an.hover), uint8(40 * an.hover)})
		c := white
		if i == a.menuHover || i == a.dropOpen {
			strokeRect(screen, x, y, w, h, 2, gold)
			c = gold
		}
		label := fmt.Sprintf("%d. %s", i+1, it.label)
		if it.options != nil {
			drawText(screen, label, x+16, y+10, 18, c, true)
			v := it.options[it.selected]
			drawText(screen, v, x+w-textWidth(v, 16, false)-34, y+12, 16, gold, false)
			ax, ay := x+w-24, y+h/2-2
			vector.StrokeLine(screen, float32(ax), float32(ay), float32(ax+5), float32(ay+5), 2, gold, true)
			vector.StrokeLine(screen, float32(ax+5), float32(ay+5), float32(ax+10), float32(ay), 2, gold, true)
			continue
		}
		drawText(screen, label, x+w/2-textWidth(label, 18, true)/2, y+10, 18, c, true)
	}
	if a.dropOpen >= 0 {
		it := items[a.dropOpen]
		for j, o := range it.options {
			x, y, w, h := dropRect(a.dropOpen, j, len(it.options))
			fillRect(screen, x, y, w, h, panel2)
			strokeRect(screen, x, y, w, h, 1, trim)
			c := white
			if j == a.dropHover {
				fillRect(screen, x+1, y+1, w-2, h-2, color.RGBA{90, 70, 40, 255})
				c = gold
			}
			if j == it.selected {
				o = "• " + o
			}
			drawText(screen, o, x+14, y+7, 15, c, j == it.selected)
		}
	}
	st := a.trainer.Stats()
	info := fmt.Sprintf("Learned AI: %d training games this session, %d updates in total. Saved to %s", st.Games, st.Updates, policyPath())
	drawText(screen, info, ScreenW/2-textWidth(info, 13, false)/2, ScreenH-60, 13, grey, false)
	help := "M toggles music   ] next track   F11 fullscreen"
	if n := a.music.nowPlaying(); n != "" {
		help = "Now playing: " + n + "      " + help
	}
	drawText(screen, help, ScreenW/2-textWidth(help, 13, false)/2, ScreenH-36, 13, grey, false)
}

// drawThinking shows what each AI decided last and how likely each
// option was, so the policy's preferences can be watched as it learns.
func (a *App) drawThinking(screen *ebiten.Image, x, y float64) {
	g := a.game
	if !window(screen, "thinking", "What the AI is choosing", x, y, 250, 300) {
		return
	}
	for t := range 2 {
		b := g.brains[t]
		p := a.watchPol[t]
		if b == nil || p == nil || p.probs == nil {
			continue
		}
		cx := x + 10 + float64(t)*120
		fillRect(screen, cx, y+28, 10, 10, teamColors[t])
		drawText(screen, actionNames[b.last], cx+14, y+25, 12, white, true)
		type pa struct {
			a int
			p float64
		}
		var top []pa
		for i, v := range p.probs {
			top = append(top, pa{i, v})
		}
		for i := range top {
			for j := i + 1; j < len(top); j++ {
				if top[j].p > top[i].p {
					top[i], top[j] = top[j], top[i]
				}
			}
		}
		for i := range min(12, len(top)) {
			yy := y + 46 + float64(i)*20
			fillRect(screen, cx, yy+14, 110*top[i].p, 4, teamColors[t])
			drawText(screen, fmt.Sprintf("%s %.0f%%", actionNames[top[i].a], top[i].p*100), cx, yy, 11, grey, false)
		}
	}
}

func (a *App) drawDashboard(screen *ebiten.Image, x, y float64) {
	st := a.trainer.Stats()
	w, h := 320.0, 300.0
	if !window(screen, "training", "Self-play training", x, y, w, h) {
		return
	}
	lines := []string{
		fmt.Sprintf("Games played: %d  (%.1f games/s)", st.Games, st.GamesPerS),
		fmt.Sprintf("Policy updates: %d", st.Updates),
		fmt.Sprintf("Win rate vs Scripted AI: %.0f%%  (last %d)", st.EvalWin*100, st.EvalN),
		fmt.Sprintf("Mean score: %.0f    Mean length: %s", st.Score, clock(st.Duration)),
		fmt.Sprintf("Entropy %.2f   Value loss %.3f   KL %.4f", st.Last.Entropy, st.Last.ValueLoss, st.Last.ApproxKL),
	}
	for i, l := range lines {
		drawText(screen, l, x+10, y+28+float64(i)*17, 12, white, false)
	}
	cx, cy, cw, ch := x+10, y+124, w-20, h-150
	fillRect(screen, cx, cy, cw, ch, color.RGBA{20, 16, 12, 255})
	strokeRect(screen, cx, cy, cw, ch, 1, trim)
	hist := st.History
	if len(hist) >= 2 {
		maxScore := 1.0
		for _, p := range hist {
			maxScore = max(maxScore, p.Score)
		}
		plot := func(val func(HistPoint) float64, c color.Color) {
			for i := 1; i < len(hist); i++ {
				x0 := cx + cw*float64(i-1)/float64(len(hist)-1)
				x1 := cx + cw*float64(i)/float64(len(hist)-1)
				line(screen, x0, cy+ch*(1-val(hist[i-1])), x1, cy+ch*(1-val(hist[i])), 1.5, c)
			}
		}
		plot(func(p HistPoint) float64 { return p.Score / maxScore }, color.RGBA{120, 170, 255, 255})
		plot(func(p HistPoint) float64 { return p.EvalWin }, color.RGBA{255, 200, 80, 255})
		plot(func(p HistPoint) float64 { return min(p.Entropy/3.2, 1) }, color.RGBA{180, 180, 180, 160})
		drawText(screen, fmt.Sprintf("max %.0f", maxScore), cx+cw-60, cy+2, 10, grey, false)
	} else {
		drawText(screen, "Collecting the first batch of games...", cx+10, cy+ch/2-8, 12, grey, false)
	}
	ly := y + h - 20
	for i, l := range []struct {
		s string
		c color.RGBA
	}{{"score", color.RGBA{120, 170, 255, 255}}, {"win % vs scripted", color.RGBA{255, 200, 80, 255}}, {"entropy", color.RGBA{180, 180, 180, 255}}} {
		lx := x + 10 + float64(i)*105
		fillRect(screen, lx, ly+5, 10, 3, l.c)
		drawText(screen, l.s, lx+14, ly, 11, grey, false)
	}
}
