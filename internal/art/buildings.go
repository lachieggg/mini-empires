package art

import (
	"image/color"
	"math"
	"sort"
)

// Ages: 0 Dark, 1 Feudal, 2 Castle, 3 Imperial.
const NumAges = 4

var (
	wallStone  = RGB(176, 168, 150)
	darkStone  = RGB(150, 142, 128)
	lightStone = RGB(194, 186, 168)
	wood       = RGB(135, 95, 58)
	darkWood   = RGB(95, 65, 38)
	straw      = RGB(196, 160, 86)
	plaster    = RGB(226, 212, 178)
	dirtCol    = RGB(150, 122, 84)
)

func teamPat(team color.RGBA) pattern {
	return func(u, v, light float64) color.RGBA { return Shade(team, light) }
}

// stakes draws a palisade of pointed logs between two ground points,
// skipping the gap [g0, g1] along the run.
func (s iso) stakes(x0, y0, x1, y1, h, g0, g1 float64) {
	n := int(math.Hypot(x1-x0, y1-y0) / 4)
	lp := logs(RGB(128, 90, 54))
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := x0+(x1-x0)*t, y0+(y1-y0)*t
		if d := t * math.Hypot(x1-x0, y1-y0); d > g0 && d < g1 {
			continue
		}
		hh := h + 2*Hash(i, int(x0+y0), 81)
		s.Box(x-1.8, y-1.8, x+1.8, y+1.8, 3, 3+hh, vplanks(RGB(128, 90, 54)), lp)
		s.Pyramid(x-1.8, y-1.8, x+1.8, y+1.8, 3+hh, 7+hh, lp)
	}
}

func (s iso) campfire(x, y float64) {
	b := s.P(x, y, 3)
	for i := range 5 {
		a := float64(i) * math.Pi * 2 / 5
		s.c.Ellipse(b.Add(V{4 * math.Cos(a), 2 * math.Sin(a)}), 1.6, 1.2, lit(RGB(130, 130, 125)))
	}
	s.c.Ellipse(b.Add(V{0, -2}), 2.6, 3.5, solid(RGB(240, 140, 40)))
	s.c.Ellipse(b.Add(V{0, -1.5}), 1.3, 2, solid(RGB(255, 225, 110)))
}

// TownCenter evolves with the age: timber hall, timber on stone, stone
// keep, then a fortified keep with corner towers.
func TownCenter(team color.RGBA, age int) *Canvas {
	switch age {
	case 0:
		return tcDark(team)
	case 1:
		return tcFeudal(team)
	case 2:
		return tcCastle(team, false)
	default:
		return tcCastle(team, true)
	}
}

func tcDark(team color.RGBA) *Canvas {
	s := newIso(3, 76)
	const S = 96.0
	s.plinth(S, ground(dirtCol))
	s.stakes(4, 4, 92, 4, 14, -1, -1)
	s.stakes(4, 8, 4, 92, 14, -1, -1)
	s.campfire(18, 20)
	wall := logs(RGB(120, 84, 50))
	s.Box(24, 24, 72, 72, 3, 26, wall, wall)
	s.door(42, 54, 72, 3, 19)
	s.BannerY(28, 72, 10, 24, team)
	s.BannerY(60, 72, 10, 24, team)
	s.BannerX(72, 44, 10, 24, team)
	s.window(72, 30, 13, 19, true)
	s.window(72, 58, 13, 19, true)
	s.GableX(18, 18, 78, 78, 26, 60, thatch(straw), logs(RGB(120, 84, 50)))
	s.c.Line(s.P(18, 48, 60), s.P(74, 48, 60), 2.5, Shade(team, 0.9))
	s.Flag(s.P(20, 48, 60), 18, team)
	s.stakes(92, 4, 92, 92, 14, -1, -1)
	s.stakes(4, 92, 88, 92, 14, 36, 54)
	for _, x := range []float64{38, 58} {
		s.Box(x-2, 90, x+2, 94, 3, 26, logs(darkWood), logs(darkWood))
		s.WallY(x-2, x+2, 94.2, 14, 24, teamPat(team))
	}
	finish(s.c)
	return s.c
}

func tcFeudal(team color.RGBA) *Canvas {
	s := newIso(3, 88)
	const S = 96.0
	s.plinth(S, ground(RGB(150, 128, 92)))
	base := stone(darkStone)
	s.Box(14, 14, 82, 82, 3, 20, base, base)
	up := timber(plaster, RGB(92, 62, 36), 68, 24)
	s.WallY(16, 80, 80, 20, 44, up)
	s.WallX(80, 16, 80, 20, 44, up)
	s.c.Line(s.P(14, 82, 20), s.P(82, 82, 20), 1.5, RGB(92, 62, 36))
	s.c.Line(s.P(82, 82, 20), s.P(82, 14, 20), 1.5, RGB(92, 62, 36))
	for _, x := range []float64{24, 38, 56, 68} {
		s.window(x, 80, 28, 37, false)
	}
	for _, y := range []float64{24, 38, 56, 68} {
		s.window(80, y, 28, 37, true)
	}
	s.door(40, 56, 82, 3, 18)
	s.BannerY(22, 82, 5, 18, team)
	s.BannerY(66, 82, 5, 18, team)
	s.BannerX(82, 24, 5, 18, team)
	s.BannerX(82, 62, 5, 18, team)
	s.Box(60, 24, 70, 34, 62, 84, stone(darkStone), stone(darkStone))
	s.GableX(10, 10, 86, 86, 44, 80, teamRoof(team), up)
	s.Box(60, 24, 70, 34, 72, 86, stone(darkStone), stone(darkStone))
	s.c.Line(s.P(10, 48, 80), s.P(82, 48, 80), 1.5, Shade(team, 0.6))
	s.Flag(s.P(12, 48, 80), 20, team)
	finish(s.c)
	return s.c
}

func tcCastle(team color.RGBA, imperial bool) *Canvas {
	top := 92.0
	if imperial {
		top = 108
	}
	s := newIso(3, top)
	const S = 96.0
	s.plinth(S, stone(RGB(170, 162, 145)))
	wall := stone(wallStone)
	if imperial {
		wall = stone(lightStone)
	}
	s.Box(8, 8, 88, 88, 3, 34, wall, stone(darkStone))
	s.Merlons(8, 8, 88, 88, 34, wall, true)
	tower := func(x, y float64) {
		s.Cylinder(x, y, 9, 3, 50, wall, wall)
		s.RoundMerlons(x, y, 9, 50, wall)
		s.Cone(x, y, 10.5, 54, 76, teamRoof(team))
		p := s.P(x, y, 76)
		s.c.Line(p, p.Add(V{0, -5}), 1.5, goldCol)
		s.c.Ellipse(p.Add(V{0, -5}), 1.6, 1.6, lit(goldCol))
	}
	if imperial {
		tower(10, 10)
	}
	kz := 74.0
	if imperial {
		kz = 82
	}
	s.Box(30, 30, 66, 66, 34, kz, stone(RGB(190, 182, 165)), wall)
	s.window(44, 66, kz-22, kz-12, false)
	s.window(66, 44, kz-22, kz-12, true)
	s.Pyramid(26, 26, 70, 70, kz, kz+28, teamRoof(team))
	if imperial {
		s.PyramidTrim(26, 26, 70, 70, kz, kz+28, goldCol)
		s.c.Line(s.P(26, 70, kz), s.P(70, 70, kz), 2, goldCol)
		s.c.Line(s.P(70, 70, kz), s.P(70, 26, kz), 2, goldCol)
		s.BannerY(36, 66, kz-30, kz-4, team)
		s.BannerX(66, 50, kz-30, kz-4, team)
	}
	s.Flag(s.P(48, 48, kz+28), 20, team)
	if imperial {
		tower(86, 10)
		tower(10, 86)
	}
	s.Merlons(8, 8, 88, 88, 34, wall, false)
	s.door(38, 58, 88, 3, 24)
	s.BannerY(18, 88, 12, 30, team)
	s.BannerY(66, 88, 12, 30, team)
	s.BannerX(88, 44, 12, 30, team)
	s.window(88, 22, 16, 24, true)
	s.window(88, 68, 16, 24, true)
	if imperial {
		s.c.Line(s.P(8, 88, 33), s.P(88, 88, 33), 1.5, goldCol)
		s.c.Line(s.P(88, 88, 33), s.P(88, 8, 33), 1.5, goldCol)
		tower(86, 86)
	}
	finish(s.c)
	return s.c
}

// House evolves with the age: thatch hut, timber house, stone house,
// grand stone house.
func House(team color.RGBA, age int) *Canvas {
	switch age {
	case 0:
		return hut(team)
	case 1:
		return timberHouse(team)
	case 2:
		return stoneHouse(team, false)
	default:
		return stoneHouse(team, true)
	}
}

func hut(team color.RGBA) *Canvas {
	s := newIso(2, 50)
	s.plinth(64, ground(dirtCol))
	wattle := func(u, v, light float64) color.RGBA {
		f := 0.85 + 0.2*math.Sin(u*1.6+math.Floor(v/3)*1.5)
		return Shade(RGB(150, 112, 70), light*f)
	}
	s.Cylinder(32, 32, 19, 3, 20, wattle, nil)
	a := math.Pi / 4
	p := s.P(32+19*math.Cos(a-0.25), 32+19*math.Sin(a-0.25), 3)
	q := s.P(32+19*math.Cos(a+0.25), 32+19*math.Sin(a+0.25), 3)
	s.c.Quad(p, q.Sub(p), V{0, -14}, func(u, v float64) color.RGBA {
		if v > 0.85 {
			return Shade(team, 0.7)
		}
		return Shade(team, 0.85+0.15*math.Sin(u*12))
	})
	s.Cone(32, 32, 25, 18, 48, func(u, v, light float64) color.RGBA {
		if v < 4 {
			return Shade(team, light*0.95)
		}
		return thatch(straw)(u, v, light)
	})
	s.Flag(s.P(56, 50, 3), 26, team)
	finish(s.c)
	return s.c
}

func timberHouse(team color.RGBA) *Canvas {
	s := newIso(2, 56)
	const S = 64.0
	s.plinth(S, ground(RGB(140, 115, 80)))
	wall := timber(plaster, RGB(92, 62, 36), 44, 27)
	s.WallY(10, 54, 54, 3, 30, wall)
	s.WallX(54, 10, 54, 3, 30, wall)
	s.door(20, 31, 54, 3, 21)
	s.window(38, 54, 13, 21, false)
	s.window(54, 28, 13, 21, true)
	s.GableX(5, 5, 59, 59, 30, 52, teamRoof(Mix(team, RGB(150, 80, 50), 0.35)), wall)
	s.BannerY(42, 54, 22, 29, team)
	finish(s.c)
	return s.c
}

func stoneHouse(team color.RGBA, grand bool) *Canvas {
	top, h := 58.0, 28.0
	if grand {
		top, h = 72, 40
	}
	s := newIso(2, top)
	const S = 64.0
	s.plinth(S, stone(RGB(160, 152, 136)))
	wall := stone(wallStone)
	if grand {
		wall = stone(lightStone)
	}
	s.Box(48, 18, 54, 24, h, h+22, stone(darkStone), stone(darkStone))
	s.WallY(10, 54, 54, 3, h, wall)
	s.WallX(54, 10, 54, 3, h, wall)
	s.door(18, 28, 54, 3, 18)
	s.window(36, 54, 10, 18, false)
	s.window(54, 20, 10, 18, true)
	s.window(54, 38, 10, 18, true)
	if grand {
		s.c.Line(s.P(10, 54, 22), s.P(54, 54, 22), 2, goldCol)
		s.c.Line(s.P(54, 54, 22), s.P(54, 10, 22), 2, goldCol)
		s.window(20, 54, 27, 35, false)
		s.window(38, 54, 27, 35, false)
		s.window(54, 20, 27, 35, true)
		s.window(54, 38, 27, 35, true)
		s.BannerY(46, 54, 12, 30, team)
		s.BannerX(54, 28, 18, 34, team)
	} else {
		s.BannerY(42, 54, 10, 24, team)
	}
	s.GableX(5, 5, 59, 59, h, h+22, teamRoof(team), wall)
	s.Box(14, 40, 20, 46, h+6, h+20, stone(darkStone), stone(RGB(60, 55, 50)))
	if grand {
		s.c.Line(s.P(5, 32, h+22), s.P(55, 32, h+22), 2, goldCol)
		s.c.Line(s.P(5, 59, h), s.P(59, 59, h), 1.5, goldCol)
		s.Flag(s.P(7, 32, h+22), 14, team)
	}
	finish(s.c)
	return s.c
}

func Mill(team color.RGBA) *Canvas {
	s := newIso(2, 96)
	s.plinth(64, ground(RGB(150, 128, 90)))
	for i, p := range []V{{8, 40}, {12, 46}, {8, 50}} {
		b := s.P(p.X, p.Y, 3)
		s.c.Ellipse(b.Add(V{0, -3}), 3.5, 3.5, lit(Shade(RGB(200, 180, 130), 1-0.05*float64(i))))
	}
	body := stone(RGB(204, 196, 176))
	s.Cylinder(30, 30, 16, 3, 46, body, nil)
	s.door(38, 44, 46, 3, 16)
	s.window(44, 30, 26, 32, true)
	s.Cone(30, 30, 19, 46, 66, teamRoof(team))
	hub := s.P(43, 43, 54)
	for i := range 4 {
		a := float64(i)*math.Pi/2 + math.Pi/5
		d := V{math.Cos(a), math.Sin(a)}
		dir := V{d.X, -d.Y}
		tip := hub.Add(dir.Mul(31))
		n := V{d.Y, d.X}
		s.c.Line(hub, tip, 1.6, darkWood)
		start := hub.Add(dir.Mul(7))
		s.c.Quad(start, dir.Mul(24), n.Mul(7), func(u, v float64) color.RGBA {
			if int(u*24)%4 == 0 || int(v*7)%3 == 0 {
				return RGB(96, 68, 42)
			}
			if u > 0.8 {
				return Shade(team, 1.05)
			}
			return Shade(RGB(222, 210, 180), 1-0.15*v)
		})
	}
	s.c.Ellipse(hub, 2.5, 2.5, lit(darkWood))
	s.Flag(s.P(30, 30, 66), 12, team)
	finish(s.c)
	return s.c
}

// shed is an open-sided lean-to on four posts.
func (s iso) shed(x0, y0, x1, y1, h float64, team color.RGBA, back pattern) {
	post := logs(darkWood)
	if back != nil {
		s.WallY(x0, x1, y0+2, 3, h, back)
		s.WallX(x0+2, y0, y1, 3, h, back)
	}
	for _, p := range []V{{x0, y0}, {x1, y0}, {x0, y1}} {
		s.Box(p.X-1.5, p.Y-1.5, p.X+1.5, p.Y+1.5, 3, h, post, post)
	}
}

func (s iso) shedRoof(x0, y0, x1, y1, h float64, team color.RGBA) {
	post := logs(darkWood)
	s.Box(x1-1.5, y1-1.5, x1+1.5, y1+1.5, 3, h-4, post, post)
	s.LeanTo(x0-3, y0-3, x1+3, y1+4, h+6, h-4, teamRoof(team))
}

func (s iso) logPile(x, y float64, n int, along bool) {
	bark := RGB(120, 82, 48)
	for row := 0; row < n; row++ {
		for i := 0; i < n-row; i++ {
			off := float64(i)*5 + float64(row)*2.5
			z := 5.5 + float64(row)*4.2
			var a, b V
			if along {
				a, b = s.P(x, y+off, z), s.P(x+20, y+off, z)
			} else {
				a, b = s.P(x+off, y, z), s.P(x+off, y+20, z)
			}
			s.c.Line(a, b, 4.6, Shade(bark, 0.85+0.1*float64(row)))
			s.c.Line(a.Add(V{0, -1.5}), b.Add(V{0, -1.5}), 1.2, Shade(bark, 1.15))
			s.c.Ellipse(b, 2.3, 2.3, func(nx, ny float64) color.RGBA {
				if nx*nx+ny*ny < 0.25 {
					return RGB(170, 125, 75)
				}
				return RGB(214, 176, 120)
			})
		}
	}
}

func LumberCamp(team color.RGBA) *Canvas {
	s := newIso(2, 48)
	s.plinth(64, ground(RGB(146, 120, 82)))
	s.shed(6, 6, 40, 36, 24, team, nil)
	s.logPile(12, 10, 3, false)
	s.shedRoof(6, 6, 40, 36, 24, team)
	s.logPile(36, 44, 2, true)
	b := s.P(50, 22, 3)
	for _, dx := range []float64{-5, 5} {
		s.c.Line(b.Add(V{dx - 2, 0}), b.Add(V{dx + 2, -9}), 1.5, darkWood)
		s.c.Line(b.Add(V{dx + 2, 0}), b.Add(V{dx - 2, -9}), 1.5, darkWood)
	}
	s.c.Line(b.Add(V{-9, -9}), b.Add(V{9, -9}), 4, RGB(120, 82, 48))
	s.c.Quad(b.Add(V{-1, -17}), V{3, 0}, V{1, 9}, solid(RGB(200, 204, 212)))
	s.c.Line(b.Add(V{1, -18}), b.Add(V{3, -21}), 1.5, darkWood)
	st := s.P(22, 52, 3)
	s.c.Ellipse(st.Add(V{0, -2}), 4.5, 2.5, lit(RGB(120, 82, 48)))
	s.c.Ellipse(st.Add(V{0, -4}), 4.5, 2.2, solid(RGB(206, 168, 112)))
	s.c.Line(st.Add(V{0, -4}), st.Add(V{4, -12}), 1.5, darkWood)
	s.c.Quad(st.Add(V{2, -14}), V{4, 1}, V{-1, 3}, solid(steel))
	s.Flag(s.P(58, 58, 3), 28, team)
	finish(s.c)
	return s.c
}

func MiningCamp(team color.RGBA) *Canvas {
	s := newIso(2, 48)
	s.plinth(64, ground(RGB(150, 130, 100)))
	s.shed(6, 6, 40, 36, 24, team, stone(darkStone))
	for i, p := range []blob{{14, 16, 3}, {20, 14, 3.5}, {26, 18, 3}, {18, 22, 3}} {
		b := s.P(p.x, p.y, 3)
		col := RGB(140, 135, 128)
		if i%2 == 1 {
			col = RGB(220, 180, 60)
		}
		s.c.Ellipse(b.Add(V{0, -2}), p.r, p.r*0.8, lit(col))
	}
	s.shedRoof(6, 6, 40, 36, 24, team)
	for _, x := range []float64{12, 18} {
		b := s.P(x, 40, 3)
		s.c.Line(b, b.Add(V{3, -14}), 1.5, RGB(150, 110, 65))
		s.c.Line(b.Add(V{-1, -13}), b.Add(V{7, -15}), 2, steel)
	}
	planksP := planks(RGB(120, 84, 50))
	s.Box(38, 40, 54, 50, 6, 14, planksP, flat(RGB(60, 45, 30)))
	for i, p := range []blob{{42, 43, 2.6}, {46, 45, 3}, {50, 43, 2.6}, {44, 47, 2.4}, {49, 47, 2.4}} {
		col := RGB(225, 185, 55)
		if i%2 == 1 {
			col = RGB(170, 170, 168)
		}
		s.c.Ellipse(s.P(p.x, p.y, 15), p.r, p.r*0.8, lit(col))
	}
	for _, x := range []float64{41, 51} {
		s.c.Ellipse(s.P(x, 50, 5), 3, 3, func(nx, ny float64) color.RGBA {
			if nx*nx+ny*ny < 0.2 {
				return RGB(80, 80, 85)
			}
			return Shade(darkWood, Lit(nx, ny))
		})
	}
	for _, p := range []blob{{14, 54, 4}, {22, 56, 3.5}, {18, 50, 3}} {
		s.c.Ellipse(s.P(p.x, p.y, 3).Add(V{0, -2}), p.r, p.r*0.75, lit(RGB(130, 125, 118)))
	}
	s.Flag(s.P(58, 26, 3), 28, team)
	finish(s.c)
	return s.c
}

func Barracks(team color.RGBA) *Canvas {
	s := newIso(3, 60)
	const S = 96.0
	s.plinth(S, ground(RGB(150, 125, 88)))
	base := stone(darkStone)
	woodP := planks(wood)
	s.WallY(8, 88, 70, 3, 11, base)
	s.WallX(88, 22, 70, 3, 11, base)
	s.WallY(8, 88, 70, 11, 30, woodP)
	s.WallX(88, 22, 70, 11, 30, woodP)
	s.WallY(40, 56, 70, 3, 23, planks(RGB(80, 52, 30)))
	s.GableX(4, 18, 92, 74, 30, 54, tiles(Mix(team, RGB(110, 60, 40), 0.45)), woodP)
	s.WallY(20, 30, 70, 14, 28, teamPat(team))
	s.WallY(66, 76, 70, 14, 28, teamPat(team))
	pole := darkWood
	for _, x := range []float64{16, 24, 32} {
		b := s.P(x, 84, 3)
		s.c.Line(b, b.Add(V{-3, -20}), 1.5, RGB(110, 80, 50))
		s.c.Line(b.Add(V{-3, -20}), b.Add(V{-4, -24}), 2, RGB(200, 200, 210))
	}
	r0, r1 := s.P(12, 86, 12), s.P(36, 86, 12)
	s.c.Line(r0, r1, 2, pole)
	d := s.P(70, 86, 3)
	s.c.Line(d, d.Add(V{0, -22}), 2.5, pole)
	s.c.Line(d.Add(V{-7, -15}), d.Add(V{7, -15}), 2.5, pole)
	s.c.Ellipse(d.Add(V{0, -23}), 3.5, 3.5, lit(RGB(200, 180, 120)))
	s.Flag(s.P(90, 90, 3), 46, team)
	finish(s.c)
	return s.c
}

func ArcheryRange(team color.RGBA) *Canvas {
	s := newIso(3, 60)
	const S = 96.0
	s.plinth(S, ground(RGB(152, 132, 90)))
	wall := timber(plaster, RGB(92, 62, 36), 80, 24)
	s.WallY(8, 88, 44, 3, 26, wall)
	s.WallX(88, 10, 44, 3, 26, wall)
	s.door(20, 32, 44, 3, 18)
	s.window(44, 44, 12, 20, false)
	s.window(60, 44, 12, 20, false)
	s.window(88, 22, 12, 20, true)
	s.BannerY(72, 44, 8, 24, team)
	s.GableX(4, 6, 92, 48, 26, 48, teamRoof(team), wall)
	for _, p := range []V{{60, 60}, {82, 60}} {
		b := s.P(p.X, p.Y, 3)
		s.c.Ellipse(b.Add(V{0, -3}), 6, 3.5, lit(straw))
	}
	type tgt struct{ x, y float64 }
	for _, t := range []tgt{{22, 80}, {46, 82}, {70, 80}} {
		b := s.P(t.x, t.y, 3)
		s.c.Line(b.Add(V{-4, 0}), b.Add(V{-2, -16}), 1.6, darkWood)
		s.c.Line(b.Add(V{4, 0}), b.Add(V{2, -16}), 1.6, darkWood)
		s.c.Line(b.Add(V{0, 2}), b.Add(V{0, -12}), 1.6, darkWood)
		ctr := b.Add(V{0, -17})
		s.c.Ellipse(ctr, 7, 8, func(nx, ny float64) color.RGBA {
			r := math.Sqrt(nx*nx + ny*ny)
			l := 1.05 - 0.15*nx - 0.1*ny
			switch {
			case r > 0.8:
				return Shade(straw, l)
			case r > 0.6:
				return Shade(RGB(240, 236, 224), l)
			case r > 0.38:
				return Shade(team, l)
			case r > 0.2:
				return Shade(RGB(240, 236, 224), l)
			}
			return Shade(RGB(225, 180, 40), l)
		})
		for i := range 2 {
			a := ctr.Add(V{-2 + 3*float64(i), -1 + 2*float64(i)})
			s.c.Line(a, a.Add(V{4, -2}), 1, RGB(120, 90, 60))
			s.c.Blend(int(a.X+4), int(a.Y-2), RGB(240, 240, 240))
		}
	}
	s.Flag(s.P(92, 92, 3), 40, team)
	finish(s.c)
	return s.c
}

func (s iso) fence(pts []V, z float64) {
	post := RGB(120, 86, 52)
	for i := 0; i+1 < len(pts); i++ {
		a, b := pts[i], pts[i+1]
		n := int(math.Max(1, math.Hypot(b.X-a.X, b.Y-a.Y)/10))
		for j := 0; j < n; j++ {
			q0 := a.Add(b.Sub(a).Mul(float64(j) / float64(n)))
			q1 := a.Add(b.Sub(a).Mul(float64(j+1) / float64(n)))
			for _, h := range []float64{5, 10} {
				s.c.Line(s.P(q0.X, q0.Y, z+h), s.P(q1.X, q1.Y, z+h), 1.6, Shade(post, 1.1))
			}
			pp := s.P(q0.X, q0.Y, z)
			s.c.Line(pp, pp.Add(V{0, -12}), 2, post)
		}
	}
	l := pts[len(pts)-1]
	pp := s.P(l.X, l.Y, z)
	s.c.Line(pp, pp.Add(V{0, -12}), 2, post)
}

func Stable(team color.RGBA) *Canvas {
	s := newIso(3, 72)
	const S = 96.0
	s.plinth(S, ground(RGB(146, 126, 86)))
	barn := vplanks(RGB(150, 72, 48))
	s.fence([]V{{60, 52}, {92, 52}}, 3)
	s.WallY(6, 56, 92, 3, 30, barn)
	s.WallX(56, 6, 92, 3, 30, barn)
	s.WallY(16, 34, 92.2, 3, 24, func(u, v, light float64) color.RGBA {
		if u < 1.2 || u > 16.8 || v > 19.8 || v < 1.2 || math.Abs(u/18-v/21) < 0.06 || math.Abs(u/18+v/21-1) < 0.06 || math.Abs(u-9) < 0.6 {
			return Shade(RGB(236, 226, 206), light)
		}
		return Shade(RGB(124, 60, 40), light*(0.9+0.15*Hash(int(u/3), 0, 91)))
	})
	s.window(56, 20, 14, 22, true)
	s.window(56, 70, 14, 22, true)
	s.BannerY(42, 92, 8, 24, team)
	s.GableY(2, 2, 60, 96, 30, 60, teamRoof(Mix(team, RGB(120, 70, 45), 0.3)), barn)
	s.c.Ellipse(s.P(30, 92, 48), 4, 4, solid(RGB(40, 30, 25)))
	s.Box(64, 60, 70, 76, 3, 8, planks(RGB(120, 84, 50)), flat(RGB(70, 110, 150)))
	s.c.Ellipse(s.P(80, 66, 3).Add(V{0, -4}), 6, 4.5, lit(straw))
	s.fence([]V{{92, 52}, {92, 92}, {60, 92}}, 3)
	s.Flag(s.P(60, 52, 3), 40, team)
	finish(s.c)
	return s.c
}

func Tower(team color.RGBA) *Canvas {
	s := newIso(1, 112)
	const S = 32.0
	s.plinth(S, stone(RGB(160, 152, 136)))
	wall := stone(wallStone)
	s.Box(6, 6, 26, 26, 3, 78, wall, wall)
	s.slit(15, 26, 30, 40, false)
	s.slit(26, 15, 30, 40, true)
	s.slit(15, 26, 56, 66, false)
	s.slit(26, 15, 56, 66, true)
	s.door(12, 20, 26, 3, 15)
	s.BannerY(8, 26, 42, 54, team)
	s.Box(4, 4, 28, 28, 78, 84, stone(lightStone), stone(darkStone))
	s.Merlons(4, 4, 28, 28, 84, stone(lightStone), true)
	s.Pyramid(7, 7, 25, 25, 84, 104, teamRoof(team))
	s.Merlons(4, 4, 28, 28, 84, stone(lightStone), false)
	s.Flag(s.P(16, 16, 104), 14, team)
	finish(s.c)
	return s.c
}

func Castle(team color.RGBA) *Canvas {
	s := newIso(4, 112)
	const S = 128.0
	s.plinth(S, stone(RGB(160, 152, 136)))
	wall := stone(wallStone)
	topP := stone(darkStone)
	tower := func(x, y float64) {
		s.Cylinder(x, y, 13, 3, 58, wall, topP)
		s.RoundMerlons(x, y, 13, 58, wall)
		s.slit(x+5, y+13, 30, 40, false)
		s.Cone(x, y, 11, 58, 82, teamRoof(team))
		s.Flag(s.P(x, y, 82), 12, team)
	}
	curtain := func(x0, y0, x1, y1 float64) {
		s.Box(x0, y0, x1, y1, 3, 40, wall, topP)
		s.Merlons(x0, y0, x1, y1, 40, wall, true)
		s.Merlons(x0, y0, x1, y1, 40, wall, false)
	}
	tower(14, 14)
	curtain(14, 8, 114, 20)
	curtain(8, 14, 20, 114)
	keep := stone(RGB(190, 182, 165))
	s.Box(42, 42, 86, 86, 20, 86, keep, topP)
	s.Merlons(42, 42, 86, 86, 86, keep, true)
	s.Box(56, 56, 72, 72, 86, 96, keep, topP)
	s.Pyramid(54, 54, 74, 74, 96, 112, teamRoof(team))
	s.Flag(s.P(64, 64, 112), 16, team)
	s.Merlons(42, 42, 86, 86, 86, keep, false)
	s.window(56, 86, 60, 70, false)
	s.window(86, 56, 60, 70, true)
	s.BannerY(46, 86, 50, 78, team)
	s.BannerY(74, 86, 50, 78, team)
	s.BannerX(86, 46, 50, 78, team)
	s.BannerX(86, 74, 50, 78, team)
	tower(114, 14)
	tower(14, 114)
	curtain(108, 14, 120, 114)
	curtain(14, 108, 114, 120)
	s.BannerX(120, 40, 14, 34, team)
	s.BannerX(120, 80, 14, 34, team)
	s.BannerY(36, 120, 14, 34, team)
	s.Box(50, 104, 78, 124, 3, 50, keep, topP)
	s.Merlons(50, 104, 78, 124, 50, keep, true)
	s.Merlons(50, 104, 78, 124, 50, keep, false)
	s.WallY(56, 72, 124.2, 3, 26, func(u, v, light float64) color.RGBA {
		if int(u)%3 == 0 || int(v)%4 == 0 {
			return Shade(RGB(60, 55, 50), light)
		}
		return Shade(RGB(30, 26, 24), light)
	})
	s.BannerY(52, 124, 30, 46, team)
	s.BannerY(68, 124, 30, 46, team)
	tower(114, 114)
	finish(s.c)
	return s.c
}

// Foundation is a construction site for a building of the given size.
func Foundation(size int) *Canvas {
	s := newIso(size, 44)
	S := float64(size * 32)
	s.plinth(S, ground(dirtCol))
	woodC := RGB(118, 80, 44)
	in := math.Min(8, S/5)
	part := stone(wallStone)
	s.Box(in-2, in-2, S-in+2, in+2, 3, 3+math.Min(10, 3*float64(size)), part, stone(lightStone))
	s.Box(in-2, in+2, in+2, S-in+2, 3, 3+math.Min(7, 2*float64(size)), part, stone(lightStone))
	corners := []V{s.P(in, in, 3), s.P(S-in, in, 3), s.P(in, S-in, 3), s.P(S-in, S-in, 3)}
	h := math.Min(30, 14+8*float64(size))
	for _, p := range corners {
		s.c.Line(p, p.Add(V{0, -h}), 3, woodC)
	}
	s.c.Line(corners[0].Add(V{0, -h + 8}), corners[1].Add(V{0, -h + 8}), 2.5, woodC)
	s.c.Line(corners[0].Add(V{0, -h + 8}), corners[2].Add(V{0, -h + 8}), 2.5, woodC)
	s.c.Line(corners[2].Add(V{0, -h + 8}), corners[3].Add(V{0, -h + 8}), 2.5, woodC)
	s.c.Line(corners[1].Add(V{0, -h + 8}), corners[3].Add(V{0, -h + 8}), 2.5, woodC)
	w := math.Min(24, 8*float64(size))
	for i := range 3 {
		p := s.P(S/2, S/2, 3+float64(i)*3)
		s.c.Quad(p.Add(V{-w / 2, 0}), V{w, w / 4}, V{0, -3}, func(u, v float64) color.RGBA { return Shade(woodC, 0.8+0.2*v) })
	}
	stones := []V{{S * 0.25, S * 0.7}, {S * 0.32, S * 0.78}, {S * 0.7, S * 0.3}}
	sort.Slice(stones, func(i, j int) bool { return stones[i].X+stones[i].Y < stones[j].X+stones[j].Y })
	for _, p := range stones {
		s.Box(p.X-2.5, p.Y-2.5, p.X+2.5, p.Y+2.5, 3, 7, stone(wallStone), stone(lightStone))
	}
	finish(s.c)
	return s.c
}
