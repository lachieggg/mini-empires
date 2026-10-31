package art

import (
	"image/color"
	"math"
)

// Ring is the white selection ellipse drawn under units.
func Ring() *Canvas {
	c := New(34, 18)
	c.Ellipse(V{17, 9}, 16, 8, func(nx, ny float64) color.RGBA {
		if nx*nx+ny*ny > 0.62 {
			return RGB(255, 255, 255)
		}
		return color.RGBA{}
	})
	return c
}

// Icons holds the food, wood, gold and stone carry icons side by side.
func Icons() *Canvas {
	c := New(IconSize*4, IconSize)
	c.Ellipse(V{6, 8}, 5, 3.5, lit(RGB(150, 105, 55)))
	for _, p := range []V{{4, 5}, {7, 4}, {6, 6.5}, {8.5, 6}} {
		c.Ellipse(p, 1.6, 1.6, lit(RGB(210, 40, 70)))
	}
	for i, y := range []float64{8, 5.5} {
		x := 12.0 + float64(i)
		c.Line(V{x + 1, y + 1}, V{x + 9, y - 1}, 3, RGB(130, 85, 45))
		c.Ellipse(V{x + 9, y - 1}, 1.5, 1.5, solid(RGB(210, 170, 110)))
	}
	for _, p := range []V{{28, 8}, {32, 8}, {30, 5}} {
		c.Ellipse(p, 2.6, 2.2, lit(RGB(240, 195, 45)))
	}
	for _, p := range []V{{39.5, 8}, {44.5, 8.2}, {42, 4.8}} {
		c.Quad(p.Add(V{-2.6, -1}), V{2.6, -1.4}, V{2.6, 1.4}, solid(RGB(215, 215, 210)))
		c.Quad(p.Add(V{-2.6, -1}), V{2.6, 1.4}, V{0, 2.6}, solid(RGB(178, 178, 176)))
		c.Quad(p.Add(V{0, 0.4}), V{2.6, -1.4}, V{0, 2.6}, solid(RGB(135, 135, 136)))
	}
	finish(c)
	return c
}

// Buttons are the command-button icons.
const (
	ButtonEco = iota
	ButtonMil
	ButtonBack
	ButtonAge1
	ButtonAge2
	ButtonAge3
	ButtonCancel
	ButtonStop
	ButtonLoom
	ButtonWheelbarrow
	ButtonHandcart
	ButtonDoubleBitAxe
	ButtonBowSaw
	ButtonGoldMining
	ButtonStoneMining
	ButtonHorseCollar
	ButtonHeavyPlow
	ButtonForging
	ButtonIronCasting
	ButtonBlastFurnace
	ButtonFletching
	ButtonBodkinArrow
	ButtonScaleMail
	ButtonChainMail
	ButtonScaleBarding
	ButtonChainBarding
	ButtonPaddedArmor
	ButtonTownBell
	ButtonUngarrison
	NumButtons
)

var ButtonNames = [NumButtons]string{"eco", "mil", "back", "age1", "age2", "age3", "cancel", "stop", "loom", "wheelbarrow", "handcart", "doublebitaxe", "bowsaw", "goldmining", "stonemining", "horsecollar", "heavyplow", "forging", "ironcasting", "blastfurnace", "fletching", "bodkinarrow", "scalemail", "chainmail", "scalebarding", "chainbarding", "paddedarmor", "townbell", "ungarrison"}

func grad(col color.RGBA, a, b V) func(float64, float64) color.RGBA {
	return func(u, v float64) color.RGBA { return Shade(col, 1.15-0.35*(u*a.X+v*a.Y)-0.1*(u*b.X+v*b.Y)) }
}

func Button(kind int) *Canvas {
	c := New(ButtonSize, ButtonSize)
	switch kind {
	case ButtonEco:
		c.Line(V{9, 29}, V{23, 13}, 3.6, RGB(150, 104, 58))
		c.Line(V{9.5, 28}, V{22, 13.5}, 1.2, RGB(190, 145, 90))
		d := V{14, -16}.Mul(1 / math.Hypot(14, 16))
		p := V{-d.Y, d.X}
		ctr := V{23.5, 12.5}
		c.Quad(ctr.Sub(p.Mul(9)).Sub(d.Mul(3.5)), p.Mul(18), d.Mul(7), func(u, v float64) color.RGBA {
			return Shade(steel, 1.2-0.45*v-0.15*u)
		})
		c.Quad(ctr.Add(p.Mul(9)).Sub(d.Mul(3.5)), p.Mul(-2), d.Mul(7), solid(Shade(steel, 0.7)))
	case ButtonMil:
		for _, s := range []float64{-1, 1} {
			a, b := V{18 - 11*s, 29}, V{18 + 10*s, 7}
			d := b.Sub(a).Mul(1 / math.Hypot(b.X-a.X, b.Y-a.Y))
			n := V{-d.Y, d.X}
			guard := a.Add(d.Mul(7))
			c.Line(guard, b, 3.6, RGB(205, 210, 222))
			c.Line(guard.Add(n.Mul(0.6)), b, 1.2, RGB(245, 246, 250))
			c.Line(guard.Sub(n.Mul(4.5)), guard.Add(n.Mul(4.5)), 2.4, goldCol)
			c.Line(a, guard, 2.4, RGB(110, 70, 40))
			c.Ellipse(a, 2, 2, lit(goldCol))
		}
	case ButtonBack:
		col := RGB(232, 200, 90)
		c.Quad(V{14, 14}, V{17, 0}, V{0, 8}, func(u, v float64) color.RGBA { return Shade(col, 1.12-0.3*v) })
		c.Tri(V{4, 18}, V{11, -12}, V{11, 12}, func(u, v float64) color.RGBA { return Shade(col, 1.1-0.35*v) })
	case ButtonAge1:
		iron := RGB(160, 150, 140)
		c.Ellipse(V{18, 20}, 11, 12, func(nx, ny float64) color.RGBA {
			if ny > 0.15 {
				return color.RGBA{}
			}
			return Shade(iron, Lit(nx, ny))
		})
		c.Quad(V{6.5, 20}, V{23, 0}, V{0, 4}, grad(RGB(176, 128, 70), V{}, V{0, 1}))
		c.Line(V{18, 22}, V{18, 31}, 3.2, Shade(iron, 0.95))
		c.Line(V{18, 9}, V{18, 20}, 1.6, Shade(iron, 1.25))
		for _, x := range []float64{9, 13, 23, 27} {
			c.Ellipse(V{x, 22}, 0.9, 0.9, solid(RGB(230, 200, 120)))
		}
	case ButtonAge2:
		st := RGB(178, 172, 160)
		c.Quad(V{10, 13}, V{16, 0}, V{0, 19}, grad(st, V{1, 0}, V{0, 1}))
		for _, x := range []float64{8, 14.5, 21} {
			c.Quad(V{x, 7}, V{7, 0}, V{0, 7}, grad(st, V{1, 0}, V{0, 1}))
		}
		c.Ellipse(V{18, 29}, 3.5, 5, solid(RGB(40, 34, 30)))
		c.Quad(V{14.5, 29}, V{7, 0}, V{0, 3}, solid(RGB(40, 34, 30)))
		c.Quad(V{12, 17}, V{2, 0}, V{0, 4}, solid(RGB(40, 34, 30)))
		c.Quad(V{22, 17}, V{2, 0}, V{0, 4}, solid(RGB(40, 34, 30)))
		c.Line(V{18, 7}, V{18, 1}, 1.2, RGB(90, 60, 35))
		c.Tri(V{18.5, 1}, V{7, 2}, V{0, 4.5}, solid(RGB(205, 45, 40)))
	case ButtonAge3:
		g := goldCol
		c.Quad(V{6, 20}, V{24, 0}, V{0, 8}, grad(g, V{0, 1}, V{1, 0}))
		for i, x := range []float64{6, 14, 22} {
			h := 13.0
			if i == 1 {
				h = 16
			}
			c.Tri(V{x, 20.5}, V{8, 0}, V{4, -h}, func(u, v float64) color.RGBA { return Shade(g, 1.15-0.3*u) })
			c.Ellipse(V{x + 4, 20 - h}, 1.8, 1.8, lit(RGB(250, 236, 170)))
		}
		for i, x := range []float64{10, 18, 26} {
			col := RGB(205, 40, 50)
			if i == 1 {
				col = RGB(50, 90, 210)
			}
			c.Ellipse(V{x, 24}, 2, 2, lit(col))
		}
	case ButtonCancel:
		red := RGB(215, 45, 40)
		c.Line(V{8, 8}, V{28, 28}, 6, red)
		c.Line(V{28, 8}, V{8, 28}, 6, red)
		c.Line(V{8.5, 7.5}, V{27.5, 26.5}, 1.5, RGB(250, 130, 110))
		c.Line(V{27.5, 7.5}, V{8.5, 26.5}, 1.5, RGB(250, 130, 110))
	case ButtonStop:
		st := RGB(70, 100, 170)
		c.Ellipse(V{18, 16}, 12, 17, func(nx, ny float64) color.RGBA {
			if ny < -0.45 {
				return color.RGBA{}
			}
			return Shade(st, Lit(nx, ny))
		})
		c.Quad(V{6, 8}, V{24, 0}, V{0, 3}, solid(Shade(steel, 1.1)))
		c.Ellipse(V{18, 16}, 9, 14, func(nx, ny float64) color.RGBA {
			if ny < -0.4 || (math.Abs(nx) > 0.16 && math.Abs(ny+0.05) > 0.12) {
				return color.RGBA{}
			}
			return Shade(RGB(240, 236, 222), Lit(nx, ny))
		})
	default:
		extraButton(c, kind)
	}
	finish(c)
	return c
}
