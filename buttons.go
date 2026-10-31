package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// uiButton is a clickable, animated button. It brightens and shows a gold
// edge while hovered, presses in when clicked, and stays lit while on.
type uiButton struct {
	id, label, tip string
	x, y, w, h     float64
	on, disabled   bool
	do             func()
}

type buttonAnim struct{ hover, press float64 }

// buttonAnims keeps each button's animation between frames, by id.
var buttonAnims = map[string]*buttonAnim{}

func animOf(id string) *buttonAnim {
	a := buttonAnims[id]
	if a == nil {
		a = &buttonAnim{}
		buttonAnims[id] = a
	}
	return a
}

// animate eases a button's hover glow toward whether the cursor is over it
// and lets a press spring back. It runs once per update.
func animate(id string, over bool) *buttonAnim {
	a := animOf(id)
	target := 0.0
	if over {
		target = 1
	}
	a.hover += (target - a.hover) * 0.3
	a.press = max(0, a.press-0.12)
	return a
}

// clickButtons animates bs and runs the one clicked, reporting whether a
// click was used.
func clickButtons(bs []uiButton) bool {
	mx, my := ebiten.CursorPosition()
	clicked := false
	for i := range bs {
		b := &bs[i]
		over := inRect(mx, my, b.x, b.y, b.w, b.h)
		a := animate(b.id, over)
		if over && !clicked && !b.disabled && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			a.press = 1
			clicked = true
			b.do()
		}
	}
	return clicked
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	m := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), m(a.A, b.A)}
}

func drawButtons(dst *ebiten.Image, bs []uiButton) {
	var tip *uiButton
	for i := range bs {
		b := &bs[i]
		a := animOf(b.id)
		// Pressing shrinks the button a little around its center.
		k := 1 - 0.08*a.press
		w, h := b.w*k, b.h*k
		x, y := b.x+(b.w-w)/2, b.y+(b.h-h)/2+a.press
		fill := lerpColor(panel2, color.RGBA{96, 76, 44, 230}, a.hover)
		if b.on {
			fill = lerpColor(color.RGBA{120, 92, 40, 235}, color.RGBA{150, 116, 52, 240}, a.hover)
		}
		fillRect(dst, x, y, w, h, fill)
		fillRect(dst, x+1, y+1, w-2, 2, color.RGBA{255, 240, 200, uint8(30 + 50*a.hover)})
		edge := lerpColor(trim, gold, a.hover)
		strokeRect(dst, x, y, w, h, 1.5, edge)
		c := white
		if b.disabled {
			c = grey
		}
		size := 13.0
		drawText(dst, b.label, x+w/2-textWidth(b.label, size, true)/2, y+h/2-size*0.62, size, c, true)
		if a.hover > 0.7 && b.tip != "" {
			tip = b
		}
	}
	if tip != nil {
		w := textWidth(tip.tip, 12, false)
		x := min(max(tip.x+tip.w/2-w/2-10, 4), ScreenW-w-24)
		y := tip.y + tip.h + 6
		if y+24 > ScreenH {
			y = tip.y - 30
		}
		box(dst, x, y, w+20, 24)
		drawText(dst, tip.tip, x+10, y+4, 12, white, false)
	}
}

// row lays buttons out left to right from (x, y), each sized to its label.
func row(x, y float64, bs []uiButton) []uiButton {
	for i := range bs {
		bs[i].x, bs[i].y, bs[i].h = x, y, 26
		if bs[i].w == 0 {
			bs[i].w = textWidth(bs[i].label, 13, true) + 22
		}
		x += bs[i].w + 6
	}
	return bs
}

func rowWidth(bs []uiButton) float64 {
	w := 0.0
	for _, b := range bs {
		if b.w == 0 {
			b.w = textWidth(b.label, 13, true) + 22
		}
		w += b.w + 6
	}
	return w - 6
}

// centeredRow lays bs out as a row centered on x.
func centeredRow(x, y float64, bs []uiButton) []uiButton {
	return row(x-rowWidth(bs)/2, y, bs)
}

// Game speed, in simulation steps per frame.
var speeds = []float64{1, 1.5, 2, 3, 4, 8, 16, 32}

func speedLabel(s float64) string {
	if s == float64(int(s)) {
		return fmt.Sprintf("%dx", int(s))
	}
	return fmt.Sprintf("%.1fx", s)
}

func (a *App) slower() { a.speed = max(a.speed-1, 0) }
func (a *App) faster() { a.speed = min(a.speed+1, len(speeds)-1) }

func (a *App) toMenu() {
	a.trainer.Stop()
	a.mode, a.paused = modeMenu, false
}

// toolbar is the row of buttons under the top bar.
func (a *App) toolbar() []uiButton {
	g := a.game
	pause := "Pause"
	if a.paused {
		pause = "Resume"
	}
	bs := []uiButton{
		{id: "slower", label: "Slower", tip: "Slow the game down (-)", disabled: a.speed == 0, do: a.slower},
		{id: "speed", label: speedLabel(speeds[a.speed]), tip: "Game speed. Click to reset to 1x", w: 52, do: func() { a.speed = 0 }},
		{id: "faster", label: "Faster", tip: "Speed the game up (+)", disabled: a.speed == len(speeds)-1, do: a.faster},
		{id: "pause", label: pause, tip: "Pause or resume (Space, P)", on: a.paused, do: func() { a.paused = !a.paused }},
	}
	if a.mode == modeWatch {
		bs = append(bs,
			uiButton{id: "follow", label: "Follow fights", tip: "Move the camera to the biggest battle (F)", on: a.follow, do: func() { a.follow = !a.follow }},
			uiButton{id: "newgame", label: "New game", tip: "Start a new self-play game with the latest network", do: a.newWatchGame},
		)
	} else {
		idle := 0
		for _, u := range g.units {
			if u.Team == g.human && u.Kind == Villager && u.Order == OrdIdle && u.Inside == nil {
				idle++
			}
		}
		bs = append(bs,
			uiButton{id: "idle", label: fmt.Sprintf("Idle villagers: %d", idle), tip: "Select the next idle villager (.)", disabled: idle == 0, on: idle > 0, do: g.selectIdleVillager},
			uiButton{id: "scores", label: "Scores", tip: "Show or hide the score panel (F4)", on: a.showScores, do: func() { a.showScores = !a.showScores }},
		)
	}
	musicLabel := "Music off"
	if a.music.on {
		musicLabel = "Music on"
	}
	bs = append(bs,
		uiButton{id: "music", label: musicLabel, tip: "Turn the music on or off (M)", on: a.music.on, do: a.music.toggle},
		uiButton{id: "track", label: "Next track", tip: "Skip to the next track (])", do: a.music.skip},
		uiButton{id: "menu", label: "Menu", tip: "Return to the main menu (Esc in watch mode)", do: a.toMenu},
	)
	return centeredRow(ScreenW/2, TopBarH+6, bs)
}

// overlayButtons are the buttons on the pause and game over screens.
func (a *App) overlayButtons() []uiButton {
	if a.mode != modePlay {
		return nil
	}
	g := a.game
	switch {
	case g.over():
		return centeredRow(ScreenW/2, ScreenH/2+40, []uiButton{
			{id: "again", label: "Play again", tip: "Start a new game against the same opponent", do: func() { a.startPlay(a.vsPolicy, a.vsName) }},
			{id: "overmenu", label: "Main menu", do: a.toMenu},
		})
	case a.paused:
		return centeredRow(ScreenW/2, ScreenH/2+40, []uiButton{
			{id: "resume", label: "Resume", do: func() { a.paused = false }},
			{id: "pausemenu", label: "Main menu", do: a.toMenu},
		})
	}
	return nil
}
