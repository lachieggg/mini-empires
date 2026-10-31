package art

import (
	"image/color"
	"math"
)

func (s iso) smoke(p V, n int) {
	for i := range n {
		t := float64(i)
		q := p.Add(V{2*t + 1.5*math.Sin(t*1.7), -6 * t})
		r := 3 + 1.2*t
		s.c.Ellipse(q, r, r*0.85, func(nx, ny float64) color.RGBA {
			g := uint8(150 + 50*Lit(nx, ny))
			return color.RGBA{g, g, g, uint8(170 - 25*t)}
		})
	}
}

func (s iso) glow(p V, rx, ry float64) {
	s.c.Ellipse(p, rx, ry, func(nx, ny float64) color.RGBA {
		d := 1 - (nx*nx + ny*ny)
		return color.RGBA{255, 170, 60, uint8(90 * d)}
	})
}

func (s iso) anvil(x, y float64) {
	iron := RGB(70, 72, 80)
	s.Box(x-2, y-2, x+2, y+2, 3, 8, flat(Shade(RGB(110, 80, 50), 1)), flat(RGB(120, 90, 56)))
	s.Box(x-4, y-2.5, x+4, y+2.5, 8, 12, flat(iron), flat(Shade(iron, 1.4)))
	t := s.P(x+4, y, 11)
	s.c.Tri(t.Add(V{0, -1}), V{6, 3}, V{0, 3}, solid(Shade(iron, 0.9)))
}

func Blacksmith(team color.RGBA) *Canvas {
	s := newIso(3, 80)
	const S = 96.0
	s.plinth(S, ground(RGB(140, 120, 94)))
	base := stone(darkStone)
	up := timber(plaster, RGB(92, 62, 36), 52, 14)
	s.Box(44, 14, 54, 24, 3, 66, stone(RGB(120, 112, 104)), flat(RGB(40, 36, 34)))
	s.WallY(8, 60, 60, 3, 16, base)
	s.WallX(60, 8, 60, 3, 16, base)
	s.WallY(8, 60, 60, 16, 30, up)
	s.WallX(60, 8, 60, 16, 30, up)
	s.door(16, 28, 60, 3, 20)
	s.window(38, 60, 18, 26, false)
	s.window(60, 22, 18, 26, true)
	s.BannerY(46, 60, 6, 22, team)
	s.GableX(4, 4, 64, 64, 30, 50, teamRoof(team), up)
	s.Box(44, 14, 54, 24, 50, 66, stone(RGB(120, 112, 104)), flat(RGB(40, 36, 34)))
	s.smoke(s.P(49, 19, 70), 4)
	s.Box(68, 18, 88, 38, 3, 12, stone(RGB(130, 120, 110)), func(u, v, light float64) color.RGBA {
		d := math.Hypot(u-10, v-10) / 10
		return Mix(RGB(255, 236, 150), RGB(200, 60, 20), math.Min(1, d))
	})
	s.glow(s.P(78, 28, 14), 16, 9)
	s.Box(80, 20, 86, 26, 12, 40, stone(RGB(120, 112, 104)), flat(RGB(40, 36, 34)))
	s.smoke(s.P(83, 23, 44), 3)
	post := logs(darkWood)
	for _, p := range []V{{66, 16}, {66, 42}, {92, 42}} {
		s.Box(p.X-1.5, p.Y-1.5, p.X+1.5, p.Y+1.5, 3, 24, post, post)
	}
	s.LeanTo(64, 14, 94, 46, 30, 22, teamRoof(Mix(team, RGB(110, 70, 45), 0.3)))
	s.anvil(72, 70)
	s.Box(30, 72, 46, 80, 3, 8, planks(RGB(120, 84, 50)), flat(RGB(70, 110, 150)))
	for i, x := range []float64{84, 88} {
		b := s.P(x, 74, 3)
		s.c.Line(b, b.Add(V{float64(i), -14}), 1.4, haftCol)
		s.c.Line(b.Add(V{float64(i) - 2, -14}), b.Add(V{float64(i) + 2, -14}), 2.6, RGB(80, 82, 90))
	}
	s.Flag(s.P(92, 92, 3), 36, team)
	finish(s.c)
	return s.c
}

func SiegeWorkshop(team color.RGBA) *Canvas {
	s := newIso(3, 80)
	const S = 96.0
	s.plinth(S, ground(RGB(146, 120, 82)))
	back := vplanks(RGB(130, 92, 56))
	s.WallY(6, 90, 10, 3, 36, back)
	s.WallX(10, 6, 56, 3, 36, back)
	s.logPile(16, 16, 3, false)
	post := logs(darkWood)
	for _, p := range []V{{8, 8}, {50, 8}, {90, 8}, {8, 56}} {
		s.Box(p.X-2, p.Y-2, p.X+2, p.Y+2, 3, 38, post, post)
	}
	s.GableX(2, 2, 96, 62, 38, 62, teamRoof(team), back)
	for _, p := range []V{{50, 56}, {90, 56}} {
		s.Box(p.X-2, p.Y-2, p.X+2, p.Y+2, 3, 40, post, post)
	}
	beam := RGB(160, 112, 62)
	b0, b1 := s.P(30, 74, 3), s.P(62, 74, 3)
	top := s.P(46, 74, 34)
	s.c.Line(b0, top, 3, beam)
	s.c.Line(b1, top, 3, beam)
	s.c.Line(s.P(36, 74, 14), s.P(56, 74, 14), 2.5, beam)
	s.c.Line(s.P(30, 84, 3), s.P(62, 84, 3), 3, Shade(beam, 0.85))
	s.c.Line(b0, s.P(30, 84, 3), 3, Shade(beam, 0.9))
	s.c.Line(b1, s.P(62, 84, 3), 3, Shade(beam, 0.9))
	s.c.Line(top, s.P(26, 80, 20), 2.5, Shade(beam, 1.1))
	for _, p := range []V{s.P(30, 86, 6), s.P(62, 86, 6)} {
		s.c.Ellipse(p, 5, 5, func(nx, ny float64) color.RGBA {
			if r := nx*nx + ny*ny; r > 0.6 || r < 0.08 {
				return Shade(darkWood, Lit(nx, ny))
			}
			if math.Abs(nx) < 0.15 || math.Abs(ny) < 0.15 {
				return Shade(darkWood, 1.1)
			}
			return color.RGBA{}
		})
	}
	s.logPile(68, 70, 3, true)
	s.Flag(s.P(92, 92, 3), 40, team)
	finish(s.c)
	return s.c
}

func Monastery(team color.RGBA) *Canvas {
	s := newIso(3, 100)
	const S = 96.0
	s.plinth(S, stone(RGB(166, 158, 140)))
	wall := stone(lightStone)
	s.Box(10, 10, 30, 30, 3, 74, wall, wall)
	s.WallY(14, 26, 30.2, 54, 66, flat(RGB(36, 32, 30)))
	s.WallX(30.2, 14, 26, 54, 66, flat(RGB(36, 32, 30)))
	s.c.Ellipse(s.P(22, 30, 59), 3.2, 3.8, lit(goldCol))
	s.Pyramid(8, 8, 32, 32, 74, 96, teamRoof(team))
	cr := s.P(20, 20, 96)
	s.c.Line(cr, cr.Add(V{0, -9}), 1.6, goldCol)
	s.c.Line(cr.Add(V{-3, -6}), cr.Add(V{3, -6}), 1.6, goldCol)
	s.WallY(8, 76, 74, 3, 32, wall)
	s.WallX(76, 36, 74, 3, 32, wall)
	arch := func(u, v, light float64) color.RGBA {
		if v > 9 {
			return Shade(RGB(230, 200, 90), light)
		}
		return Shade(Mix(RGB(40, 50, 90), team, 0.4), light)
	}
	for _, x := range []float64{14, 30, 54, 66} {
		s.WallY(x, x+5, 74.2, 12, 26, arch)
	}
	for _, y := range []float64{44, 60} {
		s.WallX(76.2, y, y+5, 12, 26, arch)
	}
	s.door(38, 48, 74, 3, 22)
	s.BannerY(22, 74, 8, 28, team)
	s.GableX(4, 32, 80, 78, 32, 56, teamRoof(Mix(team, RGB(150, 140, 130), 0.25)), wall)
	s.c.Ellipse(s.P(76, 55, 40), 4, 5, func(nx, ny float64) color.RGBA {
		if nx*nx+ny*ny > 0.7 {
			return RGB(130, 120, 110)
		}
		if int((nx+1)*3)%2 == int((ny+1)*3)%2 {
			return Shade(team, 1.1)
		}
		return RGB(240, 200, 80)
	})
	s.c.Line(s.P(4, 55, 56), s.P(78, 55, 56), 1.4, goldCol)
	for _, p := range []V{{86, 84}, {60, 88}} {
		b := s.P(p.X, p.Y, 3)
		s.c.Ellipse(b.Add(V{0, -5}), 5, 5.5, lit(RGB(58, 120, 50)))
	}
	s.Flag(s.P(92, 74, 3), 36, team)
	finish(s.c)
	return s.c
}

// Wall connection bits for Palisade and StoneWall.
const (
	WallN = 1 << iota
	WallE
	WallS
	WallW
)

// wallTile draws a centre post plus sections towards each connected side.
func wallTile(mask int, half, h, ph float64, seg func(s iso, x0, y0, x1, y1, h float64), post func(s iso, x0, y0, x1, y1, h float64)) *Canvas {
	s := newIso(1, 52)
	const c = 16.0
	pr := half + 1
	if mask&WallN != 0 {
		seg(s, c-half, 0, c+half, c-pr, h)
	}
	if mask&WallW != 0 {
		seg(s, 0, c-half, c-pr, c+half, h)
	}
	post(s, c-pr, c-pr, c+pr, c+pr, ph)
	if mask&WallE != 0 {
		seg(s, c+pr, c-half, 32, c+half, h)
	}
	if mask&WallS != 0 {
		seg(s, c-half, c+pr, c+half, 32, h)
	}
	finish(s.c)
	return s.c
}

func Palisade(mask int) *Canvas {
	stake := func(s iso, x0, y0, x1, y1, h float64) {
		wood := RGB(140, 98, 58)
		alongX := x1-x0 > y1-y0
		n := int(math.Max(x1-x0, y1-y0) / 3.2)
		for i := 0; i <= n; i++ {
			t := (float64(i) + 0.5) / float64(n+1)
			x, y := (x0+x1)/2, (y0+y1)/2
			if alongX {
				x = x0 + (x1-x0)*t
			} else {
				y = y0 + (y1-y0)*t
			}
			hh := h + 2*Hash(int(x), int(y), 95)
			s.Box(x-1.6, y-1.6, x+1.6, y+1.6, 0, hh, vplanks(wood), logs(wood))
			s.Pyramid(x-1.6, y-1.6, x+1.6, y+1.6, hh, hh+4, logs(Shade(wood, 1.1)))
		}
	}
	post := func(s iso, x0, y0, x1, y1, h float64) {
		wood := RGB(120, 84, 50)
		s.Box(x0, y0, x1, y1, 0, h, logs(wood), logs(wood))
		s.Pyramid(x0, y0, x1, y1, h, h+5, logs(Shade(wood, 1.1)))
	}
	return wallTile(mask, 1.8, 20, 24, stake, post)
}

func StoneWall(mask int) *Canvas {
	wall := stone(wallStone)
	seg := func(s iso, x0, y0, x1, y1, h float64) {
		s.Box(x0, y0, x1, y1, 0, h, wall, stone(darkStone))
		if x1-x0 > y1-y0 {
			for x := x0 + 0.5; x < x1-2; x += 5 {
				s.Box(x, y0, x+2.6, y1, h, h+4, wall, wall)
			}
		} else {
			for y := y0 + 0.5; y < y1-2; y += 5 {
				s.Box(x0, y, x1, y+2.6, h, h+4, wall, wall)
			}
		}
	}
	post := func(s iso, x0, y0, x1, y1, h float64) {
		s.Box(x0, y0, x1, y1, 0, h, stone(lightStone), stone(darkStone))
		s.Merlons(x0-0.5, y0-0.5, x1+0.5, y1+0.5, h, stone(lightStone), true)
		s.Merlons(x0-0.5, y0-0.5, x1+0.5, y1+0.5, h, stone(lightStone), false)
	}
	return wallTile(mask, 3.5, 28, 34, seg, post)
}
