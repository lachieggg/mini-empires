// Command spritegen renders the game's sprites into PNG files.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"mini-empires/internal/art"
)

type sprite struct {
	name  string
	group string
	img   *image.RGBA
}

var teams = []struct{ R, G, B uint8 }{{45, 95, 215}, {205, 45, 40}}

func sprites() []sprite {
	var out []sprite
	add := func(group, name string, c *art.Canvas) { out = append(out, sprite{name, group, c.Img}) }
	for i := range 4 {
		add("terrain", fmt.Sprintf("grass%d", i), art.Grass(i))
	}
	for i := range 2 {
		add("terrain", fmt.Sprintf("drygrass%d", i), art.DryGrass(i))
		add("terrain", fmt.Sprintf("dirt%d", i), art.Dirt(i))
		add("terrain", fmt.Sprintf("sand%d", i), art.Sand(i))
		add("terrain", fmt.Sprintf("forest%d", i), art.Forest(i))
	}
	for i := range 4 {
		add("terrain", fmt.Sprintf("water%d", i), art.Water(i))
	}
	for i := range art.NumTrees {
		add("resource", fmt.Sprintf("tree%d", i), art.Tree(i))
	}
	add("resource", "gold", art.GoldMine())
	add("resource", "stone", art.StoneMine())
	add("resource", "berries", art.Berries())
	add("resource", "carcass_deer", art.Carcass(art.Deer))
	add("resource", "carcass_boar", art.Carcass(art.Boar))
	for i := range 3 {
		add("resource", fmt.Sprintf("farm%d", i), art.Farm(i))
	}
	for i := 1; i <= 4; i++ {
		add("building", fmt.Sprintf("foundation%d", i), art.Foundation(i))
	}
	for t, col := range teams {
		team := art.RGB(col.R, col.G, col.B)
		for a := range art.NumAges {
			add("building", fmt.Sprintf("towncenter_%d_age%d", t, a), art.TownCenter(team, a))
		}
		for a := range art.NumAges {
			add("building", fmt.Sprintf("house_%d_age%d", t, a), art.House(team, a))
		}
		add("building", fmt.Sprintf("mill_%d", t), art.Mill(team))
		add("building", fmt.Sprintf("lumbercamp_%d", t), art.LumberCamp(team))
		add("building", fmt.Sprintf("miningcamp_%d", t), art.MiningCamp(team))
		add("building", fmt.Sprintf("barracks_%d", t), art.Barracks(team))
		add("building", fmt.Sprintf("archeryrange_%d", t), art.ArcheryRange(team))
		add("building", fmt.Sprintf("stable_%d", t), art.Stable(team))
		add("building", fmt.Sprintf("tower_%d", t), art.Tower(team))
		add("building", fmt.Sprintf("castle_%d", t), art.Castle(team))
		add("building", fmt.Sprintf("blacksmith_%d", t), art.Blacksmith(team))
		add("building", fmt.Sprintf("siegeworkshop_%d", t), art.SiegeWorkshop(team))
		add("building", fmt.Sprintf("monastery_%d", t), art.Monastery(team))
	}
	for t, col := range teams {
		team := art.RGB(col.R, col.G, col.B)
		for u := art.Villager; u < art.NumUnits; u++ {
			if !u.Animal() {
				add("unit", fmt.Sprintf("%s_%d", u, t), art.UnitSheet(u, team))
			}
		}
	}
	add("unit", "deer", art.UnitSheet(art.Deer, art.RGB(0, 0, 0)))
	for t, col := range teams {
		team := art.RGB(col.R, col.G, col.B)
		for k := art.Ram; k < art.NumSiege; k++ {
			add("siege", fmt.Sprintf("%s_%d", k, t), art.SiegeSheet(k, team))
		}
	}
	for m := range 16 {
		add("wall", fmt.Sprintf("palisade_%d", m), art.Palisade(m))
		add("wall", fmt.Sprintf("stonewall_%d", m), art.StoneWall(m))
	}
	for t, col := range teams {
		team := art.RGB(col.R, col.G, col.B)
		for _, g := range []string{"m", "f"} {
			for task := art.TaskIdle; task < art.NumVillagerTasks; task++ {
				add("villager", fmt.Sprintf("villager_%d_%s_%s", t, g, task), art.VillagerSheet(team, g == "f", task))
			}
		}
	}
	add("unit", "boar", art.UnitSheet(art.Boar, art.RGB(0, 0, 0)))
	add("ground", "ground_atlas", art.GroundAtlas())
	add("ui", "ring", art.Ring())
	add("ui", "icons", art.Icons())
	for b := range art.NumButtons {
		add("ui", "button_"+art.ButtonNames[b], art.Button(b))
	}
	return out
}

func savePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}

func main() {
	out := flag.String("out", "assets", "output directory")
	sheet := flag.String("sheet", "", "also write a 2x contact sheet of all sprites to this PNG path")
	villSheet := flag.String("villagersheet", "", "also write a 2x contact sheet of the villager sheets to this PNG path")
	flag.Parse()
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	all := sprites()
	for _, s := range all {
		savePNG(filepath.Join(*out, s.name+".png"), s.img)
	}
	if *villSheet != "" {
		savePNG(*villSheet, contactSheet(all, "villager"))
	}
	if *sheet != "" {
		savePNG(*sheet, contactSheet(all, ""))
		base := strings.TrimSuffix(*sheet, ".png")
		for _, g := range []string{"terrain", "resource", "building", "wall", "unit", "siege", "villager", "ui"} {
			savePNG(base+"-"+g+".png", contactSheet(all, g))
		}
		for _, s := range all {
			if s.name == "ground_atlas" {
				savePNG(base+"-ground-preview.png", groundPreview(s.img))
			}
		}
	}
}

// groundPreview shows each atlas cell tiled 2x2 in one quadrant, so seams
// at the cell period would show as lines through each quadrant's middle.
func groundPreview(atlas *image.RGBA) *image.RGBA {
	const cell = art.GroundCell
	out := image.NewRGBA(image.Rect(0, 0, 4*cell, 4*cell))
	for y := range 4 * cell {
		for x := range 4 * cell {
			q := (x / (2 * cell)) * cell
			r := (y / (2 * cell)) * cell
			i := out.PixOffset(x, y)
			j := atlas.PixOffset(q+x%cell, r+y%cell)
			copy(out.Pix[i:i+4], atlas.Pix[j:j+4])
		}
	}
	return out
}

// contactSheet lays the sprites of one group (or all, for "") out on a
// tiled grass background and scales the result up 2x.
func contactSheet(all []sprite, group string) *image.RGBA {
	const maxW, pad = 1600, 10
	type placed struct {
		s    sprite
		x, y int
	}
	var ps []placed
	x, y, rowH, w := pad, pad, 0, 0
	for _, s := range all {
		if (group != "" && s.group != group) || s.group == "ground" {
			continue
		}
		b := s.img.Bounds()
		if x+b.Dx()+pad > maxW && x > pad {
			x, y, rowH = pad, y+rowH+pad, 0
		}
		ps = append(ps, placed{s, x, y})
		x += b.Dx() + pad
		rowH = max(rowH, b.Dy())
		w = max(w, x)
	}
	h := y + rowH + pad
	terrains := []string{"grass0", "drygrass0", "dirt0", "sand0", "water0", "forest0"}
	if group == "" || group == "terrain" {
		h += 4*32 + pad
	}
	bg := image.NewRGBA(image.Rect(0, 0, w, h))
	tile := func(img *image.RGBA, x0, y0, x1, y1 int) {
		for ty := -1; ty*16 < y1-y0+32; ty++ {
			for tx := -1; tx*64 < x1-x0+64; tx++ {
				px, py := x0+tx*64+(ty%2)*32, y0+ty*16
				r := image.Rect(px, py, px+64, py+32).Intersect(image.Rect(x0, y0, x1, y1))
				draw.Draw(bg, r, img, image.Point{r.Min.X - px, r.Min.Y - py}, draw.Over)
			}
		}
	}
	byName := map[string]*image.RGBA{}
	for _, s := range all {
		byName[s.name] = s.img
	}
	tile(byName["grass1"], 0, 0, w, h)
	if group == "" || group == "terrain" {
		pw := (w - pad) / len(terrains)
		for i, n := range terrains {
			tile(byName[n], pad+i*pw, h-4*32-pad, pad+i*pw+pw-pad, h-pad)
		}
	}
	for _, p := range ps {
		b := p.s.img.Bounds()
		draw.Draw(bg, image.Rect(p.x, p.y, p.x+b.Dx(), p.y+b.Dy()), p.s.img, image.Point{}, draw.Over)
	}
	big := image.NewRGBA(image.Rect(0, 0, w*2, h*2))
	for yy := 0; yy < h*2; yy++ {
		for xx := 0; xx < w*2; xx++ {
			i, j := big.PixOffset(xx, yy), bg.PixOffset(xx/2, yy/2)
			copy(big.Pix[i:i+4], bg.Pix[j:j+4])
		}
	}
	return big
}
