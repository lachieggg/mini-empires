package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// fogMap is the fog of war for the human player. Tiles start unexplored
// (black). Tiles a unit or building can see are visible, and tiles that
// were seen once stay explored (dimmed) and show the buildings last seen.
type fogMap struct {
	team     int
	explored []bool
	visible  []bool
	t        float64
	tex      []byte
	img      *ebiten.Image
	dirty    bool
}

const fogInterval = 0.25

func newFog(team int) *fogMap {
	return &fogMap{
		team:     team,
		explored: make([]bool, MapW*MapH),
		visible:  make([]bool, MapW*MapH),
		tex:      make([]byte, SplatSize*SplatSize*4),
		dirty:    true,
	}
}

func (g *Game) updateFog() {
	f := g.fog
	if f == nil {
		return
	}
	if f.t -= dt; f.t > 0 {
		return
	}
	f.t = fogInterval
	clear(f.visible)
	reveal := func(cx, cy, r float64) {
		tr := int(r/TileSize) + 1
		tx, ty := toTile(cx), toTile(cy)
		r2 := (r / TileSize) * (r / TileSize)
		for y := ty - tr; y <= ty+tr; y++ {
			for x := tx - tr; x <= tx+tr; x++ {
				dx, dy := float64(x-tx), float64(y-ty)
				if inBounds(x, y) && dx*dx+dy*dy <= r2 {
					f.visible[y*MapW+x] = true
				}
			}
		}
	}
	for _, u := range g.units {
		if u.Team == f.team && u.Inside == nil {
			reveal(u.X, u.Y, u.def().Sight)
		}
	}
	for _, b := range g.buildings {
		if b.Team == f.team {
			cx, cy := b.center()
			reveal(cx, cy, float64(b.def().Size*TileSize/2)+4*TileSize+b.def().Range)
		}
	}
	for i, v := range f.visible {
		if v {
			f.explored[i] = true
		}
	}
	f.dirty = true
}

// seenTile reports whether tile (x, y) has ever been seen, and canSee
// whether it is in sight now. Without fog everything is seen.
func (g *Game) seenTile(x, y int) bool {
	return g.fog == nil || inBounds(x, y) && g.fog.explored[y*MapW+x]
}

func (g *Game) canSee(x, y int) bool {
	return g.fog == nil || inBounds(x, y) && g.fog.visible[y*MapW+x]
}

// unitShown reports whether u is drawn: your own units always, others only
// in sight.
func (g *Game) unitShown(u *Unit) bool {
	return g.fog == nil || u.Team == g.fog.team || g.canSee(toTile(u.X), toTile(u.Y))
}

// fogImage returns the fog as a texture matching the ground blend maps:
// 255 visible, 110 explored, 0 unexplored.
func (g *Game) fogImage() *ebiten.Image {
	f := g.fog
	if f.img == nil {
		f.img = ebiten.NewImageWithOptions(image.Rect(0, 0, SplatSize, SplatSize), &ebiten.NewImageOptions{Unmanaged: true})
	}
	if !f.dirty {
		return f.img
	}
	f.dirty = false
	level := func(x, y int) float64 {
		x, y = min(max(x, 0), MapW-1), min(max(y, 0), MapH-1)
		switch i := y*MapW + x; {
		case f.visible[i]:
			return 255
		case f.explored[i]:
			return 110
		}
		return 0
	}
	// Blend between tile centers so the fog has soft, round edges rather
	// than following the tile diamonds.
	k := float64(MapW) / SplatSize
	for py := range SplatSize {
		wy := (float64(py)+0.5)*k - 0.5
		ty := int(wy)
		fy := wy - float64(ty)
		for px := range SplatSize {
			wx := (float64(px)+0.5)*k - 0.5
			tx := int(wx)
			fx := wx - float64(tx)
			top := level(tx, ty)*(1-fx) + level(tx+1, ty)*fx
			bot := level(tx, ty+1)*(1-fx) + level(tx+1, ty+1)*fx
			v := byte(top*(1-fy) + bot*fy)
			o := (py*SplatSize + px) * 4
			f.tex[o], f.tex[o+1], f.tex[o+2], f.tex[o+3] = v, v, v, 255
		}
	}
	f.img.WritePixels(f.tex)
	return f.img
}

// whiteImage stands in for the fog texture when there is no fog.
var whiteImage *ebiten.Image

func noFogImage() *ebiten.Image {
	if whiteImage == nil {
		whiteImage = ebiten.NewImageWithOptions(image.Rect(0, 0, SplatSize, SplatSize), &ebiten.NewImageOptions{Unmanaged: true})
		whiteImage.Fill(white)
	}
	return whiteImage
}
