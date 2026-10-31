package art

import (
	"image/color"
	"math"
)

// Terrain tiles are 64x32 diamonds. Patterns are written in the tile
// lattice coordinates (u, v): moving one tile along either diamond edge adds
// 1 to u or v, so anything periodic in u and v with period 1 tiles without
// seams.

func inDiamond(x, y int) bool {
	return math.Abs(float64(x)-31.5)/32+math.Abs(float64(y)-15.5)/16 <= 1
}

func latticeUV(x, y int) (float64, float64) {
	px, py := float64(x)+0.5, float64(y)+0.5
	return px/64 + py/32, px/64 - py/32
}

func wrap(i, n int) int { return ((i % n) + n) % n }

// pHash is a hash on an n x n grid of lattice cells that wraps every tile.
func pHash(u, v float64, n int, seed uint32) float64 {
	return Hash(wrap(int(math.Floor(u*float64(n))), n), wrap(int(math.Floor(v*float64(n))), n), seed)
}

// pSmooth is value noise on an n x n lattice grid that wraps every tile.
func pSmooth(u, v float64, n int, seed uint32) float64 {
	x, y := u*float64(n), v*float64(n)
	ix, iy := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(ix), y-float64(iy)
	fx, fy = fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	h := func(i, j int) float64 { return Hash(wrap(i, n), wrap(j, n), seed) }
	a := h(ix, iy) + (h(ix+1, iy)-h(ix, iy))*fx
	b := h(ix, iy+1) + (h(ix+1, iy+1)-h(ix, iy+1))*fx
	return a + (b-a)*fy
}

func terrain(fn func(x, y int, u, v float64) color.RGBA) *Canvas {
	c := New(64, 32)
	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			if !inDiamond(x, y) {
				continue
			}
			u, v := latticeUV(x, y)
			c.Blend(x, y, fn(x, y, u, v))
		}
	}
	return c
}

func Grass(variant int) *Canvas {
	seed := uint32(variant*31 + 7)
	base := RGB(92, 142, 60)
	return terrain(func(x, y int, u, v float64) color.RGBA {
		f := 0.9 + 0.2*Smooth(float64(x)/7, float64(y)/3.5, seed)
		col := Shade(base, f)
		if h := Hash(x/2, y, seed); h < 0.05 {
			col = Shade(RGB(120, 175, 75), f)
		} else if h > 0.96 {
			col = Shade(base, f*0.8)
		}
		if variant == 3 && Hash(x, y, seed+9) < 0.01 {
			col = RGB(230, 220, 120)
		}
		return col
	})
}

// speckled is a base colour with soft tile-periodic mottling and a few
// lighter and darker specks.
func speckled(base, light, dark color.RGBA, seed uint32, amount float64) func(x, y int, u, v float64) color.RGBA {
	return func(x, y int, u, v float64) color.RGBA {
		f := 0.9 + 0.2*pSmooth(u, v, 4, seed)
		col := Shade(base, f)
		if h := pHash(u, v, 24, seed+1); h < amount {
			col = Shade(light, f)
		} else if h > 1-amount {
			col = Shade(dark, f)
		}
		return col
	}
}

func DryGrass(variant int) *Canvas {
	seed := uint32(variant*17 + 101)
	return terrain(speckled(RGB(160, 152, 72), RGB(196, 182, 98), RGB(120, 118, 58), seed, 0.06))
}

func Dirt(variant int) *Canvas {
	seed := uint32(variant*13 + 201)
	sp := speckled(RGB(136, 104, 68), RGB(165, 132, 92), RGB(104, 78, 50), seed, 0.05)
	return terrain(func(x, y int, u, v float64) color.RGBA {
		col := sp(x, y, u, v)
		if variant == 1 && pHash(u, v, 12, seed+5) < 0.06 {
			col = Shade(RGB(150, 140, 125), 0.9+0.2*pHash(u, v, 24, seed+6))
		}
		return col
	})
}

func Sand(variant int) *Canvas {
	seed := uint32(variant*11 + 301)
	sp := speckled(RGB(214, 192, 132), RGB(232, 214, 160), RGB(186, 164, 108), seed, 0.04)
	return terrain(func(x, y int, u, v float64) color.RGBA {
		col := sp(x, y, u, v)
		w := math.Sin(2*math.Pi*(3*u+float64(variant+1)*v) + 3*pSmooth(u, v, 3, seed+3))
		if w > 0.82 {
			col = Shade(col, 0.92)
		}
		return col
	})
}

// Water frames loop: frame 4 equals frame 0.
func Water(frame int) *Canvas {
	ph := float64(frame) * math.Pi / 2
	base := RGB(54, 112, 168)
	return terrain(func(x, y int, u, v float64) color.RGBA {
		depth := pSmooth(u, v, 3, 401)
		col := Mix(Shade(base, 1.05), RGB(40, 88, 150), depth*0.6)
		w1 := math.Sin(2*math.Pi*(2*u+v) + ph + 2*pSmooth(u, v, 2, 402))
		w2 := math.Sin(2*math.Pi*(u-3*v) - ph + 2*pSmooth(u, v, 2, 403))
		switch w := w1 + 0.7*w2; {
		case w > 1.45:
			col = Mix(col, RGB(200, 226, 240), 0.55)
		case w > 1.15:
			col = Mix(col, RGB(150, 196, 226), 0.35)
		case w < -1.4:
			col = Shade(col, 0.88)
		}
		return col
	})
}

func Forest(variant int) *Canvas {
	seed := uint32(variant*19 + 501)
	base := RGB(66, 102, 46)
	return terrain(func(x, y int, u, v float64) color.RGBA {
		f := 0.85 + 0.25*pSmooth(u, v, 4, seed)
		col := Shade(base, f)
		switch h := pHash(u, v, 20, seed+1); {
		case h < 0.07:
			col = Shade(RGB(120, 84, 42), f)
		case h < 0.1:
			col = Shade(RGB(160, 112, 40), f)
		case h > 0.95:
			col = Shade(RGB(44, 72, 34), f)
		case h > 0.92:
			col = Shade(RGB(96, 128, 58), f)
		}
		return col
	})
}
