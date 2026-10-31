package art

import (
	"image/color"
	"math"
)

// poly fills a convex polygon; fn gets the pixel position.
func (c *Canvas) poly(pts []V, fn func(x, y float64) color.RGBA) {
	for i := 1; i+1 < len(pts); i++ {
		p, e1, e2 := pts[0], pts[i].Sub(pts[0]), pts[i+1].Sub(pts[0])
		c.Tri(p, e1, e2, func(u, v float64) color.RGBA {
			q := p.Add(e1.Mul(u)).Add(e2.Mul(v))
			return fn(q.X, q.Y)
		})
	}
}

// mask fills every pixel where in is true.
func (c *Canvas) mask(in func(x, y float64) bool, fn func(x, y float64) color.RGBA) {
	for y := range c.H {
		for x := range c.W {
			px, py := float64(x)+0.5, float64(y)+0.5
			if in(px, py) {
				c.Blend(x, y, fn(px, py))
			}
		}
	}
}

// diag shades a colour lighter towards the upper left of a 36 px button.
func diag(col color.RGBA) func(x, y float64) color.RGBA {
	return func(x, y float64) color.RGBA { return Shade(col, 1.2-0.012*(x+y)) }
}

func segDist(p, a, b V) float64 {
	d := b.Sub(a)
	t := math.Max(0, math.Min(1, ((p.X-a.X)*d.X+(p.Y-a.Y)*d.Y)/(d.X*d.X+d.Y*d.Y)))
	q := a.Add(d.Mul(t))
	return math.Hypot(p.X-q.X, p.Y-q.Y)
}

func handle(c *Canvas, a, b V, w float64) {
	c.Line(a, b, w, haftCol)
	c.Line(a.Add(V{-0.5, -0.5}), b.Add(V{-0.5, -0.5}), w*0.35, Shade(haftCol, 1.3))
}

func btnWheel(c *Canvas, ctr V, r float64) {
	c.Ellipse(ctr, r, r, func(nx, ny float64) color.RGBA {
		d := nx*nx + ny*ny
		switch {
		case d > 0.55:
			return Shade(RGB(110, 76, 44), Lit(nx, ny))
		case d < 0.06:
			return RGB(70, 70, 76)
		case math.Abs(nx) < 0.14 || math.Abs(ny) < 0.14:
			return RGB(150, 108, 64)
		}
		return color.RGBA{}
	})
}

func btnPick(c *Canvas) {
	handle(c, V{9, 31}, V{24, 10}, 3)
	d := V{15, -21}.Mul(1 / math.Hypot(15, 21))
	n := V{-d.Y, d.X}
	hd := V{24, 10}
	for _, s := range []float64{-1, 1} {
		prev := hd
		for i := 1; i <= 6; i++ {
			t := float64(i) / 6
			p := hd.Add(n.Mul(s * 10 * t)).Sub(d.Mul(4 * t * t))
			c.Line(prev, p, 4-2.6*t, Shade(toolHead, 1.15-0.3*t))
			prev = p
		}
	}
}

func shirt(c *Canvas, fill func(x, y float64) color.RGBA) {
	in := func(x, y float64) bool {
		p := V{x, y}
		if math.Hypot(x-18, y-7.5) < 3.5 {
			return false
		}
		body := x >= 11 && x <= 25 && y >= 8 && y <= 31
		return body || segDist(p, V{12, 10}, V{6, 21}) < 3.2 || segDist(p, V{24, 10}, V{30, 21}) < 3.2
	}
	c.mask(in, func(x, y float64) color.RGBA { return Shade(fill(x, y), 1.15-0.01*(x+y)*0.6) })
}

func scales(base color.RGBA) func(x, y float64) color.RGBA {
	return func(x, y float64) color.RGBA {
		row := math.Floor(y / 3)
		off := math.Mod(row, 2) * 2
		fy := math.Mod(y, 3)
		fx := math.Mod(x+off, 4)
		f := 1.1 - 0.12*fy
		if fy > 2.2 || (fy > 1.2 && (fx < 0.8 || fx > 3.2)) {
			f = 0.65
		}
		return Shade(base, f)
	}
}

func chain(base color.RGBA) func(x, y float64) color.RGBA {
	return func(x, y float64) color.RGBA {
		ix, iy := int(math.Floor(x)), int(math.Floor(y))
		if (ix+iy)%3 == 0 || ((ix-iy)%3+3)%3 == 0 {
			return Shade(base, 0.7)
		}
		return Shade(base, 1.1)
	}
}

func horseHead(c *Canvas, barding func(x, y float64) color.RGBA) {
	coat := RGB(150, 100, 58)
	c.Capsule(V{12, 31}, V{17, 15}, 5.5, smoothLit(coat))
	c.Capsule(V{17, 13}, V{29, 21}, 4.4, smoothLit(coat))
	c.Tri(V{14, 10}, V{3, -6}, V{4, 1}, solid(Shade(coat, 0.9)))
	c.Ellipse(V{29.5, 22}, 2.5, 2, solid(Shade(coat, 0.7)))
	c.mask(func(x, y float64) bool {
		p := V{x, y}
		return segDist(p, V{12, 31}, V{17, 15}) < 6 || segDist(p, V{16, 12}, V{26, 17}) < 3.4
	}, barding)
	c.Ellipse(V{22, 15}, 1.3, 1.3, solid(RGB(20, 16, 14)))
}

func bellShape(c *Canvas) {
	bronze := RGB(214, 164, 64)
	c.mask(func(x, y float64) bool {
		if y < 8 || y > 28 {
			return math.Hypot(x-18, y-9) < 5.5 && y < 9
		}
		w := 5.5 + (y-8)*0.28
		if y > 23 {
			w += (y - 23) * 1.3
		}
		return math.Abs(x-18) < w
	}, func(x, y float64) color.RGBA {
		f := 1.25 - 0.045*math.Abs(x-14)
		if y > 26 {
			f *= 0.85
		}
		return Shade(bronze, f)
	})
	c.Ellipse(V{18, 30.5}, 2.4, 2.4, lit(RGB(120, 90, 40)))
	c.Line(V{15, 4}, V{21, 4}, 2, RGB(90, 70, 40))
}

func arrowDiag(c *Canvas, a, b V, headLen, w float64, fletch color.RGBA) {
	d := b.Sub(a).Mul(1 / math.Hypot(b.X-a.X, b.Y-a.Y))
	n := V{-d.Y, d.X}
	c.Line(a, b.Sub(d.Mul(headLen*0.6)), 1.8, RGB(170, 126, 70))
	base := b.Sub(d.Mul(headLen))
	c.poly([]V{base.Add(n.Mul(w)), b, base.Sub(n.Mul(w))}, diag(toolHead))
	for _, s := range []float64{-1, 1} {
		p := a.Add(d.Mul(1))
		c.poly([]V{p, p.Add(d.Mul(6)), p.Add(d.Mul(2)).Add(n.Mul(3.5 * s)), p.Add(n.Mul(3 * s))}, diag(fletch))
	}
}

func anvilIcon(c *Canvas, y float64) {
	iron := RGB(84, 86, 96)
	c.poly([]V{{3, y}, {10, y}, {10, y + 4}}, diag(iron))
	c.Quad(V{10, y}, V{20, 0}, V{0, 5}, func(u, v float64) color.RGBA { return Shade(iron, 1.35-0.4*v) })
	c.Quad(V{15, y + 5}, V{8, 0}, V{0, 5}, solid(Shade(iron, 0.8)))
	c.Quad(V{11, y + 10}, V{16, 0}, V{0, 4}, solid(Shade(iron, 0.95)))
}

func extraButton(c *Canvas, kind int) {
	switch kind {
	case ButtonLoom:
		wood := RGB(150, 104, 58)
		c.Quad(V{10, 17}, V{16, 0}, V{0, 11}, func(u, v float64) color.RGBA {
			if int(v*11)%4 < 2 {
				return Shade(RGB(200, 60, 50), 1.1-0.2*u)
			}
			return Shade(RGB(236, 220, 180), 1.1-0.2*u)
		})
		for x := 11.0; x < 26; x += 2 {
			c.Line(V{x, 7}, V{x, 17}, 0.6, RGB(240, 232, 210))
		}
		for _, x := range []float64{8, 28} {
			c.Line(V{x, 4}, V{x, 32}, 3, wood)
		}
		c.Line(V{7, 6}, V{29, 6}, 3, Shade(wood, 1.1))
		c.Line(V{7, 29}, V{29, 29}, 3, Shade(wood, 0.9))
	case ButtonWheelbarrow:
		handle(c, V{14, 22}, V{33, 13}, 2.4)
		c.poly([]V{{4, 12}, {24, 12}, {21, 22}, {8, 22}}, diag(RGB(150, 104, 58)))
		c.Line(V{4, 12}, V{24, 12}, 1.5, RGB(190, 150, 96))
		c.Line(V{20, 22}, V{22, 30}, 2, RGB(110, 76, 44))
		btnWheel(c, V{9, 26}, 6)
	case ButtonHandcart:
		handle(c, V{26, 16}, V{34, 8}, 2.4)
		c.Quad(V{3, 10}, V{25, 0}, V{0, 11}, func(u, v float64) color.RGBA {
			if int(v*11)%4 == 0 {
				return Shade(RGB(110, 76, 44), 1)
			}
			return Shade(RGB(160, 112, 62), 1.15-0.3*u)
		})
		btnWheel(c, V{9, 25}, 6.5)
		btnWheel(c, V{22, 25}, 6.5)
	case ButtonDoubleBitAxe:
		handle(c, V{18, 33}, V{18, 5}, 3)
		for _, s := range []float64{-1, 1} {
			c.poly([]V{{18, 8}, {18 + 12*s, 3}, {18 + 13*s, 19}, {18, 15}}, func(x, y float64) color.RGBA {
				return Shade(toolHead, 1.25-0.03*math.Abs(x-18)-0.01*y)
			})
			c.Line(V{18 + 12*s, 3}, V{18 + 13*s, 19}, 1.4, RGB(250, 252, 255))
		}
	case ButtonBowSaw:
		prev := V{5, 25}
		for i := 1; i <= 12; i++ {
			a := math.Pi * float64(i) / 12
			p := V{18 - 13*math.Cos(a), 25 - 17*math.Sin(a)}
			c.Line(prev, p, 3, Shade(haftCol, 1-0.1*math.Sin(a)))
			prev = p
		}
		c.Line(V{5, 26}, V{31, 26}, 3, steel)
		for x := 6.0; x < 31; x += 2 {
			c.Blend(int(x), 28, Shade(steel, 0.8))
		}
		c.Line(V{6, 25}, V{30, 25}, 0.8, RGB(250, 252, 255))
	case ButtonGoldMining:
		btnPick(c)
		nugget(c, V{25, 27}, 6, 3)
		nugget(c, V{16, 30}, 4, 5)
	case ButtonStoneMining:
		btnPick(c)
		st := RGB(186, 186, 182)
		c.poly([]V{{18, 24}, {25, 20}, {33, 24}, {26, 28}}, solid(Shade(st, 1.2)))
		c.poly([]V{{18, 24}, {26, 28}, {26, 34}, {18, 30}}, solid(Shade(st, 0.95)))
		c.poly([]V{{26, 28}, {33, 24}, {33, 30}, {26, 34}}, solid(Shade(st, 0.72)))
	case ButtonHorseCollar:
		c.Ellipse(V{18, 18}, 13, 15, func(nx, ny float64) color.RGBA {
			d := nx*nx + ny*ny
			if d < 0.38 {
				return color.RGBA{}
			}
			if d < 0.5 {
				return Shade(RGB(120, 80, 44), 0.8)
			}
			return Shade(RGB(150, 100, 56), Lit(nx, ny))
		})
		for _, s := range []float64{-1, 1} {
			c.Line(V{18 + 9*s, 7}, V{18 + 11*s, 26}, 2, goldCol)
		}
	case ButtonHeavyPlow:
		handle(c, V{4, 6}, V{10, 18}, 2.4)
		c.Line(V{8, 14}, V{28, 22}, 3.6, RGB(150, 104, 58))
		c.poly([]V{{22, 20}, {34, 30}, {20, 30}}, diag(toolHead))
		c.Line(V{23, 21}, V{33, 30}, 1, RGB(250, 252, 255))
		btnWheel(c, V{12, 27}, 5)
	case ButtonForging:
		anvilIcon(c, 18)
		handle(c, V{30, 15}, V{20, 4}, 2.6)
		c.Line(V{16, 7}, V{24, 1}, 4.5, Shade(toolHead, 0.8))
		c.Line(V{16.5, 6}, V{23.5, 0.5}, 1.2, Shade(toolHead, 1.25))
		for _, p := range []V{{8, 14}, {12, 11}, {26, 13}} {
			c.Ellipse(p, 1, 1, solid(RGB(255, 200, 80)))
		}
	case ButtonIronCasting:
		c.Quad(V{12, 24}, V{20, 0}, V{0, 7}, solid(RGB(70, 70, 76)))
		c.Quad(V{14, 24}, V{16, 0}, V{0, 2.5}, func(u, v float64) color.RGBA { return Mix(RGB(255, 236, 140), RGB(230, 90, 20), u) })
		c.Line(V{14, 13}, V{19, 24}, 2.6, RGB(255, 170, 40))
		c.Line(V{14.5, 13}, V{19, 23}, 1, RGB(255, 240, 170))
		c.Ellipse(V{10, 11}, 7, 5, func(nx, ny float64) color.RGBA {
			if ny < -0.2 && nx*nx+ny*ny < 0.6 {
				return RGB(255, 140, 30)
			}
			return Shade(RGB(80, 80, 88), Lit(nx, ny))
		})
		handle(c, V{4, 7}, V{1, 2}, 2)
	case ButtonBlastFurnace:
		brick := func(x, y float64) color.RGBA {
			row := math.Floor(y / 3)
			if math.Mod(y, 3) < 0.7 || math.Mod(x+math.Mod(row, 2)*3, 6) < 0.7 {
				return RGB(90, 60, 50)
			}
			return Shade(RGB(170, 86, 60), 1.15-0.012*x)
		}
		c.poly([]V{{9, 8}, {27, 8}, {31, 33}, {5, 33}}, brick)
		c.Ellipse(V{18, 24}, 5.5, 6, func(nx, ny float64) color.RGBA {
			return Mix(RGB(255, 240, 150), RGB(220, 70, 20), math.Min(1, nx*nx+ny*ny+0.2*ny))
		})
		for i, x := range []float64{12, 18, 24} {
			h := 7.0 + float64(i%2)*3
			c.poly([]V{{x - 3, 8}, {x + 3, 8}, {x, 8 - h}}, func(px, py float64) color.RGBA {
				return Mix(RGB(255, 220, 90), RGB(230, 80, 20), (8-py)/h)
			})
		}
	case ButtonFletching:
		arrowDiag(c, V{4, 32}, V{32, 4}, 7, 3.5, RGB(240, 236, 226))
		for i := range 3 {
			p := V{7 + float64(i)*1.5, 25 - float64(i)*1.5}
			c.Line(p, p.Add(V{2.5, 2.5}), 0.8, RGB(200, 60, 50))
		}
	case ButtonBodkinArrow:
		for i, o := range []float64{-4, 0, 4} {
			arrowDiag(c, V{5 + o, 33 - o*0.2}, V{30 + o*0.3, 6 + o}, 9, 1.8, []color.RGBA{RGB(240, 236, 226), RGB(200, 60, 50), RGB(240, 236, 226)}[i])
		}
		c.Line(V{9, 20}, V{18, 27}, 2, RGB(120, 80, 44))
	case ButtonScaleMail:
		shirt(c, scales(RGB(196, 146, 64)))
	case ButtonChainMail:
		shirt(c, chain(RGB(170, 174, 186)))
	case ButtonPaddedArmor:
		shirt(c, func(x, y float64) color.RGBA {
			if math.Mod(x+y, 5) < 0.8 || math.Mod(x-y+40, 5) < 0.8 {
				return RGB(190, 172, 130)
			}
			return RGB(232, 216, 176)
		})
	case ButtonScaleBarding:
		horseHead(c, scales(RGB(196, 146, 64)))
	case ButtonChainBarding:
		horseHead(c, chain(RGB(170, 174, 186)))
	case ButtonTownBell:
		bellShape(c)
	case ButtonUngarrison:
		st := RGB(178, 172, 160)
		c.Quad(V{3, 6}, V{18, 0}, V{0, 27}, func(u, v float64) color.RGBA { return stone(st)(u*18, v*27, 1.1-0.2*u) })
		c.mask(func(x, y float64) bool {
			return x > 7 && x < 17 && y < 33 && (y > 16 || math.Hypot(x-12, y-16) < 5)
		}, solid(RGB(34, 28, 24)))
		g := RGB(120, 200, 90)
		c.Quad(V{14, 20}, V{12, 0}, V{0, 6}, func(u, v float64) color.RGBA { return Shade(g, 1.15-0.3*v) })
		c.poly([]V{{25, 15}, {34, 23}, {25, 31}}, diag(g))
	}
}
