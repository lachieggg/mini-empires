// Package art draws the game's sprites procedurally into RGBA images.
package art

import (
	"image"
	"image/color"
	"math"
)

type V struct{ X, Y float64 }

func (a V) Add(b V) V       { return V{a.X + b.X, a.Y + b.Y} }
func (a V) Sub(b V) V       { return V{a.X - b.X, a.Y - b.Y} }
func (a V) Mul(k float64) V { return V{a.X * k, a.Y * k} }

type Canvas struct {
	Img  *image.RGBA
	W, H int
}

func New(w, h int) *Canvas {
	return &Canvas{image.NewRGBA(image.Rect(0, 0, w, h)), w, h}
}

func RGB(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }

func Shade(c color.RGBA, f float64) color.RGBA {
	s := func(v uint8) uint8 { return uint8(math.Max(0, math.Min(255, float64(v)*f))) }
	return color.RGBA{s(c.R), s(c.G), s(c.B), c.A}
}

func Mix(a, b color.RGBA, t float64) color.RGBA {
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), m(a.A, b.A)}
}

// Blend draws a straight-alpha colour over the premultiplied image.
func (c *Canvas) Blend(x, y int, col color.RGBA) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H || col.A == 0 {
		return
	}
	p := c.Img.Pix[c.Img.PixOffset(x, y):]
	a := float64(col.A) / 255
	p[0] = uint8(float64(col.R)*a + float64(p[0])*(1-a))
	p[1] = uint8(float64(col.G)*a + float64(p[1])*(1-a))
	p[2] = uint8(float64(col.B)*a + float64(p[2])*(1-a))
	p[3] = uint8(255*a + float64(p[3])*(1-a))
}

func (c *Canvas) alpha(x, y int) uint8 {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return 0
	}
	return c.Img.Pix[c.Img.PixOffset(x, y)+3]
}

func bbox(pts ...V) (int, int, int, int) {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range pts {
		x0, y0, x1, y1 = math.Min(x0, p.X), math.Min(y0, p.Y), math.Max(x1, p.X), math.Max(y1, p.Y)
	}
	return int(math.Floor(x0)), int(math.Floor(y0)), int(math.Ceil(x1)), int(math.Ceil(y1))
}

// Quad fills the parallelogram p + u*e1 + v*e2 for u, v in [0, 1].
func (c *Canvas) Quad(p, e1, e2 V, fn func(u, v float64) color.RGBA) {
	c.para(p, e1, e2, false, fn)
}

// Tri fills the triangle p, p+e1, p+e2. fn gets barycentric u (along e1) and v (along e2).
func (c *Canvas) Tri(p, e1, e2 V, fn func(u, v float64) color.RGBA) {
	c.para(p, e1, e2, true, fn)
}

func (c *Canvas) para(p, e1, e2 V, tri bool, fn func(u, v float64) color.RGBA) {
	det := e1.X*e2.Y - e1.Y*e2.X
	if math.Abs(det) < 1e-9 {
		return
	}
	x0, y0, x1, y1 := bbox(p, p.Add(e1), p.Add(e2), p.Add(e1).Add(e2))
	for py := y0; py <= y1; py++ {
		for px := x0; px <= x1; px++ {
			dx, dy := float64(px)+0.5-p.X, float64(py)+0.5-p.Y
			u := (dx*e2.Y - dy*e2.X) / det
			v := (e1.X*dy - e1.Y*dx) / det
			if u < 0 || v < 0 || u > 1 || v > 1 || (tri && u+v > 1) {
				continue
			}
			c.Blend(px, py, fn(u, v))
		}
	}
}

// Ellipse fills an ellipse. fn gets the offset from the centre normalised to the radii.
func (c *Canvas) Ellipse(ctr V, rx, ry float64, fn func(nx, ny float64) color.RGBA) {
	for py := int(math.Floor(ctr.Y - ry)); py <= int(math.Ceil(ctr.Y+ry)); py++ {
		for px := int(math.Floor(ctr.X - rx)); px <= int(math.Ceil(ctr.X+rx)); px++ {
			nx, ny := (float64(px)+0.5-ctr.X)/rx, (float64(py)+0.5-ctr.Y)/ry
			if nx*nx+ny*ny <= 1 {
				c.Blend(px, py, fn(nx, ny))
			}
		}
	}
}

func (c *Canvas) Line(a, b V, w float64, col color.RGBA) {
	x0, y0, x1, y1 := bbox(a, b)
	r := int(math.Ceil(w))
	d := b.Sub(a)
	l2 := d.X*d.X + d.Y*d.Y
	for py := y0 - r; py <= y1+r; py++ {
		for px := x0 - r; px <= x1+r; px++ {
			p := V{float64(px) + 0.5, float64(py) + 0.5}
			t := 0.0
			if l2 > 0 {
				t = math.Max(0, math.Min(1, ((p.X-a.X)*d.X+(p.Y-a.Y)*d.Y)/l2))
			}
			q := a.Add(d.Mul(t))
			if math.Hypot(p.X-q.X, p.Y-q.Y) <= w/2 {
				c.Blend(px, py, col)
			}
		}
	}
}

// Outline paints every mostly transparent pixel next to a solid one.
func (c *Canvas) Outline(col color.RGBA) {
	var pts [][2]int
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			if c.alpha(x, y) >= 128 {
				continue
			}
			if c.alpha(x-1, y) >= 128 || c.alpha(x+1, y) >= 128 || c.alpha(x, y-1) >= 128 || c.alpha(x, y+1) >= 128 {
				pts = append(pts, [2]int{x, y})
			}
		}
	}
	for _, p := range pts {
		i := c.Img.PixOffset(p[0], p[1])
		copy(c.Img.Pix[i:i+4], []uint8{col.R, col.G, col.B, 255})
	}
}

// Blit copies src onto c at (x, y).
func (c *Canvas) Blit(src *Canvas, x, y int) {
	for sy := 0; sy < src.H; sy++ {
		for sx := 0; sx < src.W; sx++ {
			i := src.Img.PixOffset(sx, sy)
			copy(c.Img.Pix[c.Img.PixOffset(x+sx, y+sy):], src.Img.Pix[i:i+4])
		}
	}
}

func Hash(x, y int, seed uint32) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + seed*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xffffff) / float64(0xffffff)
}

// Smooth is bilinear value noise in [0, 1].
func Smooth(x, y float64, seed uint32) float64 {
	ix, iy := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(ix), y-float64(iy)
	fx, fy = fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	a := Hash(ix, iy, seed) + (Hash(ix+1, iy, seed)-Hash(ix, iy, seed))*fx
	b := Hash(ix, iy+1, seed) + (Hash(ix+1, iy+1, seed)-Hash(ix, iy+1, seed))*fx
	return a + (b-a)*fy
}

// Lit returns a light factor for a point on a sphere, lit from the upper left.
func Lit(nx, ny float64) float64 {
	nz := math.Sqrt(math.Max(0, 1-nx*nx-ny*ny))
	d := -0.5*nx - 0.6*ny + 0.62*nz
	return 0.55 + 0.6*math.Max(0, d)
}
