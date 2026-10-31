package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// HUD windows can be minimized to their title bar. hudMin remembers which
// are minimized for the rest of the session, by window id.
var hudMin = map[string]bool{}

const titleH = 22

type hudButton struct {
	id         string
	x, y, w, h float64
}

// hudButtons holds the minimize buttons drawn in the last frame, so a click
// can be matched to one before the next frame is drawn.
var hudButtons []hudButton

// window draws a panel with a title bar and a minimize button. It returns
// false when the window is minimized and its contents should be skipped.
func window(dst *ebiten.Image, id, title string, x, y, w, h float64) bool {
	min := hudMin[id]
	if min {
		h = titleH
	}
	box(dst, x, y, w, h)
	drawText(dst, title, x+10, y+4, 13, gold, true)
	bx, by := x+w-titleH, y+2
	minButton(dst, bx, by, min)
	hudButtons = append(hudButtons, hudButton{id, bx, by, titleH - 4, titleH - 4})
	return !min
}

// minButton draws a minimize (–) or restore (+) button.
func minButton(dst *ebiten.Image, x, y float64, minimized bool) {
	s := float64(titleH - 4)
	fillRect(dst, x, y, s, s, panel2)
	mx, my := ebiten.CursorPosition()
	edge := trim
	if inRect(mx, my, x, y, s, s) {
		fillRect(dst, x, y, s, s, color.RGBA{96, 76, 44, 230})
		edge = gold
	}
	strokeRect(dst, x, y, s, s, 1, edge)
	line(dst, x+4, y+s/2, x+s-4, y+s/2, 2, white)
	if minimized {
		line(dst, x+s/2, y+4, x+s/2, y+s-4, 2, white)
	}
}

// handleHUDClick toggles the window whose minimize button was clicked, and
// reports whether the click was used.
func handleHUDClick() bool {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}
	mx, my := ebiten.CursorPosition()
	for _, b := range hudButtons {
		if inRect(mx, my, b.x, b.y, b.w, b.h) {
			hudMin[b.id] = !hudMin[b.id]
			return true
		}
	}
	return false
}

// The bottom command panel shrinks to a strip when minimized.
const bottomStripH = 26

func bottomH() int {
	if hudMin["bottom"] {
		return bottomStripH
	}
	return BottomBarH
}
