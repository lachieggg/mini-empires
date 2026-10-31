package art

import (
	"image/color"
	"math"
)

// Ground atlas: four screen-space ground textures that the game samples at
// the screen position modulo GroundCell and blends per pixel.
const (
	GroundCell      = 256
	GroundAtlasSize = 2 * GroundCell
)

// Ground atlas cells as (col, row).
const (
	GroundGrass = iota
	GroundDryGrass
	GroundDirt
	GroundSand
)

// gnoise is value noise that wraps every GroundCell px, with cx cells across
// and cy cells down. cy = 2*cx gives features squashed 2:1 like the ground
// under the isometric camera.
func gnoise(x, y float64, cx, cy int, seed uint32) float64 {
	fx, fy := x/GroundCell*float64(cx), y/GroundCell*float64(cy)
	ix, iy := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := fx-float64(ix), fy-float64(iy)
	tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	h := func(i, j int) float64 { return Hash(wrap(i, cx), wrap(j, cy), seed) }
	a := h(ix, iy) + (h(ix+1, iy)-h(ix, iy))*tx
	b := h(ix, iy+1) + (h(ix+1, iy+1)-h(ix, iy+1))*tx
	return a + (b-a)*ty
}

// gfbm sums octaves of gnoise, returning roughly [0, 1].
func gfbm(x, y float64, cx int, oct int, seed uint32) float64 {
	sum, amp, norm := 0.0, 1.0, 0.0
	for o := range oct {
		sum += amp * gnoise(x, y, cx, 2*cx, seed+uint32(o)*7)
		norm += amp
		amp *= 0.5
		cx *= 2
	}
	return sum / norm
}

func smoothstep(a, b, x float64) float64 {
	t := math.Max(0, math.Min(1, (x-a)/(b-a)))
	return t * t * (3 - 2*t)
}

// groundCell is one texture cell that wraps when drawn on.
type groundCell struct{ c *Canvas }

func (g groundCell) blend(x, y float64, col color.RGBA) {
	g.c.Blend(wrap(int(math.Floor(x)), GroundCell), wrap(int(math.Floor(y)), GroundCell), col)
}

func (g groundCell) at(x, y int) color.RGBA {
	i := g.c.Img.PixOffset(wrap(x, GroundCell), wrap(y, GroundCell))
	p := g.c.Img.Pix[i : i+4]
	return color.RGBA{p[0], p[1], p[2], 255}
}

func (g groundCell) fill(fn func(x, y float64) color.RGBA) {
	for y := range GroundCell {
		for x := range GroundCell {
			g.c.Blend(x, y, fn(float64(x)+0.5, float64(y)+0.5))
		}
	}
}

// blades draws n short grass strokes, tinted from the colour below.
func (g groundCell) blades(n int, seed uint32, light, dark float64) {
	for i := range n {
		x := Hash(i, 1, seed) * GroundCell
		y := Hash(i, 2, seed) * GroundCell
		h := 2 + 3*Hash(i, 3, seed)
		lean := (Hash(i, 4, seed) - 0.5) * 2
		f := light
		if Hash(i, 5, seed) < 0.5 {
			f = dark
		}
		base := g.at(int(x), int(y))
		col := Shade(base, f)
		col.A = 150
		for s := 0.0; s < h; s++ {
			g.blend(x+lean*s/h, y-s, col)
		}
	}
}

func (g groundCell) pebbles(n int, seed uint32, tint color.RGBA) {
	for i := range n {
		x := Hash(i, 1, seed) * GroundCell
		y := Hash(i, 2, seed) * GroundCell
		r := 0.8 + 1.4*Hash(i, 3, seed)
		col := Mix(g.at(int(x), int(y)), tint, 0.5+0.3*Hash(i, 4, seed))
		for dy := -2.0; dy <= 2; dy++ {
			for dx := -3.0; dx <= 3; dx++ {
				nx, ny := dx/(r*1.4), dy/r
				if d := nx*nx + ny*ny; d <= 1 {
					f := 1.08 - 0.25*ny - 0.1*nx
					c := Shade(col, f)
					c.A = 200
					g.blend(x+dx, y+dy, c)
				} else if ny > 0 && nx*nx+(ny-0.6)*(ny-0.6) <= 1 {
					g.blend(x+dx, y+dy, color.RGBA{0, 0, 0, 40})
				}
			}
		}
	}
}

func clumpy(base, dark, light color.RGBA, seed uint32) func(x, y float64) color.RGBA {
	return func(x, y float64) color.RGBA {
		low := gfbm(x, y, 3, 3, seed)
		mid := gfbm(x, y, 8, 3, seed+20)
		col := Mix(base, dark, 0.55*smoothstep(0.45, 0.75, low))
		col = Mix(col, light, 0.45*smoothstep(0.55, 0.8, mid))
		return Shade(col, 0.95+0.08*gnoise(x, y, 32, 64, seed+40))
	}
}

func groundGrass() *Canvas {
	g := groundCell{New(GroundCell, GroundCell)}
	g.fill(clumpy(RGB(92, 142, 60), RGB(72, 122, 50), RGB(116, 162, 70), 601))
	g.blades(900, 611, 1.18, 0.84)
	return g.c
}

func groundDryGrass() *Canvas {
	g := groundCell{New(GroundCell, GroundCell)}
	g.fill(clumpy(RGB(168, 156, 80), RGB(146, 136, 66), RGB(194, 180, 100), 621))
	g.blades(700, 631, 1.14, 0.84)
	return g.c
}

func groundDirt() *Canvas {
	g := groundCell{New(GroundCell, GroundCell)}
	g.fill(clumpy(RGB(140, 108, 72), RGB(118, 90, 60), RGB(160, 128, 90), 641))
	g.pebbles(90, 651, RGB(168, 152, 128))
	return g.c
}

func groundSand() *Canvas {
	g := groundCell{New(GroundCell, GroundCell)}
	base := clumpy(RGB(218, 196, 136), RGB(204, 180, 120), RGB(232, 214, 160), 661)
	g.fill(func(x, y float64) color.RGBA {
		warp := gfbm(x, y, 2, 2, 671)
		r := math.Sin(2 * math.Pi * (20*y/GroundCell + 1.5*warp + 2*x/GroundCell))
		f := 1 + 0.035*r
		if r > 0.85 {
			f += 0.03
		}
		return Shade(base(x, y), f)
	})
	g.pebbles(25, 681, RGB(170, 150, 110))
	return g.c
}

// GroundAtlas returns the 2x2 atlas of seamless ground textures: (0,0)
// grass, (1,0) dry grass, (0,1) dirt, (1,1) sand.
func GroundAtlas() *Canvas {
	c := New(GroundAtlasSize, GroundAtlasSize)
	for i, cell := range []*Canvas{groundGrass(), groundDryGrass(), groundDirt(), groundSand()} {
		c.Blit(cell, (i%2)*GroundCell, (i/2)*GroundCell)
	}
	return c
}
