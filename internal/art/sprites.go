package art

import (
	"image/color"
	"math"
	"sort"
)

// Sprite conventions shared with the game:
//   - Terrain tiles are 64x32 diamonds.
//   - Building and resource sprites are s*64 px wide, and the centre of the
//     footprint diamond sits at (W/2, H - s*16).
//   - Unit sheets hold 8 rows of UnitFrames frames of UnitFW x UnitFH px,
//     with the feet at (UnitAX, UnitAY). Row d faces ground angle d*45
//     degrees clockwise from screen east; a ground offset (gx, gy) is drawn
//     at screen offset (gx, gy*0.5). Columns: 0 idle, 1-4 walk, 5-6 action
//     (wind-up, strike), 7 dead.
const (
	UnitFW, UnitFH = 48, 48
	UnitAX, UnitAY = 24, 42
	UnitFrames     = 8
	IconSize       = 12
	ButtonSize     = 36
)

var (
	outline = RGB(28, 22, 18)
	goldCol = RGB(232, 186, 50)
	steel   = RGB(170, 174, 186)
)

func solid(c color.RGBA) func(float64, float64) color.RGBA {
	return func(float64, float64) color.RGBA { return c }
}

// Capsule fills the set of points within rad of segment a-b. fn gets the
// offset from the nearest point of the segment, normalised to rad.
func (c *Canvas) Capsule(a, b V, rad float64, fn func(nx, ny float64) color.RGBA) {
	x0, y0, x1, y1 := bbox(a, b)
	ri := int(math.Ceil(rad))
	d := b.Sub(a)
	l2 := d.X*d.X + d.Y*d.Y
	for py := y0 - ri; py <= y1+ri; py++ {
		for px := x0 - ri; px <= x1+ri; px++ {
			p := V{float64(px) + 0.5, float64(py) + 0.5}
			t := 0.0
			if l2 > 0 {
				t = math.Max(0, math.Min(1, ((p.X-a.X)*d.X+(p.Y-a.Y)*d.Y)/l2))
			}
			q := p.Sub(a.Add(d.Mul(t))).Mul(1 / rad)
			if q.X*q.X+q.Y*q.Y <= 1 {
				c.Blend(px, py, fn(q.X, q.Y))
			}
		}
	}
}

// Isometric building geometry, in world units (32 per tile).

type iso struct {
	c      *Canvas
	ox, oy float64
}

func newIso(size int, top float64) iso {
	c := New(size*64, int(top)+size*32)
	return iso{c, float64(size * 32), top}
}

func (s iso) P(x, y, z float64) V { return V{s.ox + x - y, s.oy + (x+y)/2 - z} }

type pattern func(u, v, light float64) color.RGBA

func flat(c color.RGBA) pattern {
	return func(u, v, light float64) color.RGBA { return Shade(c, light) }
}

// WallY draws the face of plane y that faces the viewer's lower left.
func (s iso) WallY(x0, x1, y, z0, z1 float64, pat pattern) {
	p := s.P(x0, y, z0)
	s.c.Quad(p, s.P(x1, y, z0).Sub(p), s.P(x0, y, z1).Sub(p), func(u, v float64) color.RGBA {
		return pat(u*(x1-x0), v*(z1-z0), 0.92+0.1*v)
	})
}

// WallX draws the face of plane x that faces the viewer's lower right.
func (s iso) WallX(x, y0, y1, z0, z1 float64, pat pattern) {
	p := s.P(x, y1, z0)
	s.c.Quad(p, s.P(x, y0, z0).Sub(p), s.P(x, y1, z1).Sub(p), func(u, v float64) color.RGBA {
		return pat(u*(y1-y0), v*(z1-z0), 0.68+0.08*v)
	})
}

func (s iso) Top(x0, y0, x1, y1, z float64, pat pattern) {
	p := s.P(x0, y0, z)
	s.c.Quad(p, s.P(x1, y0, z).Sub(p), s.P(x0, y1, z).Sub(p), func(u, v float64) color.RGBA {
		return pat(u*(x1-x0), v*(y1-y0), 1.12)
	})
}

func (s iso) Box(x0, y0, x1, y1, z0, z1 float64, wall, top pattern) {
	s.WallY(x0, x1, y1, z0, z1, wall)
	s.WallX(x1, y0, y1, z0, z1, wall)
	s.Top(x0, y0, x1, y1, z1, top)
}

// GableX draws a gable roof with its ridge running along x.
func (s iso) GableX(x0, y0, x1, y1, z, zr float64, roof, gable pattern) {
	ym := (y0 + y1) / 2
	slope := math.Hypot(ym-y0, zr-z)
	p := s.P(x0, y0, z)
	s.c.Quad(p, s.P(x1, y0, z).Sub(p), s.P(x0, ym, zr).Sub(p), func(u, v float64) color.RGBA {
		return roof(u*(x1-x0), v*slope, 0.8)
	})
	p = s.P(x1-4, y1, z)
	s.c.Tri(p, s.P(x1-4, y0, z).Sub(p), s.P(x1-4, ym, zr).Sub(p), func(u, v float64) color.RGBA {
		return gable(u*(y1-y0), v*(zr-z), 0.72)
	})
	p = s.P(x0, y1, z)
	s.c.Quad(p, s.P(x1, y1, z).Sub(p), s.P(x0, ym, zr).Sub(p), func(u, v float64) color.RGBA {
		return roof(u*(x1-x0), v*slope, 1.05)
	})
}

// GableY draws a gable roof with its ridge running along y.
func (s iso) GableY(x0, y0, x1, y1, z, zr float64, roof, gable pattern) {
	xm := (x0 + x1) / 2
	slope := math.Hypot(xm-x0, zr-z)
	p := s.P(x0, y0, z)
	s.c.Quad(p, s.P(x0, y1, z).Sub(p), s.P(xm, y0, zr).Sub(p), func(u, v float64) color.RGBA {
		return roof(u*(y1-y0), v*slope, 1.0)
	})
	p = s.P(x0, y1-4, z)
	s.c.Tri(p, s.P(x1, y1-4, z).Sub(p), s.P(xm, y1-4, zr).Sub(p), func(u, v float64) color.RGBA {
		return gable(u*(x1-x0), v*(zr-z), 1)
	})
	p = s.P(x1, y0, z)
	s.c.Quad(p, s.P(x1, y1, z).Sub(p), s.P(xm, y0, zr).Sub(p), func(u, v float64) color.RGBA {
		return roof(u*(y1-y0), v*slope, 0.74)
	})
}

// LeanTo draws a single-pitch roof that falls from zb at y0 to zf at y1.
func (s iso) LeanTo(x0, y0, x1, y1, zb, zf float64, roof pattern) {
	p := s.P(x0, y0, zb)
	l := math.Hypot(y1-y0, zb-zf)
	s.c.Quad(p, s.P(x1, y0, zb).Sub(p), s.P(x0, y1, zf).Sub(p), func(u, v float64) color.RGBA {
		return roof(u*(x1-x0), v*l, 1.05)
	})
	s.WallY(x0, x1, y1, zf-1.5, zf, roof)
	s.c.Tri(s.P(x1, y1, zf), s.P(x1, y0, zf).Sub(s.P(x1, y1, zf)), s.P(x1, y0, zb).Sub(s.P(x1, y1, zf)), func(u, v float64) color.RGBA {
		return roof(u*10, v*10, 0.7)
	})
}

func (s iso) Pyramid(x0, y0, x1, y1, z, zr float64, roof pattern) {
	apex := s.P((x0+x1)/2, (y0+y1)/2, zr)
	face := func(a, b V, light float64) {
		s.c.Tri(a, b.Sub(a), apex.Sub(a), func(u, v float64) color.RGBA {
			return roof(u*40, v*30, light)
		})
	}
	face(s.P(x0, y0, z), s.P(x1, y0, z), 0.8)
	face(s.P(x0, y1, z), s.P(x0, y0, z), 0.85)
	face(s.P(x1, y1, z), s.P(x1, y0, z), 0.72)
	face(s.P(x0, y1, z), s.P(x1, y1, z), 1.05)
}

// PyramidTrim outlines the visible edges of a pyramid roof.
func (s iso) PyramidTrim(x0, y0, x1, y1, z, zr float64, col color.RGBA) {
	apex := s.P((x0+x1)/2, (y0+y1)/2, zr)
	for _, p := range []V{s.P(x0, y1, z), s.P(x1, y1, z), s.P(x1, y0, z)} {
		s.c.Line(p, apex, 1.3, col)
	}
	s.c.Line(s.P(x0, y1, z), s.P(x1, y1, z), 1.3, col)
	s.c.Line(s.P(x1, y1, z), s.P(x1, y0, z), 1.3, col)
}

// light returns the shading factor for a vertical face with ground normal n.
func faceLight(nx, ny float64) float64 { return 0.86 - 0.14*nx + 0.14*ny }

type seg struct {
	a0, a1, nx, ny float64
}

func ring(n int) []seg {
	segs := make([]seg, 0, n)
	for i := range n {
		a0 := 2 * math.Pi * float64(i) / float64(n)
		a1 := 2 * math.Pi * float64(i+1) / float64(n)
		am := (a0 + a1) / 2
		segs = append(segs, seg{a0, a1, math.Cos(am), math.Sin(am)})
	}
	sort.Slice(segs, func(i, j int) bool { return segs[i].nx+segs[i].ny < segs[j].nx+segs[j].ny })
	return segs
}

// Cylinder draws a round tower body with a flat top.
func (s iso) Cylinder(cx, cy, r, z0, z1 float64, wall, top pattern) {
	for _, g := range ring(12) {
		if g.nx+g.ny < -0.3 {
			continue
		}
		p0 := s.P(cx+r*math.Cos(g.a0), cy+r*math.Sin(g.a0), z0)
		p1 := s.P(cx+r*math.Cos(g.a1), cy+r*math.Sin(g.a1), z0)
		light := faceLight(g.nx, g.ny)
		a0, a1 := g.a0, g.a1
		s.c.Quad(p0.Sub(p1.Sub(p0).Mul(0.04)), p1.Sub(p0).Mul(1.08), V{0, -(z1 - z0)}, func(u, v float64) color.RGBA {
			return wall(r*(a0+u*(a1-a0)), v*(z1-z0), light)
		})
	}
	if top != nil {
		s.c.Ellipse(s.P(cx, cy, z1), r*math.Sqrt2, r/math.Sqrt2, func(nx, ny float64) color.RGBA {
			return top((nx+1)*r, (ny+1)*r, 1.12)
		})
	}
}

// Cone draws a conical roof over a round tower.
func (s iso) Cone(cx, cy, r, z, zr float64, roof pattern) {
	apex := s.P(cx, cy, zr)
	sl := math.Hypot(r, zr-z)
	for _, g := range ring(10) {
		p0 := s.P(cx+r*math.Cos(g.a0), cy+r*math.Sin(g.a0), z)
		p1 := s.P(cx+r*math.Cos(g.a1), cy+r*math.Sin(g.a1), z)
		light := 0.95 - 0.12*g.nx + 0.14*g.ny
		a0, a1 := g.a0, g.a1
		s.c.Tri(p0, p1.Sub(p0).Mul(1.1), apex.Sub(p0), func(u, v float64) color.RGBA {
			return roof(r*(a0+u*(a1-a0))*(1-v), (1-v)*sl, light)
		})
	}
}

// RoundMerlons puts crenellations around the rim of a round tower.
func (s iso) RoundMerlons(cx, cy, r, z float64, wall pattern) {
	type m struct{ x, y, d float64 }
	var ms []m
	for i := range 12 {
		a := 2 * math.Pi * float64(i) / 12
		x, y := cx+(r-1.5)*math.Cos(a), cy+(r-1.5)*math.Sin(a)
		ms = append(ms, m{x, y, x + y})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].d < ms[j].d })
	for _, q := range ms {
		s.Box(q.x-1.8, q.y-1.8, q.x+1.8, q.y+1.8, z, z+5, wall, wall)
	}
}

// Merlons puts crenellations along the edges of a rectangular wall top.
func (s iso) Merlons(x0, y0, x1, y1, z float64, wall pattern, back bool) {
	m := func(x0, y0, x1, y1 float64) { s.Box(x0, y0, x1, y1, z, z+5, wall, wall) }
	if back {
		for x := x0 + 1; x < x1-3; x += 8 {
			m(x, y0, x+4, y0+3)
		}
		for y := y0 + 1; y < y1-3; y += 8 {
			m(x0, y, x0+3, y+4)
		}
		return
	}
	for x := x0 + 1; x < x1-3; x += 8 {
		m(x, y1-3, x+4, y1)
	}
	for y := y0 + 1; y < y1-3; y += 8 {
		m(x1-3, y, x1, y+4)
	}
}

func (s iso) Flag(base V, height float64, team color.RGBA) {
	top := base.Add(V{0, -height})
	s.c.Line(base, top, 1.6, RGB(70, 50, 30))
	s.c.Quad(top.Add(V{1, 0}), V{12, 2}, V{0, 7}, func(u, v float64) color.RGBA {
		return Shade(team, 1.05-0.25*u+0.15*math.Sin(u*9))
	})
}

// Banners hang on walls: bannerY on a y-plane face, bannerX on an x-plane face.
func banner(team color.RGBA, trim color.RGBA) pattern {
	return func(u, v, light float64) color.RGBA {
		if v < 1.2 {
			return Shade(trim, light)
		}
		if u > 2.5 && u < 5.5 && v > 6 && v < 12 {
			return Shade(Mix(team, RGB(250, 240, 200), 0.75), light)
		}
		return Shade(team, light*(0.92+0.08*math.Sin(u*1.3)))
	}
}

func (s iso) BannerY(x, y, z0, z1 float64, team color.RGBA) {
	s.WallY(x, x+8, y+0.3, z0, z1, banner(team, goldCol))
	tip := s.P(x+4, y+0.3, z0-4)
	s.c.Tri(s.P(x, y+0.3, z0+0.5), V{8, 4}, tip.Sub(s.P(x, y+0.3, z0+0.5)), func(u, v float64) color.RGBA { return team })
}

func (s iso) BannerX(x, y, z0, z1 float64, team color.RGBA) {
	s.WallX(x+0.3, y, y+8, z0, z1, banner(team, goldCol))
	p := s.P(x+0.3, y+8, z0+0.5)
	s.c.Tri(p, s.P(x+0.3, y, z0+0.5).Sub(p), s.P(x+0.3, y+4, z0-4).Sub(p), func(u, v float64) color.RGBA { return Shade(team, 0.72) })
}

func noise(u, v float64, seed uint32) float64 { return Hash(int(u), int(v), seed) - 0.5 }

func stone(base color.RGBA) pattern {
	return func(u, v, light float64) color.RGBA {
		row := int(math.Floor(v / 6))
		off := float64(row%2) * 6
		col := int(math.Floor((u + off) / 12))
		fu, fv := math.Mod(u+off, 12), math.Mod(v, 6)
		if fv < 0.9 || fu < 0.9 {
			return Shade(base, 0.78*light)
		}
		f := light * (0.95 + 0.1*Hash(col, row, 11))
		if fv > 4.8 {
			f *= 0.92
		}
		return Shade(base, f)
	}
}

func planks(base color.RGBA) pattern {
	return func(u, v, light float64) color.RGBA {
		row := int(math.Floor(v / 5))
		fv := math.Mod(v, 5)
		if fv < 0.9 {
			return Shade(base, 0.7*light)
		}
		return Shade(base, light*(0.93+0.1*Hash(row, int(u/24), 21)+0.04*(fv-2.5)/2.5))
	}
}

// vplanks are vertical boards, as on barns and palisades.
func vplanks(base color.RGBA) pattern {
	return func(u, v, light float64) color.RGBA {
		col := int(math.Floor(u / 4))
		fu := math.Mod(u, 4)
		if fu < 0.8 {
			return Shade(base, 0.72*light)
		}
		return Shade(base, light*(0.93+0.1*Hash(col, 0, 23)-0.03*fu/4))
	}
}

func logs(base color.RGBA) pattern {
	return func(u, v, light float64) color.RGBA {
		row := math.Mod(v, 4)
		f := 0.8 + 0.3*math.Sin(row/4*math.Pi)
		return Shade(base, light*f)
	}
}

func timber(plaster, beam color.RGBA, w, h float64) pattern {
	return func(u, v, light float64) color.RGBA {
		if u < 2 || u > w-2 || v < 2 || v > h-2 || int(u)%16 < 2 {
			return Shade(beam, light)
		}
		return Shade(plaster, light*(0.96+0.04*v/h))
	}
}

func tiles(base color.RGBA) pattern {
	return func(u, v, light float64) color.RGBA {
		row := int(math.Floor(v / 5))
		off := float64(row%2) * 4
		fv := math.Mod(v, 5)
		if fv > 4.1 {
			return Shade(base, 0.68*light)
		}
		if math.Mod(u+off, 8) < 0.9 {
			return Shade(base, 0.86*light)
		}
		return Shade(base, light*(0.9+0.06*Hash(int((u+off)/8), row, 41)+0.12*fv/5))
	}
}

func thatch(base color.RGBA) pattern {
	return func(u, v, light float64) color.RGBA {
		fv := math.Mod(v, 6)
		f := light * (0.86 + 0.1*Smooth(u/2, v/6, 43) + 0.14*fv/6)
		return Shade(base, f)
	}
}

func ground(base color.RGBA) pattern {
	return func(u, v, light float64) color.RGBA {
		return Shade(base, light*(0.93+0.1*Smooth(u/9, v/9, 51)))
	}
}

// teamRoof is a tiled roof in the team colour.
func teamRoof(team color.RGBA) pattern { return tiles(Shade(team, 0.95)) }

func (s iso) plinth(size float64, top pattern) {
	edge := stone(RGB(130, 120, 100))
	s.WallY(0, size, size, 0, 3, edge)
	s.WallX(size, 0, size, 0, 3, edge)
	s.Top(0, 0, size, size, 3, top)
}

func (s iso) window(x, y, z0, z1 float64, onX bool) {
	glass := func(u, v, light float64) color.RGBA {
		if v > (z1-z0)-1.5 {
			return Shade(RGB(110, 100, 85), light)
		}
		return Shade(RGB(40, 45, 60), light)
	}
	if onX {
		s.WallX(x+0.2, y, y+6, z0, z1, glass)
	} else {
		s.WallY(x, x+6, y+0.2, z0, z1, glass)
	}
}

// slit is a narrow arrow slit in a stone wall.
func (s iso) slit(x, y, z0, z1 float64, onX bool) {
	dark := flat(RGB(30, 30, 38))
	if onX {
		s.WallX(x+0.2, y, y+2, z0, z1, dark)
	} else {
		s.WallY(x, x+2, y+0.2, z0, z1, dark)
	}
}

func (s iso) door(x0, x1, y, z0, z1 float64) {
	s.WallY(x0, x1, y+0.2, z0, z1, func(u, v, light float64) color.RGBA {
		if u < 1 || u > x1-x0-1 || v > z1-z0-1 {
			return Shade(RGB(70, 60, 50), light)
		}
		return planks(RGB(100, 64, 36))(v, u, light)
	})
}

func (s iso) doorX(x, y0, y1, z0, z1 float64) {
	s.WallX(x+0.2, y0, y1, z0, z1, func(u, v, light float64) color.RGBA {
		if u < 1 || u > y1-y0-1 || v > z1-z0-1 {
			return Shade(RGB(70, 60, 50), light)
		}
		return planks(RGB(100, 64, 36))(v, u, light)
	})
}
