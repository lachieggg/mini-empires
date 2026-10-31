package main

import (
	"image"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// The ground is drawn in one pass by a shader that blends four ground
// textures with smooth per-pixel weights and renders animated water. The
// weights live in two SplatSize x SplatSize maps covering the whole world:
//
//	splat: R grass, G dry grass, B dirt, A sand
//	extra: R forest shade, G water coverage (the shore is at 0.5),
//	       B water depth, A unused
const SplatSize = 512

type groundMaps struct {
	splat, extra []byte
	splatImg     *ebiten.Image
	extraImg     *ebiten.Image
	dirty        bool
	dirtyR       rect
	landDist     []float32
	uploaded     bool
	shaderStart  time.Time
}

var groundShader *ebiten.Shader

const groundShaderSrc = `//kage:unit pixels

package main

var Time float
var Cam vec2
var Zoom float
var Tiles float
var Size float

func sample0(p vec2) vec4 {
	p = clamp(p-0.5, vec2(0), vec2(Size-1.001))
	i := floor(p)
	f := fract(p)
	o := imageSrc0Origin()
	a := imageSrc0UnsafeAt(o + i + vec2(0.5, 0.5))
	b := imageSrc0UnsafeAt(o + i + vec2(1.5, 0.5))
	c := imageSrc0UnsafeAt(o + i + vec2(0.5, 1.5))
	d := imageSrc0UnsafeAt(o + i + vec2(1.5, 1.5))
	return mix(mix(a, b, f.x), mix(c, d, f.x), f.y)
}

func sample2(p vec2) vec4 {
	p = clamp(p-0.5, vec2(0), vec2(Size-1.001))
	i := floor(p)
	f := fract(p)
	o := imageSrc2Origin()
	a := imageSrc2UnsafeAt(o + i + vec2(0.5, 0.5))
	b := imageSrc2UnsafeAt(o + i + vec2(1.5, 0.5))
	c := imageSrc2UnsafeAt(o + i + vec2(0.5, 1.5))
	d := imageSrc2UnsafeAt(o + i + vec2(1.5, 1.5))
	return mix(mix(a, b, f.x), mix(c, d, f.x), f.y)
}

func sample3(p vec2) float {
	p = clamp(p-0.5, vec2(0), vec2(Size-1.001))
	i := floor(p)
	f := fract(p)
	o := imageSrc3Origin()
	a := imageSrc3UnsafeAt(o + i + vec2(0.5, 0.5)).r
	b := imageSrc3UnsafeAt(o + i + vec2(1.5, 0.5)).r
	c := imageSrc3UnsafeAt(o + i + vec2(0.5, 1.5)).r
	d := imageSrc3UnsafeAt(o + i + vec2(1.5, 1.5)).r
	return mix(mix(a, b, f.x), mix(c, d, f.x), f.y)
}

func tex(cell vec2, p vec2) vec3 {
	q := mod(p, 256.0)
	return imageSrc1UnsafeAt(imageSrc1Origin() + cell*256.0 + q).rgb
}

func hash(p vec2) float {
	return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453)
}

func vnoise(p vec2) float {
	i := floor(p)
	f := fract(p)
	u := f * f * (3.0 - 2.0*f)
	a := hash(i)
	b := hash(i + vec2(1, 0))
	c := hash(i + vec2(0, 1))
	d := hash(i + vec2(1, 1))
	return mix(mix(a, b, u.x), mix(c, d, u.x), u.y)
}

func waterShade(ground vec3, p vec2, water float, deep float) vec3 {
	t := Time
	d := clamp(deep, 0.0, 1.0)
	shallow := vec3(0.22, 0.60, 0.64)
	deepC := vec3(0.05, 0.22, 0.42)
	base := mix(shallow, deepC, smoothstep(0.0, 1.0, d))

	// Height field from crossing swells and drifting noise, differentiated
	// analytically for the surface normal.
	a1 := p.x*2.6 + p.y*1.3 + t*1.1
	a2 := -p.x*1.7 + p.y*2.9 + t*1.45
	a3 := p.x*4.3 - p.y*3.1 + t*2.0
	n1 := vnoise(p*1.8 + vec2(t*0.35, t*0.2))
	n2 := vnoise(p*3.7 - vec2(t*0.25, t*0.4))
	gx := 2.6*cos(a1) - 1.7*cos(a2) + 2.15*cos(a3) + (n1-0.5)*4.0
	gy := 1.3*cos(a1) + 2.9*cos(a2) - 1.55*cos(a3) + (n2-0.5)*4.0
	n := normalize(vec3(-gx*0.055, -gy*0.055, 1.0))

	l := normalize(vec3(-0.45, -0.55, 0.7))
	v := normalize(vec3(0.5, 0.5, 0.7))
	// The sun sits slightly behind the camera, so its reflection only shows
	// on wave slopes that tilt toward the viewer and flat water stays dark.
	sun := normalize(vec3(-0.11, -0.11, 0.99))
	h := normalize(sun + v)
	spec := pow(max(dot(n, h), 0.0), 120.0) * 1.2
	fres := pow(1.0-max(dot(n, v), 0.0), 3.0)
	sky := vec3(0.72, 0.86, 1.0)
	col := mix(base, sky, 0.06+0.5*fres)
	col *= 0.9 + 0.2*dot(n, l)

	c := vnoise(p*2.4+vec2(t*0.6, -t*0.4)) + vnoise(p*4.1-vec2(t*0.5, t*0.3))
	caust := pow(max(1.0-abs(c-1.0)*1.6, 0.0), 5.0) * (1.0 - d) * 0.22
	col += vec3(caust*0.8, caust, caust)
	col += vec3(spec)

	shore := 1.0 - smoothstep(0.3, 0.64, water)
	band := 0.5 + 0.5*sin(water*34.0-t*2.4+n1*5.0)
	foam := shore*smoothstep(0.55, 0.95, band)*0.85 + shore*0.2
	col = mix(col, vec3(0.94, 0.97, 1.0), clamp(foam, 0.0, 0.85))

	alpha := smoothstep(0.3, 0.52, water) * (0.62 + 0.38*d)
	return mix(ground*0.78, col, alpha)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	w := sample0(srcPos)
	ex := sample2(srcPos)
	world := srcPos / Size * Tiles
	iso := dstPos.xy/Zoom + Cam
	grass := tex(vec2(0, 0), iso)
	dry := tex(vec2(1, 0), iso)
	dirt := tex(vec2(0, 1), iso)
	sand := tex(vec2(1, 1), iso)
	sum := w.r + w.g + w.b + w.a + 0.0001
	col := (grass*w.r + dry*w.g + dirt*w.b + sand*w.a) / sum
	col *= 1.0 - 0.32*ex.r
	col *= 0.92 + 0.16*vnoise(world*0.35)
	water := ex.g
	col *= 1.0 - 0.3*smoothstep(0.1, 0.5, water)
	if water > 0.3 {
		col = waterShade(col, world, water, ex.b)
	}
	return vec4(col*sample3(srcPos), 1)
}
`

// buildGround computes the blend maps for the whole map.
func (g *Game) buildGround() {
	gm := &groundMaps{
		splat:       make([]byte, SplatSize*SplatSize*4),
		extra:       make([]byte, SplatSize*SplatSize*4),
		shaderStart: time.Now(),
	}
	g.ground = gm
	g.computeLandDist()
	g.updateGround(rect{0, 0, MapW - 1, MapH - 1})
}

// computeLandDist stores, for each water tile, its distance in tiles to the
// nearest land tile.
func (g *Game) computeLandDist() {
	gm := g.ground
	gm.landDist = make([]float32, MapW*MapH)
	var queue []pt
	for y := range MapH {
		for x := range MapW {
			if g.tile(x, y).Ter == TerWater {
				gm.landDist[y*MapW+x] = -1
			} else {
				queue = append(queue, pt{x, y})
			}
		}
	}
	for head := 0; head < len(queue); head++ {
		p := queue[head]
		for _, d := range dirs8 {
			x, y := p.X+d.X, p.Y+d.Y
			if inBounds(x, y) && gm.landDist[y*MapW+x] < 0 {
				gm.landDist[y*MapW+x] = gm.landDist[p.Y*MapW+p.X] + 1
				queue = append(queue, pt{x, y})
			}
		}
	}
}

// markGroundDirty schedules the blend maps around r for recomputation.
func (g *Game) markGroundDirty(r rect) {
	gm := g.ground
	if gm == nil {
		return
	}
	if !gm.dirty {
		gm.dirty, gm.dirtyR = true, r
		return
	}
	gm.dirtyR = rect{min(gm.dirtyR.x0, r.x0), min(gm.dirtyR.y0, r.y0), max(gm.dirtyR.x1, r.x1), max(gm.dirtyR.y1, r.y1)}
}

func smoothNoise(x, y float64) float64 {
	return fbm(x, y, 9157) - 0.5
}

// updateGround recomputes the blend-map texels over tiles r (plus a margin).
func (g *Game) updateGround(r rect) {
	gm := g.ground
	k := float64(SplatSize) / float64(MapW)
	px0, py0 := max(int(float64(r.x0-3)*k), 0), max(int(float64(r.y0-3)*k), 0)
	px1, py1 := min(int(float64(r.x1+4)*k), SplatSize-1), min(int(float64(r.y1+4)*k), SplatSize-1)
	const radius = 1.4
	for py := py0; py <= py1; py++ {
		for px := px0; px <= px1; px++ {
			wx := (float64(px) + 0.5) / k
			wy := (float64(py) + 0.5) / k
			// Warp the sample point so borders between terrain types wander.
			sx := wx + smoothNoise(wx*0.7, wy*0.7)*0.9
			sy := wy + smoothNoise(wx*0.7+40, wy*0.7)*0.9
			var w [4]float64
			var forest, water, deep, total, wtotal float64
			for ty := int(sy - radius - 1); ty <= int(sy+radius+1); ty++ {
				for tx := int(sx - radius - 1); tx <= int(sx+radius+1); tx++ {
					cx, cy := min(max(tx, 0), MapW-1), min(max(ty, 0), MapH-1)
					d := math.Hypot(float64(tx)+0.5-sx, float64(ty)+0.5-sy)
					if d >= radius {
						continue
					}
					kw := (1 - d/radius) * (1 - d/radius)
					t := g.tile(cx, cy)
					switch t.Ter {
					case TerGrass:
						w[0] += kw
					case TerForest:
						w[0] += kw
						forest += kw
					case TerDryGrass:
						w[1] += kw
					case TerDirt:
						w[2] += kw
					case TerSand, TerWater:
						w[3] += kw
					}
					total += kw
					// Water coverage uses the unwarped point, so the drawn
					// shore stays close to the walkable edge.
					dw := math.Hypot(float64(tx)+0.5-wx, float64(ty)+0.5-wy)
					if dw < radius {
						kk := (1 - dw/radius) * (1 - dw/radius)
						wtotal += kk
						if t.Ter == TerWater {
							water += kk
							deep += kk * float64(gm.landDist[cy*MapW+cx])
						}
					}
				}
			}
			i := (py*SplatSize + px) * 4
			if total == 0 {
				continue
			}
			// Sharpen the weights so each terrain keeps a solid core and
			// blends only near borders.
			var sum float64
			for j := range w {
				w[j] = math.Pow(w[j]/total, 2.2)
				sum += w[j]
			}
			for j := range w {
				gm.splat[i+j] = uint8(255 * w[j] / sum)
			}
			wf := 0.0
			if wtotal > 0 {
				wf = water / wtotal
			}
			depth := 0.0
			if water > 0 {
				depth = math.Min(deep/water/4, 1)
			}
			gm.extra[i] = uint8(255 * math.Min(forest/total*1.3, 1))
			gm.extra[i+1] = uint8(255 * wf)
			gm.extra[i+2] = uint8(255 * depth)
			gm.extra[i+3] = 255
		}
	}
	gm.uploaded = false
}

func (g *Game) drawGround(screen *ebiten.Image) {
	gm := g.ground
	if gm == nil {
		g.buildGround()
		gm = g.ground
	}
	if groundShader == nil {
		s, err := ebiten.NewShader([]byte(groundShaderSrc))
		if err != nil {
			panic(err)
		}
		groundShader = s
	}
	if gm.dirty {
		g.computeLandDist()
		g.updateGround(gm.dirtyR)
		gm.dirty = false
	}
	if gm.splatImg == nil {
		gm.splatImg = ebiten.NewImageWithOptions(image.Rect(0, 0, SplatSize, SplatSize), &ebiten.NewImageOptions{Unmanaged: true})
		gm.extraImg = ebiten.NewImageWithOptions(image.Rect(0, 0, SplatSize, SplatSize), &ebiten.NewImageOptions{Unmanaged: true})
	}
	if !gm.uploaded {
		gm.splatImg.WritePixels(gm.splat)
		gm.extraImg.WritePixels(gm.extra)
		gm.uploaded = true
	}
	corners := [4][2]float64{{0, 0}, {worldW(), 0}, {worldW(), worldH()}, {0, worldH()}}
	src := [4][2]float32{{0, 0}, {SplatSize, 0}, {SplatSize, SplatSize}, {0, SplatSize}}
	vs := make([]ebiten.Vertex, 4)
	for i, c := range corners {
		sx, sy := g.toView(c[0], c[1])
		vs[i] = ebiten.Vertex{DstX: float32(sx), DstY: float32(sy), SrcX: src[i][0], SrcY: src[i][1], ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	}
	fogTex := noFogImage()
	if g.fog != nil {
		fogTex = g.fogImage()
	}
	op := &ebiten.DrawTrianglesShaderOptions{
		Images: [4]*ebiten.Image{gm.splatImg, sprites().groundAtlas, gm.extraImg, fogTex},
		Uniforms: map[string]any{
			"Time":  float32(time.Since(gm.shaderStart).Seconds()),
			"Cam":   []float32{float32(g.camX), float32(g.camY)},
			"Zoom":  float32(g.zoomLevel()),
			"Tiles": float32(MapW),
			"Size":  float32(SplatSize),
		},
	}
	screen.DrawTrianglesShader(vs, []uint16{0, 1, 2, 0, 2, 3}, groundShader, op)
}
