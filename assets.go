package main

import (
	"embed"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"mini-empires/internal/art"
)

//go:generate go run ./cmd/spritegen -out assets

//go:embed assets/*.png
var assetFS embed.FS

// Sprite is an image with the screen offset of its anchor: the footprint
// center for tiles and buildings.
type Sprite struct {
	img    *ebiten.Image
	rgba   *image.RGBA
	ax, ay float64
}

func (s *Sprite) opaque(px, py int) bool {
	if !(image.Point{px, py}).In(s.rgba.Rect) {
		return false
	}
	return s.rgba.Pix[s.rgba.PixOffset(px, py)+3] > 100
}

type unitFrames [8][art.UnitFrames]*ebiten.Image

type spriteSet struct {
	trees       []*Sprite
	gold        *Sprite
	stone       *Sprite
	food        [3]*Sprite
	farm        [3]*Sprite
	bld         [numBldKinds][2][NumAges]*Sprite
	found       [5]*Sprite
	units       [numUnitKinds][2][]unitFrames // kind, team, tier
	walls       [2][16]*Sprite
	vills       [2][2][art.NumVillagerTasks]unitFrames // team, female, task
	ring        *ebiten.Image
	icons       [numRes]*ebiten.Image
	buttons     map[string]*ebiten.Image
	groundAtlas *ebiten.Image
}

var spr *spriteSet

func loadRGBA(name string) *image.RGBA {
	f, err := assetFS.Open("assets/" + name + ".png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Rect, img, img.Bounds().Min, draw.Src)
	return rgba
}

func loadSprite(name string, size int) *Sprite {
	rgba := loadRGBA(name)
	b := rgba.Bounds()
	return &Sprite{
		img:  ebiten.NewImageFromImage(rgba),
		rgba: rgba,
		ax:   float64(b.Dx()) / 2,
		ay:   float64(b.Dy() - size*16),
	}
}

func loadFrames(name string, fw, fh int) unitFrames {
	sheet := ebiten.NewImageFromImage(loadRGBA(name))
	var f unitFrames
	for d := range 8 {
		for c := range art.UnitFrames {
			r := image.Rect(c*fw, d*fh, (c+1)*fw, (d+1)*fh)
			f[d][c] = sheet.SubImage(r).(*ebiten.Image)
		}
	}
	return f
}

// unitAnchor returns a unit's frame size and the position of its feet
// within the frame. Siege engines use bigger frames.
func unitAnchor(k UnitKind) (fw, fh, ax, ay float64) {
	if unitDefs[k].Siege {
		return art.SiegeFW, art.SiegeFH, art.SiegeAX, art.SiegeAY
	}
	return art.UnitFW, art.UnitFH, art.UnitAX, art.UnitAY
}

var bldFiles = [numBldKinds]string{
	TownCenter: "towncenter", House: "house", Mill: "mill", LumberCamp: "lumbercamp", MiningCamp: "miningcamp",
	Barracks: "barracks", ArcheryRange: "archeryrange", Stable: "stable", Blacksmith: "blacksmith", Tower: "tower",
	SiegeWorkshop: "siegeworkshop", Monastery: "monastery", Castle: "castle",
}

var unitFiles = [numUnitKinds]string{
	Villager: "villager", Militia: "militia", Spearman: "spearman", Archer: "archer",
	Scout: "scout", Knight: "knight", CavalryArcher: "cavalryarcher", Longbowman: "longbowman", Monk: "monk",
	Ram: "ram", Mangonel: "mangonel", Trebuchet: "trebuchet", Deer: "deer", Boar: "boar",
}

func sprites() *spriteSet {
	if spr != nil {
		return spr
	}
	s := &spriteSet{buttons: map[string]*ebiten.Image{}}
	for i := range 6 {
		s.trees = append(s.trees, loadSprite(fmt.Sprintf("tree%d", i), 1))
	}
	s.gold = loadSprite("gold", 1)
	s.stone = loadSprite("stone", 1)
	s.food = [3]*Sprite{loadSprite("berries", 1), loadSprite("carcass_deer", 1), loadSprite("carcass_boar", 1)}
	for i := range s.farm {
		s.farm[i] = loadSprite(fmt.Sprintf("farm%d", i), 2)
	}
	for k, n := range bldFiles {
		if n == "" {
			continue
		}
		for t := range 2 {
			for a := range NumAges {
				name := fmt.Sprintf("%s_%d", n, t)
				if BldKind(k) == TownCenter || BldKind(k) == House {
					name = fmt.Sprintf("%s_age%d", name, a)
				} else if a > 0 {
					s.bld[k][t][a] = s.bld[k][t][0]
					continue
				}
				s.bld[k][t][a] = loadSprite(name, bldDefs[k].Size)
			}
		}
	}
	for m := range 16 {
		s.walls[0][m] = loadSprite(fmt.Sprintf("palisade_%d", m), 1)
		s.walls[1][m] = loadSprite(fmt.Sprintf("stonewall_%d", m), 1)
	}
	for i := 1; i <= 4; i++ {
		s.found[i] = loadSprite(fmt.Sprintf("foundation%d", i), i)
	}
	for k, n := range unitFiles {
		fw, fh, _, _ := unitAnchor(UnitKind(k))
		if unitDefs[k].Animal {
			f := loadFrames(n, int(fw), int(fh))
			s.units[k] = [2][]unitFrames{{f}, {f}}
			continue
		}
		files := []string{n}
		if ts, ok := unitTiers[UnitKind(k)]; ok {
			files = nil
			for _, t := range ts {
				files = append(files, t.File)
			}
		}
		for t := range 2 {
			for _, f := range files {
				s.units[k][t] = append(s.units[k][t], loadFrames(fmt.Sprintf("%s_%d", f, t), int(fw), int(fh)))
			}
		}
	}
	for t := range 2 {
		for f, g := range []string{"m", "f"} {
			for task := range art.NumVillagerTasks {
				s.vills[t][f][task] = loadFrames(fmt.Sprintf("villager_%d_%s_%s", t, g, task), art.UnitFW, art.UnitFH)
			}
		}
	}
	s.ring = ebiten.NewImageFromImage(loadRGBA("ring"))
	s.groundAtlas = ebiten.NewImageFromImageWithOptions(loadRGBA("ground_atlas"), &ebiten.NewImageFromImageOptions{Unmanaged: true})
	icons := ebiten.NewImageFromImage(loadRGBA("icons"))
	for i := range s.icons {
		s.icons[i] = icons.SubImage(image.Rect(i*art.IconSize, 0, (i+1)*art.IconSize, art.IconSize)).(*ebiten.Image)
	}
	names := []string{"eco", "mil", "back", "age1", "age2", "age3", "cancel", "stop", "townbell", "ungarrison"}
	for _, d := range techDefs {
		if d.Icon != "" {
			names = append(names, d.Icon)
		}
	}
	for _, n := range names {
		s.buttons[n] = ebiten.NewImageFromImage(loadRGBA("button_" + n))
	}
	spr = s
	return s
}

func (s *spriteSet) resource(r *Resource) *Sprite {
	switch r.Kind {
	case Gold:
		return s.gold
	case Stone:
		return s.stone
	case Food:
		return s.food[min(int(r.Variant), 2)]
	}
	return s.trees[int(r.Variant)%len(s.trees)]
}

// wallMask says which neighboring tiles hold the same kind of wall:
// 1 north, 2 east, 4 south, 8 west.
func (g *Game) wallMask(b *Building) int {
	m := 0
	for i, d := range []pt{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
		if o := g.buildingAtTile(b.TX+d.X, b.TY+d.Y); o != nil && o.Kind == b.Kind {
			m |= 1 << i
		}
	}
	return m
}

func (g *Game) bldSprite(b *Building) *Sprite {
	s := sprites()
	switch {
	case b.def().Wall:
		return s.walls[boolIdx(b.Kind == StoneWall)][g.wallMask(b)]
	case !b.Built:
		return s.found[b.def().Size]
	case b.Kind == Farm:
		f := float64(b.Food) / float64(b.def().Food+g.bonus(b.Team).FarmFood)
		switch {
		case f > 0.66:
			return s.farm[0]
		case f > 0.33:
			return s.farm[1]
		}
		return s.farm[2]
	}
	return s.bld[b.Kind][min(b.Team, 1)][g.players[min(b.Team, 1)].Age]
}

func (s *spriteSet) finished(k BldKind, team, age int) *Sprite {
	switch {
	case k == Farm:
		return s.farm[0]
	case k == PalisadeWall:
		return s.walls[0][10]
	case k == StoneWall:
		return s.walls[1][10]
	}
	return s.bld[k][team][age]
}

func (s *spriteSet) bldIcon(k BldKind, team, age int) *ebiten.Image {
	return s.finished(k, team, age).img
}

// frames returns the sheet for unit kind k of a team at an upgrade tier.
func (s *spriteSet) frames(k UnitKind, team, tier int) *unitFrames {
	fs := s.units[k][min(team, 1)]
	return &fs[min(tier, len(fs)-1)]
}

func (s *spriteSet) unitIcon(k UnitKind, team, tier int) *ebiten.Image {
	return s.frames(k, team, tier)[1][0]
}

// techIcon is a technology's button icon; unit upgrades show the unit.
func (s *spriteSet) techIcon(t Tech, team int) *ebiten.Image {
	d := &techDefs[t]
	if d.Icon != "" {
		return s.buttons[d.Icon]
	}
	return s.unitIcon(d.Line, team, d.Tier)
}

func ax(k UnitKind) float64 { _, _, x, _ := unitAnchor(k); return x }
func ay(k UnitKind) float64 { _, _, _, y := unitAnchor(k); return y }
