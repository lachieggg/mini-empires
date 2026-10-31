package art

import (
	"image/color"
	"math"
	"sort"
)

const NumTrees = 6

func leaf(base color.RGBA, seed uint32) func(nx, ny float64) color.RGBA {
	return func(nx, ny float64) color.RGBA {
		f := Facet(nx, ny)
		if nx*nx+ny*ny > 0.85 {
			f *= 0.9
		}
		return Shade(base, f)
	}
}

type blob struct{ x, y, r float64 }

// Tree variants: 0 oak, 1 light oak, 2 pine, 3 birch, 4 palm, 5 autumn.
func Tree(variant int) *Canvas {
	c := New(64, 96)
	g := V{32, 80}
	groundShadow(c, g.Add(V{7, 1}), 19, 7)
	bark := RGB(112, 76, 44)
	trunk := func(w, h float64, col color.RGBA, marks bool) {
		c.Quad(g.Add(V{-w / 2, -h}), V{w, 0}, V{0, h + 1}, func(u, v float64) color.RGBA {
			if marks && math.Mod(v*h+u*3, 7) < 1.2 && Hash(int(v*h/7), 0, 7) < 0.7 {
				return RGB(60, 54, 48)
			}
			f := 1.12
			if u > 0.5 {
				f = 0.8
			}
			return Shade(col, f-0.08*(1-v))
		})
	}
	switch variant {
	case 2:
		trunk(6, 24, bark, false)
		pine := RGB(36, 112, 64)
		for i, w := range []float64{22, 17, 12} {
			base := g.Y - 18 - float64(i)*14
			top := base - 24
			c.Tri(V{g.X - w, base}, V{2 * w, 0}, V{w, top - base}, func(u, v float64) color.RGBA {
				s := u + 0.5*v
				f := 1.16
				switch {
				case s > 0.68:
					f = 0.74
				case s > 0.38:
					f = 0.95
				}
				return Shade(pine, f*(0.88+0.14*v))
			})
		}
	case 3:
		trunk(4, 40, RGB(232, 228, 216), true)
		c.Line(g.Add(V{0, -30}), g.Add(V{-7, -40}), 1.5, RGB(214, 208, 196))
		c.Line(g.Add(V{0, -34}), g.Add(V{7, -44}), 1.5, RGB(214, 208, 196))
		col := RGB(132, 178, 62)
		for i, b := range []blob{{-7, -46, 9}, {7, -50, 9}, {0, -58, 10}, {-2, -44, 8}, {6, -40, 7}} {
			c.Ellipse(g.Add(V{b.x, b.y}), b.r, b.r*1.1, leaf(col, uint32(i+40)))
		}
	case 4:
		palm(c, g)
	default:
		trunk(6, 24, bark, false)
		green := RGB(52, 128, 46)
		blobs := []blob{{0, -50, 13}, {-10, -40, 13}, {10, -40, 13}, {0, -34, 15}}
		cols := []color.RGBA{green, green, green, green}
		switch variant {
		case 1:
			for i := range cols {
				cols[i] = RGB(84, 142, 44)
			}
		case 5:
			cols = []color.RGBA{RGB(232, 156, 38), RGB(208, 92, 30), RGB(224, 128, 34), RGB(214, 106, 32)}
		}
		for i, b := range blobs {
			c.Ellipse(g.Add(V{b.x, b.y}), b.r, b.r*0.95, leaf(cols[i], uint32(i+variant*5)))
		}
		if variant == 5 {
			for i := range 7 {
				p := g.Add(V{(Hash(i, 3, 61) - 0.5) * 26, 1 + Hash(i, 4, 61)*4 - 2})
				c.Ellipse(p, 1.6, 1, solid(RGB(210, 116, 36)))
			}
		}
	}
	finish(c)
	return c
}

func palm(c *Canvas, g V) {
	const segs = 10
	crown := g.Add(V{9, -56})
	pt := func(t float64) V { return V{g.X + 9*t*t*t + 2*t, g.Y - 56*t} }
	for i := range segs {
		t0, t1 := float64(i)/segs, float64(i+1)/segs
		a, b := pt(t0), pt(t1)
		w0, w1 := 3.2-1.2*t0, 3.2-1.2*t1+0.9
		bark := RGB(170, 126, 78)
		c.Quad(a.Add(V{-w0, 0}), V{w0, 0}, b.Sub(a).Add(V{-w1 + w0, 0}), solid(Shade(bark, 1.1)))
		c.Quad(a, V{w0, 0}, b.Sub(a).Add(V{w1 - w0, 0}), solid(Shade(bark, 0.78)))
		c.Line(b.Add(V{-w1, 0.5}), b.Add(V{w1, 0}), 1, RGB(96, 64, 38))
	}
	type frond struct{ dx, dy, ln, shade float64 }
	var fronds []frond
	for i := range 8 {
		a := float64(i)*2*math.Pi/8 + 0.3
		dx, dy := math.Cos(a), math.Sin(a)*0.45
		ln := 21 + 5*math.Abs(dx) + 3*Hash(i, 0, 91)
		fronds = append(fronds, frond{dx, dy, ln, 0.95 + 0.6*dy - 0.1*dx})
	}
	sort.Slice(fronds, func(i, j int) bool { return fronds[i].dy < fronds[j].dy })
	leafC := RGB(58, 150, 52)
	for i, f := range fronds {
		col := Mix(leafC, RGB(110, 170, 50), 0.25*Hash(i, 1, 92))
		prev := crown
		for s := 1; s <= 9; s++ {
			t := float64(s) / 9
			p := crown.Add(V{f.dx * f.ln * t, f.dy*f.ln*t - 9*t + 22*t*t})
			d := p.Sub(prev)
			dl := math.Max(0.01, math.Hypot(d.X, d.Y))
			n := V{-d.Y, d.X}.Mul(1 / dl)
			fw := d.Mul(1 / dl)
			l := 6.5*(1-t*0.8) + 0.5
			for _, side := range []float64{-1, 1} {
				tip := p.Add(n.Mul(side * l)).Add(fw.Mul(2.2)).Add(V{0, 2 + 2*t})
				c.Line(p, tip, 1.3, Shade(col, f.shade*(1.08-0.2*t)*(1+0.1*side)))
			}
			c.Line(prev, p, 1.8-0.9*t, Shade(RGB(80, 120, 40), f.shade))
			prev = p
		}
	}
	for _, o := range []V{{-3, 6.5}, {1.8, 7}, {-0.5, 9.5}, {3.8, 5}} {
		c.Ellipse(crown.Add(o), 2.5, 2.4, lit(RGB(124, 80, 42)))
	}
}

func mine(ore func(c *Canvas, g V)) *Canvas {
	c := New(64, 56)
	g := V{32, 40}
	groundShadow(c, g.Add(V{4, 2}), 24, 9)
	rock := RGB(128, 120, 108)
	for _, r := range []struct{ x, y, rx, ry float64 }{{-12, -6, 10, 8}, {10, -6, 11, 8}, {0, -12, 12, 10}, {-4, 0, 11, 7}, {9, 1, 9, 6}} {
		c.Ellipse(g.Add(V{r.x, r.y}), r.rx, r.ry, lit(rock))
	}
	ore(c, g)
	finish(c)
	return c
}

var oreSpots = []blob{{-8, -10, 3.5}, {4, -16, 3}, {9, -5, 4}, {-2, -3, 3}, {-13, -2, 2.5}, {14, -10, 2.5}, {2, -9, 2.5}}

// goldShade maps a light factor onto the gold palette: orange-brown in
// shadow, warm yellow, white-yellow highlights.
func goldShade(f float64) color.RGBA {
	switch {
	case f > 0.97:
		return RGB(255, 248, 190)
	case f > 0.78:
		return RGB(255, 214, 58)
	case f > 0.64:
		return RGB(236, 166, 34)
	}
	return RGB(166, 98, 26)
}

// nugget draws a chunky faceted gold nugget around ctr.
func nugget(c *Canvas, ctr V, r float64, seed int) {
	const n = 6
	rot := Hash(seed, 0, 93) * math.Pi
	var vs [n]V
	for i := range n {
		a := rot + float64(i)*2*math.Pi/n
		j := 0.8 + 0.35*Hash(seed, i+1, 94)
		vs[i] = ctr.Add(V{math.Cos(a) * r * j, math.Sin(a) * r * 0.85 * j})
	}
	top := ctr.Add(V{-0.2 * r, -0.3 * r})
	for i := range n {
		a, b := vs[i], vs[(i+1)%n]
		m := a.Add(b).Mul(0.5).Sub(ctr)
		ml := math.Max(0.01, math.Hypot(m.X, m.Y))
		col := goldShade(Lit(0.75*m.X/ml, 0.75*m.Y/ml))
		c.Tri(top, a.Sub(top), b.Sub(top), solid(col))
	}
	c.Ellipse(top, r*0.45, r*0.36, solid(goldShade(1.1)))
	c.Blend(int(top.X-r*0.2), int(top.Y-r*0.15), RGB(255, 255, 235))
}

func GoldMine() *Canvas {
	c := New(64, 56)
	g := V{32, 40}
	groundShadow(c, g.Add(V{4, 2}), 25, 9)
	rock := RGB(128, 114, 98)
	for _, r := range []struct{ x, y, rx, ry float64 }{{-14, -4, 10, 6.5}, {12, -4, 10.5, 6.5}, {0, -8, 13, 8}, {-5, 1, 11, 5.5}, {8, 2, 9.5, 5}} {
		c.Ellipse(g.Add(V{r.x, r.y}), r.rx, r.ry, lit(rock))
	}
	for i, n := range []blob{{-6, -15, 5.5}, {5, -16, 7.5}, {-13, -9, 6.5}, {13, -9, 6.2}, {-1, -6, 6}, {8, -2, 5}, {-9, -1, 4.5}, {17, -2, 3.8}, {-18, -3, 3.8}} {
		nugget(c, g.Add(V{n.x, n.y}), n.r, i)
	}
	finish(c)
	return c
}

func StoneMine() *Canvas {
	return mine(func(c *Canvas, g V) {
		for i, n := range oreSpots {
			p := g.Add(V{n.x, n.y})
			r := n.r + 0.8
			col := RGB(196, 196, 192)
			if i%3 == 1 {
				col = RGB(176, 178, 182)
			}
			c.Quad(p.Add(V{-r, -r * 0.4}), V{r, -r * 0.5}, V{r, r * 0.5}, solid(Shade(col, 1.12)))
			c.Quad(p.Add(V{-r, -r * 0.4}), V{r, r * 0.5}, V{0, r * 0.9}, solid(Shade(col, 0.95)))
			c.Quad(p.Add(V{0, r * 0.1}), V{r, -r * 0.5}, V{0, r * 0.9}, solid(Shade(col, 0.75)))
		}
	})
}

func Berries() *Canvas {
	c := New(64, 48)
	g := V{32, 32}
	groundShadow(c, g.Add(V{4, 2}), 20, 7)
	bush := RGB(46, 108, 44)
	for _, b := range []blob{{-9, -8, 10}, {9, -8, 10}, {0, -14, 11}, {0, -5, 11}} {
		c.Ellipse(g.Add(V{b.x, b.y}), b.r, b.r*0.85, lit(bush))
	}
	for i := 0; i < 14; i++ {
		p := g.Add(V{(Hash(i, 1, 5) - 0.5) * 30, -4 - Hash(i, 2, 5)*18})
		c.Ellipse(p, 1.8, 1.8, lit(RGB(205, 35, 70)))
		c.Blend(int(p.X), int(p.Y-1), RGB(255, 170, 190))
	}
	finish(c)
	return c
}

// Farm stages: 0 full crops, 1 half harvested, 2 nearly bare.
func Farm(stage int) *Canvas {
	s := newIso(2, 10)
	const S = 64.0
	soil := RGB(118, 84, 52)
	s.WallY(0, S, S, 0, 2, flat(Shade(soil, 0.8)))
	s.WallX(S, 0, S, 0, 2, flat(Shade(soil, 0.8)))
	s.Top(0, 0, S, S, 2, func(u, v, light float64) color.RGBA {
		if u < 2 || v < 2 || u > S-2 || v > S-2 {
			return Shade(RGB(140, 112, 70), light*0.9)
		}
		f := light * (0.9 + 0.12*Smooth(u/4, v/4, 71))
		if math.Mod(v, 6) < 2 {
			f *= 0.78
		}
		return Shade(soil, f)
	})
	wheat := RGB(214, 182, 72)
	for y := 5.0; y < S-3; y += 5 {
		for x := 4.0; x < S-3; x += 2 {
			j := Hash(int(x), int(y), 72)
			cut := false
			switch stage {
			case 1:
				cut = x+y*0.4 > 40+4*j
			case 2:
				cut = j > 0.12
			}
			b := s.P(x, y, 2)
			if cut {
				s.c.Blend(int(b.X), int(b.Y)-1, RGB(176, 152, 84))
				continue
			}
			col := Mix(RGB(150, 160, 60), wheat, 0.55+0.45*j)
			s.c.Ellipse(b.Add(V{0, -3.5}), 1.8, 3.5, lit(col))
			if j > 0.5 {
				s.c.Blend(int(b.X), int(b.Y)-6, RGB(245, 220, 130))
			}
		}
	}
	finish(s.c)
	return s.c
}
