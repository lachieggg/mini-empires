package art

import (
	"image/color"
	"math"
)

// Siege sheets use larger frames than units, with the same 8 x 8 layout.
const (
	SiegeFW, SiegeFH = 64, 64
	SiegeAX, SiegeAY = 32, 54
)

// Siege selects a siege engine sheet.
type Siege int

const (
	Ram Siege = iota
	Mangonel
	Trebuchet
	NumSiege
)

var siegeNames = [NumSiege]string{"ram", "mangonel", "trebuchet"}

func (s Siege) String() string { return siegeNames[s] }

// SiegeSheet renders the 8 directions x UnitFrames frames of a siege engine.
func SiegeSheet(kind Siege, team color.RGBA) *Canvas {
	sheet := New(SiegeFW*UnitFrames, SiegeFH*8)
	for d := range 8 {
		for fr := range UnitFrames {
			fc := New(SiegeFW, SiegeFH)
			k := newRig(fc, V{SiegeAX, SiegeAY}, d)
			a := animFor(fr)
			switch {
			case a.dead:
				wreck(k, kind, team)
			case kind == Ram:
				ram(k, a, team)
			case kind == Mangonel:
				mangonel(k, a, team)
			default:
				trebuchet(k, a, team)
			}
			k.flush()
			finish(fc)
			sheet.Blit(fc, fr*SiegeFW, d*SiegeFH)
		}
	}
	return sheet
}

// obox draws a box aligned with the rig's facing: centre g, half extents a
// along f and b along r, from height h0 to h1.
func (k *rig) obox(g V, a, b, h0, h1 float64, pat pattern) {
	f, r := k.f, k.r
	cs := [4]V{g.Add(f.Mul(a)).Add(r.Mul(b)), g.Add(f.Mul(-a)).Add(r.Mul(b)), g.Add(f.Mul(-a)).Add(r.Mul(-b)), g.Add(f.Mul(a)).Add(r.Mul(-b))}
	norms := [4]V{r, f.Mul(-1), r.Mul(-1), f}
	lens := [4]float64{2 * a, 2 * b, 2 * a, 2 * b}
	k.add(g.Y+(h0+h1)*0.01, func() {
		for i := range 4 {
			n := norms[(i)%4]
			if n.Y <= 0.01 {
				continue
			}
			p0, p1 := cs[i], cs[(i+1)%4]
			q0 := k.P(p0, h0)
			e1 := k.P(p1, h0).Sub(q0)
			light := 0.86 - 0.16*n.X + 0.1*n.Y
			ln := lens[i]
			k.c.Quad(q0, e1, V{0, -(h1 - h0)}, func(u, v float64) color.RGBA {
				return pat(u*ln, v*(h1-h0), light)
			})
		}
		q := k.P(cs[0], h1)
		k.c.Quad(q, k.P(cs[1], h1).Sub(q), k.P(cs[3], h1).Sub(q), func(u, v float64) color.RGBA {
			return pat(u*2*a, v*2*b, 1.1)
		})
	})
}

// beam is a thick timber between two 3D points.
func (k *rig) beam(g0 V, h0 float64, g1 V, h1, w float64, col color.RGBA, depth float64) {
	k.add(depth, func() {
		k.c.Line(k.P(g0, h0), k.P(g1, h1), w, col)
		k.c.Line(k.P(g0, h0).Add(V{-0.4, -0.5}), k.P(g1, h1).Add(V{-0.4, -0.5}), w*0.35, Shade(col, 1.2))
	})
}

// wheel draws a spoked wheel at ground point g, axle height h, turned by
// spin radians.
func (k *rig) wheel(g V, h, rad, spin float64) {
	k.add(g.Y+0.05, func() {
		f := k.f
		pt := func(a, s float64) V { return k.P(g.Add(f.Mul(math.Cos(a)*rad*s)), h+math.Sin(a)*rad*s) }
		wood := RGB(110, 76, 44)
		for i := range 16 {
			a0, a1 := float64(i)*math.Pi/8, float64(i+1)*math.Pi/8
			k.c.Line(pt(a0, 1), pt(a1, 1), 2.2, Shade(wood, 0.9+0.2*math.Sin(a0+2)))
		}
		for i := range 4 {
			a := spin + float64(i)*math.Pi/4
			k.c.Line(pt(a, 1), pt(a+math.Pi, 1), 1.2, Shade(wood, 1.15))
		}
		k.c.Ellipse(pt(0, 0), 1.3, 1.3, solid(RGB(70, 70, 76)))
	})
}

func (k *rig) wheels(along, side, h, rad float64, a anim) {
	spin := float64(a.frame) * math.Pi / 8
	for _, s := range []float64{-1, 1} {
		for _, d := range []float64{-1, 1} {
			k.wheel(k.at(along*d, side*s), h, rad, spin)
		}
	}
}

func bobOf(a anim) float64 {
	if a.frame >= 1 && a.frame <= 4 {
		return 0.6 * math.Abs(math.Sin(float64(a.frame)*math.Pi/2))
	}
	return 0
}

func ram(k *rig, a anim, team color.RGBA) {
	f, r := k.f, k.r
	k.c.Ellipse(k.o, 22, 9, softShadow)
	b := bobOf(a)
	k.wheels(8, 8, 4.5, 4.5, a)
	wood := planks(RGB(140, 98, 58))
	k.obox(V{}, 12, 6.5, 4+b, 7+b, wood)
	push := 0.0
	switch a.act {
	case 0:
		push = -4
	case 1:
		push = 5
	}
	k.add(0.05, func() {
		k.c.Capsule(k.P(f.Mul(-9+push), 11+b), k.P(f.Mul(12+push), 11+b), 2.6, lit(RGB(130, 88, 50)))
		k.c.Ellipse(k.P(f.Mul(13+push), 11+b), 3.6, 3.6, lit(RGB(90, 92, 100)))
	})
	roof := func(sd float64) {
		lo, hi := r.Mul(7.5*sd), V{}
		k.add(lo.Y*0.5+0.1, func() {
			p := k.P(f.Mul(-12).Add(lo), 9+b)
			e1 := k.P(f.Mul(12).Add(lo), 9+b).Sub(p)
			e2 := k.P(f.Mul(-12).Add(hi), 21+b).Sub(p)
			light := 1.05 - 0.2*sd*r.X
			k.c.Quad(p, e1, e2, func(u, v float64) color.RGBA {
				if int(u*8)%2 == 0 && v < 0.35 {
					return Shade(team, light*0.92)
				}
				if v < 0.35 {
					return Shade(team, light)
				}
				return tiles(RGB(120, 86, 52))(u*24, v*12, light)
			})
		})
	}
	roof(-1)
	roof(1)
	for _, d := range []float64{-1, 1} {
		g := f.Mul(12 * d)
		if g.Y > 0 {
			k.add(g.Y+0.2, func() {
				p := k.P(g.Add(r.Mul(-7.5)), 9+b)
				k.c.Tri(p, k.P(g.Add(r.Mul(7.5)), 9+b).Sub(p), k.P(g, 21+b).Sub(p), solid(Shade(RGB(120, 84, 50), 0.85)))
			})
		}
	}
}

func mangonel(k *rig, a anim, team color.RGBA) {
	f, r := k.f, k.r
	k.c.Ellipse(k.o, 18, 7, softShadow)
	b := bobOf(a)
	k.wheels(7, 6, 3.5, 3.5, a)
	wood := RGB(150, 104, 58)
	k.obox(V{}, 10, 5, 3+b, 6+b, planks(wood))
	for _, s := range []float64{-1, 1} {
		g := f.Mul(2).Add(r.Mul(4.5 * s))
		k.beam(g, 6+b, g.Add(f.Mul(1)), 17+b, 2.4, wood, g.Y+0.1)
	}
	k.add(f.Mul(3).Y+0.15, func() {
		k.c.Line(k.P(f.Mul(3).Add(r.Mul(-5)), 17+b), k.P(f.Mul(3).Add(r.Mul(5)), 17+b), 3, Shade(team, 1))
	})
	k.add(f.Mul(-5).Y+0.12, func() {
		k.c.Line(k.P(f.Mul(-5).Add(r.Mul(-4.5)), 7+b), k.P(f.Mul(-5).Add(r.Mul(4.5)), 7+b), 4, RGB(170, 150, 110))
	})
	pivot := f.Mul(-5)
	tip, th := f.Mul(2.5), 17.5
	switch a.act {
	case 0:
		tip, th = f.Mul(-17), 5
	case 1:
		tip, th = f.Mul(1), 21
	}
	k.beam(pivot, 7+b, tip, th+b, 2.6, Shade(wood, 1.05), (pivot.Y+tip.Y)/2+0.13)
	k.add(tip.Y+0.2, func() {
		p := k.P(tip, th+b+1)
		k.c.Ellipse(p, 3, 2.2, lit(RGB(110, 76, 44)))
		if a.act != 1 {
			k.c.Ellipse(p.Add(V{0, -1.5}), 2.2, 2.2, lit(RGB(140, 136, 128)))
		}
	})
	if a.act == 1 {
		k.add(99, func() {
			k.c.Ellipse(k.P(f.Mul(9), 31), 2.4, 2.4, lit(RGB(140, 136, 128)))
		})
	}
	k.add(-50, func() {
		g := f.Mul(-9).Add(r.Mul(-4))
		k.c.Line(k.P(g, 6), k.P(g, 18), 1, darkWood)
		k.c.Tri(k.P(g, 18), V{5, 1}, V{0, 4}, solid(team))
	})
}

func trebuchet(k *rig, a anim, team color.RGBA) {
	f, r := k.f, k.r
	k.c.Ellipse(k.o, 22, 9, softShadow)
	b := bobOf(a)
	wood := RGB(150, 104, 58)
	k.obox(V{}, 12, 7, 1+b, 4+b, planks(wood))
	const ph = 30.0
	for _, s := range []float64{-1, 1} {
		side := r.Mul(6 * s)
		top := side
		dep := side.Y + 0.1
		k.beam(side.Add(f.Mul(-9)), 4+b, top, ph+b, 2.6, wood, dep)
		k.beam(side.Add(f.Mul(9)), 4+b, top, ph+b, 2.6, wood, dep)
		k.beam(side.Add(f.Mul(-6)), 14+b, side.Add(f.Mul(6)), 14+b, 1.8, Shade(wood, 0.9), dep)
	}
	k.add(0.12, func() {
		k.c.Line(k.P(r.Mul(-6.5), ph+b), k.P(r.Mul(6.5), ph+b), 2.4, RGB(80, 80, 86))
	})
	long, lh := f.Mul(-12), ph+18.0
	short, sh := f.Mul(5), ph-5.0
	switch a.act {
	case 0:
		long, lh = f.Mul(-18), 6
		short, sh = f.Mul(5), ph+6
	case 1:
		long, lh = f.Mul(13), ph+16
		short, sh = f.Mul(-4), ph-5
	}
	k.beam(short, sh+b, long, lh+b, 2.8, Shade(wood, 1.1), 0.14)
	k.add(short.Y+0.16, func() {
		cw := k.P(short, sh+b)
		k.c.Line(cw, cw.Add(V{0, 4}), 1.2, RGB(70, 70, 76))
		k.c.Quad(cw.Add(V{-5, 4}), V{10, 0}, V{0, 8}, func(u, v float64) color.RGBA {
			if v < 0.3 {
				return Shade(RGB(110, 80, 50), 1.1-0.3*u)
			}
			return Shade(team, 1.1-0.35*u)
		})
		k.c.Line(cw.Add(V{-5, 6.4}), cw.Add(V{5, 6.4}), 1, goldCol)
	})
	k.add(long.Y+0.17, func() {
		end := k.P(long, lh+b)
		sl := end.Add(V{0, 7})
		if a.act == 1 {
			sl = end.Add(V{4, -3})
		}
		k.c.Line(end, sl, 0.8, RGB(200, 190, 160))
		k.c.Ellipse(sl, 1.8, 1.4, lit(RGB(120, 90, 60)))
	})
	k.add(-50, func() {
		g := f.Mul(-11).Add(r.Mul(-7))
		k.c.Line(k.P(g, 3), k.P(g, 20), 1, darkWood)
		k.c.Tri(k.P(g, 20), V{6, 1}, V{0, 4.5}, solid(team))
	})
}

// wreck draws broken siege timbers lying on the ground.
func wreck(k *rig, kind Siege, team color.RGBA) {
	f, r := k.f, k.r
	k.c.Ellipse(k.o, 20, 7, softShadow)
	wood := RGB(120, 84, 50)
	for i := range 7 {
		a := float64(i)*0.9 + float64(kind)
		p := f.Mul(math.Cos(a) * (4 + float64(i))).Add(r.Mul(math.Sin(a) * 5))
		q := p.Add(f.Mul(math.Cos(a+1.3) * 9)).Add(r.Mul(math.Sin(a+1.3) * 4))
		k.beam(p, 1.5, q, 2.5+float64(i%3), 2.4, Shade(wood, 0.8+0.06*float64(i)), (p.Y+q.Y)/2)
	}
	k.add(10, func() {
		p := k.P(r.Mul(4).Add(f.Mul(-4)), 2)
		k.c.Ellipse(p, 4.5, 2.2, func(nx, ny float64) color.RGBA {
			if nx*nx+ny*ny < 0.35 {
				return RGB(70, 70, 76)
			}
			return Shade(wood, 0.95)
		})
		k.c.Quad(k.P(f.Mul(3), 1), V{7, 1}, V{1, -4}, solid(Shade(team, 0.75)))
		k.c.Ellipse(k.P(f.Mul(-2).Add(r.Mul(-3)), 6), 3.5, 3, func(nx, ny float64) color.RGBA {
			return color.RGBA{90, 90, 90, uint8(120 * (1 - nx*nx - ny*ny))}
		})
	})
}
