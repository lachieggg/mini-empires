package art

import (
	"image/color"
	"math"
)

// The sprites use a low-poly look: rounded parts are lit as a ring of flat
// facets blended with a little smooth shading, colours are pushed slightly
// more saturated, and silhouettes get a soft dark outline tinted by the
// colour next to it.

// Facet returns a light factor like Lit, but constant over polygonal
// facets of the unit disc, so ellipses read as low-poly solids.
func Facet(nx, ny float64) float64 {
	r := math.Min(0.999, math.Hypot(nx, ny))
	a := math.Atan2(ny, nx)
	ring := int(r * 3)
	n := []int{5, 8, 11}[ring]
	off := float64(ring) * 0.5
	step := 2 * math.Pi / float64(n)
	si := math.Floor(a/step - off)
	ac := (si + 0.5 + off) * step
	rc := (float64(ring) + 0.55) / 3
	f := Lit(rc*math.Cos(ac), rc*math.Sin(ac))
	return 0.72*f + 0.28*Lit(nx, ny)
}

func lit(c color.RGBA) func(float64, float64) color.RGBA {
	return func(nx, ny float64) color.RGBA { return Shade(c, Facet(nx, ny)) }
}

// smoothLit is Gouraud-like smooth shading, for small round parts.
func smoothLit(c color.RGBA) func(float64, float64) color.RGBA {
	return func(nx, ny float64) color.RGBA { return Shade(c, Lit(nx, ny)) }
}

func softShadow(nx, ny float64) color.RGBA {
	d := 1 - (nx*nx + ny*ny)
	return color.RGBA{0, 0, 0, uint8(85 * math.Pow(math.Max(0, d), 0.6))}
}

func groundShadow(c *Canvas, at V, rx, ry float64) {
	c.Ellipse(at, rx, ry, softShadow)
}

// Saturate pushes every opaque pixel's colour away from its grey value.
func (c *Canvas) Saturate(k float64) {
	p := c.Img.Pix
	for i := 0; i < len(p); i += 4 {
		if p[i+3] < 200 {
			continue
		}
		r, g, b := float64(p[i]), float64(p[i+1]), float64(p[i+2])
		y := 0.3*r + 0.59*g + 0.11*b
		cl := func(v float64) uint8 { return uint8(math.Max(0, math.Min(255, y+(v-y)*k))) }
		p[i], p[i+1], p[i+2] = cl(r), cl(g), cl(b)
	}
}

// SoftOutline paints transparent pixels next to solid ones in a dark shade
// of their solid neighbours, then adds a faint second ring outside that.
func (c *Canvas) SoftOutline() {
	type px struct {
		x, y int
		col  color.RGBA
	}
	pass := func(minA uint8, shade float64, alpha uint8) {
		var pts []px
		for y := 0; y < c.H; y++ {
			for x := 0; x < c.W; x++ {
				if c.alpha(x, y) >= 128 {
					continue
				}
				var r, g, b, n float64
				for _, d := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					if c.alpha(x+d[0], y+d[1]) >= minA {
						i := c.Img.PixOffset(x+d[0], y+d[1])
						r += float64(c.Img.Pix[i])
						g += float64(c.Img.Pix[i+1])
						b += float64(c.Img.Pix[i+2])
						n++
					}
				}
				if n == 0 {
					continue
				}
				col := Mix(RGB(uint8(r/n), uint8(g/n), uint8(b/n)), outline, 0.55)
				pts = append(pts, px{x, y, Shade(col, shade)})
			}
		}
		for _, p := range pts {
			i := c.Img.PixOffset(p.x, p.y)
			a := float64(alpha) / 255
			old := c.Img.Pix[i : i+4]
			o := func(v, w uint8) uint8 { return uint8(float64(v)*a + float64(w)*(1-a)) }
			copy(old, []uint8{o(p.col.R, old[0]), o(p.col.G, old[1]), o(p.col.B, old[2]), o(255, old[3])})
		}
	}
	pass(128, 0.55, 255)
	pass(250, 0.5, 60)
}

// finish applies the shared colour grading and outline to a sprite.
func finish(c *Canvas) {
	c.Saturate(1.15)
	c.SoftOutline()
}
