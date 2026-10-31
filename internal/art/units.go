package art

import (
	"image/color"
	"math"
	"sort"
)

// Unit selects a unit or animal sprite sheet.
type Unit int

const (
	Villager Unit = iota
	Militia
	Spearman
	Archer
	Champion
	Scout
	Knight
	Deer
	Boar
	ManAtArms
	Longswordsman
	Pikeman
	Crossbowman
	Arbalest
	LightCavalry
	Cavalier
	CavalryArcher
	Longbowman
	Monk
	NumUnits
)

var unitNames = [NumUnits]string{"villager", "militia", "spearman", "archer", "champion", "scout", "knight", "deer", "boar", "manatarms", "longswordsman", "pikeman", "crossbowman", "arbalest", "lightcavalry", "cavalier", "cavalryarcher", "longbowman", "monk"}

func (u Unit) String() string { return unitNames[u] }

// Animal reports whether the sheet has no team colour.
func (u Unit) Animal() bool { return u == Deer || u == Boar }

// rig places body parts at ground offsets and heights, then draws them back
// to front.
type rig struct {
	c    *Canvas
	o    V
	f, r V
	els  []el
}

type el struct {
	depth float64
	draw  func()
}

func newRig(c *Canvas, o V, dir int) *rig {
	a := float64(dir) * math.Pi / 4
	f := V{math.Cos(a), math.Sin(a)}
	return &rig{c: c, o: o, f: f, r: V{-f.Y, f.X}}
}

func (k *rig) P(g V, h float64) V { return V{k.o.X + g.X, k.o.Y + g.Y*0.5 - h} }

// at is a ground offset a along the facing and b to the right.
func (k *rig) at(a, b float64) V { return k.f.Mul(a).Add(k.r.Mul(b)) }

func (k *rig) add(depth float64, fn func()) { k.els = append(k.els, el{depth, fn}) }

func (k *rig) flush() {
	sort.SliceStable(k.els, func(i, j int) bool { return k.els[i].depth < k.els[j].depth })
	for _, e := range k.els {
		e.draw()
	}
	k.els = nil
}

func (k *rig) capsule(g0 V, h0 float64, g1 V, h1, rad float64, col color.RGBA) {
	k.c.Capsule(k.P(g0, h0), k.P(g1, h1), rad, lit(col))
}

func (k *rig) limb(g0 V, h0 float64, g1 V, h1, w float64, col color.RGBA) {
	k.c.Line(k.P(g0, h0), k.P(g1, h1), w, col)
}

type anim struct {
	frame int
	swing float64 // walk leg swing, -1..1
	act   int     // -1, or 0 wind-up / 1 strike
	dead  bool
}

func animFor(frame int) anim {
	a := anim{frame: frame, act: -1}
	switch {
	case frame >= 1 && frame <= 4:
		a.swing = math.Sin(float64(frame-1) * math.Pi / 2)
	case frame == 5 || frame == 6:
		a.act = frame - 5
	case frame == 7:
		a.dead = true
	}
	return a
}

const (
	headHair = iota
	headHelm
	headCap
	headHood
	headGreat
	headLeather
	headTonsure
)

const (
	wTool = iota
	wSword
	wSpear
	wBow
	wGreat
	wLance
	wNone
	wXbow
	wStaff
)

// Villager tools.
const (
	toolNone = iota
	toolAxe
	toolPick
	toolHoe
	toolHammer
	toolBasket
)

type kit struct {
	tunic, legs, sleeve, boots color.RGBA
	tabard, quiver, shield     bool
	female, robe, elite        bool
	reach, bowScale            float64
	head, weapon, tool         int
	bulk                       float64
}

// VillagerTask selects the tool and work animation of a villager sheet.
type VillagerTask int

const (
	TaskIdle VillagerTask = iota
	TaskWood
	TaskMine
	TaskFarm
	TaskForage
	TaskBuild
	TaskHunt
	NumVillagerTasks
)

var taskNames = [NumVillagerTasks]string{"idle", "wood", "mine", "farm", "forage", "build", "hunt"}

func (t VillagerTask) String() string { return taskNames[t] }

func villagerKit(team color.RGBA, female bool, task VillagerTask) kit {
	cloth := Mix(team, RGB(205, 195, 170), 0.2)
	kt := kit{tunic: cloth, legs: RGB(88, 66, 44), sleeve: Shade(cloth, 0.85), boots: RGB(52, 36, 24), head: headHair, weapon: wNone}
	if female {
		dress := Mix(team, RGB(220, 200, 170), 0.12)
		kt.female, kt.tunic, kt.sleeve, kt.legs = true, dress, Shade(dress, 0.88), RGB(120, 90, 70)
	}
	switch task {
	case TaskWood:
		kt.tool = toolAxe
	case TaskMine:
		kt.tool = toolPick
	case TaskFarm:
		kt.tool = toolHoe
	case TaskForage:
		kt.tool = toolBasket
	case TaskBuild:
		kt.tool = toolHammer
	case TaskHunt:
		kt.weapon = wSpear
	}
	return kt
}

// VillagerSheet renders a villager of one gender doing one task, in the
// same layout as UnitSheet.
func VillagerSheet(team color.RGBA, female bool, task VillagerTask) *Canvas {
	kt := villagerKit(team, female, task)
	return sheetOf(func(k *rig, a anim) { drawPerson(k, kt, team, a) })
}

func sheetOf(draw func(k *rig, a anim)) *Canvas {
	sheet := New(UnitFW*UnitFrames, UnitFH*8)
	for d := range 8 {
		for fr := range UnitFrames {
			fc := New(UnitFW, UnitFH)
			draw(newRig(fc, V{UnitAX, UnitAY}, d), animFor(fr))
			finish(fc)
			sheet.Blit(fc, fr*UnitFW, d*UnitFH)
		}
	}
	return sheet
}

func drawPerson(k *rig, kt kit, team color.RGBA, a anim) {
	if a.dead {
		k.c.Ellipse(k.o, 9, 3, softShadow)
		deadPerson(k, kt, team, V{})
		return
	}
	k.c.Ellipse(k.o, 7, 3, softShadow)
	person(k, kt, team, a, 0, false)
	k.flush()
}

var skin = RGB(228, 182, 140)

func kitFor(u Unit, team color.RGBA) kit {
	boots := RGB(52, 36, 24)
	plate := RGB(168, 172, 184)
	mail := RGB(138, 140, 150)
	switch u {
	case Militia:
		return kit{tunic: RGB(150, 152, 162), legs: RGB(70, 70, 82), sleeve: RGB(130, 132, 142), boots: boots, tabard: true, shield: true, head: headHelm, weapon: wSword}
	case Spearman:
		t := Mix(team, RGB(150, 110, 70), 0.3)
		return kit{tunic: t, legs: RGB(96, 72, 48), sleeve: RGB(140, 104, 66), boots: boots, head: headCap, weapon: wSpear}
	case Archer:
		return kit{tunic: Mix(team, RGB(90, 120, 60), 0.15), legs: RGB(80, 70, 50), sleeve: Shade(team, 0.85), boots: boots, quiver: true, head: headHood, weapon: wBow}
	case Champion:
		return kit{tunic: plate, legs: Shade(plate, 0.8), sleeve: plate, boots: RGB(90, 92, 100), tabard: true, head: headGreat, weapon: wGreat, bulk: 1}
	case Scout:
		return kit{tunic: Mix(team, RGB(200, 190, 160), 0.15), legs: RGB(100, 76, 50), sleeve: RGB(150, 112, 70), boots: boots, head: headLeather, weapon: wSword}
	case Knight:
		return kit{tunic: plate, legs: Shade(plate, 0.8), sleeve: plate, boots: RGB(90, 92, 100), tabard: true, shield: true, head: headGreat, weapon: wLance}
	case Cavalier:
		return kit{tunic: plate, legs: Shade(plate, 0.8), sleeve: plate, boots: RGB(90, 92, 100), tabard: true, shield: true, elite: true, head: headGreat, weapon: wLance}
	case ManAtArms:
		return kit{tunic: mail, legs: RGB(70, 70, 82), sleeve: mail, boots: boots, tabard: true, shield: true, head: headCap, weapon: wSword}
	case Longswordsman:
		return kit{tunic: plate, legs: Shade(plate, 0.75), sleeve: mail, boots: RGB(90, 92, 100), tabard: true, shield: true, head: headGreat, weapon: wSword, bulk: 0.5}
	case Pikeman:
		return kit{tunic: mail, legs: RGB(80, 70, 60), sleeve: Shade(team, 0.85), boots: boots, tabard: true, head: headCap, weapon: wSpear, reach: 1.35}
	case Crossbowman:
		padded := Mix(team, RGB(220, 205, 170), 0.35)
		return kit{tunic: padded, legs: RGB(90, 74, 54), sleeve: Shade(padded, 0.88), boots: boots, quiver: true, head: headCap, weapon: wXbow}
	case Arbalest:
		return kit{tunic: mail, legs: RGB(80, 70, 60), sleeve: mail, boots: boots, tabard: true, quiver: true, elite: true, head: headCap, weapon: wXbow, bulk: 0.4}
	case LightCavalry:
		return kit{tunic: mail, legs: RGB(100, 76, 50), sleeve: mail, boots: boots, tabard: true, head: headCap, weapon: wSword}
	case CavalryArcher:
		return kit{tunic: Mix(team, RGB(200, 190, 160), 0.15), legs: RGB(100, 76, 50), sleeve: RGB(150, 112, 70), boots: boots, quiver: true, head: headLeather, weapon: wBow}
	case Longbowman:
		return kit{tunic: Mix(team, RGB(80, 120, 60), 0.25), legs: RGB(80, 70, 50), sleeve: Shade(team, 0.85), boots: boots, quiver: true, head: headHood, weapon: wBow, bowScale: 1.4}
	case Monk:
		robe := RGB(150, 112, 72)
		return kit{tunic: robe, legs: Shade(robe, 0.8), sleeve: Shade(robe, 0.92), boots: RGB(90, 64, 40), robe: true, head: headTonsure, weapon: wStaff}
	}
	return villagerKit(team, false, TaskBuild)
}

// UnitSheet renders the 8 directions x UnitFrames frames of one unit. The
// Villager sheet is the male builder.
func UnitSheet(u Unit, team color.RGBA) *Canvas {
	return sheetOf(func(k *rig, a anim) { drawUnit(k, u, a, team) })
}

func drawUnit(k *rig, u Unit, a anim, team color.RGBA) {
	switch u {
	case Deer:
		deer(k, a, V{})
	case Boar:
		boar(k, a, V{})
	case Scout, Knight, LightCavalry, Cavalier, CavalryArcher:
		kt := kitFor(u, team)
		coat, mane := RGB(150, 98, 56), RGB(70, 45, 28)
		var capar *color.RGBA
		switch u {
		case Knight:
			coat, mane = RGB(222, 216, 206), RGB(150, 140, 130)
			capar = &team
		case Cavalier:
			coat, mane = RGB(70, 64, 62), RGB(36, 32, 30)
			capar = &team
		case LightCavalry:
			coat, mane = RGB(196, 160, 110), RGB(240, 232, 214)
		case CavalryArcher:
			coat, mane = RGB(110, 76, 50), RGB(40, 30, 24)
		}
		if a.dead {
			k.c.Ellipse(k.o, 13, 4, softShadow)
			deadHorse(k, coat, mane, capar, k.r.Mul(-2))
			deadPerson(k, kt, team, k.r.Mul(-11).Add(k.f.Mul(-2)))
			return
		}
		k.c.Ellipse(k.o, 13, 4.5, softShadow)
		horse(k, a, coat, mane, team, capar, k.f.Mul(1))
		person(k, kt, team, a, 7, true)
		k.flush()
	default:
		drawPerson(k, kitFor(u, team), team, a)
	}
}

func person(k *rig, kt kit, team color.RGBA, a anim, lift float64, mounted bool) {
	f, r, c := k.f, k.r, k.c
	P := func(g V, h float64) V { return k.P(g, h+lift) }
	swing, act := a.swing, a.act
	bob := 0.0
	if !mounted {
		bob = 0.6 * math.Abs(swing)
	}
	switch {
	case kt.tool == toolBasket && act >= 0:
		bob = -4.5
	case kt.tool == toolHoe && act == 1:
		bob = -1
	}
	B := func(g V, h float64) V { return P(g, h+bob) }

	for _, side := range []float64{-1, 1} {
		hip := r.Mul(2.3 * side)
		if mounted {
			knee := r.Mul(4.6 * side).Add(f.Mul(1.8))
			foot := r.Mul(5 * side).Add(f.Mul(0.6))
			k.add(r.Mul(5*side).Y+0.2, func() {
				c.Line(P(hip, 13), P(knee, 10), 3, kt.legs)
				c.Line(P(knee, 10), P(foot, 5), 2.6, kt.legs)
				c.Ellipse(P(foot, 4.5), 1.8, 1.3, solid(kt.boots))
			})
			continue
		}
		foot := hip.Add(f.Mul(3 * swing * side))
		if act >= 0 {
			foot = hip.Add(f.Mul(1.5 * side))
		}
		if kt.weapon == wSpear && act >= 0 {
			foot = hip.Add(f.Mul(2.5 * side))
		}
		if kt.tool == toolBasket && act >= 0 {
			foot = hip.Add(f.Mul(side*2 + 1))
		}
		d := foot.Y
		if kt.female || kt.robe {
			d = -100
		}
		k.add(d, func() {
			c.Line(B(hip, 13), P(foot, 2), 3, kt.legs)
			c.Ellipse(P(foot, 1), 2, 1.3, solid(kt.boots))
		})
	}

	if kt.female || kt.robe {
		k.add(-0.02, func() {
			sway := f.Mul(0.9 * swing)
			w0, w1 := 4.6, 6.6
			if kt.tool == toolBasket && act >= 0 {
				w1 = 7.6
			}
			wc, hc := B(V{}, 13.5), P(sway, 3)
			a0, b0 := wc.Add(V{-w0, 0}), wc.Add(V{w0, 0})
			a1, b1 := hc.Add(V{-w1, 0}), hc.Add(V{w1, 0})
			shadeAt := func(x float64) color.RGBA {
				t := (x - a1.X) / math.Max(1, b1.X-a1.X)
				return Shade(kt.tunic, 1.12-0.4*t)
			}
			fill := func(u, v float64, p0, e1, e2 V) color.RGBA { return shadeAt(p0.X + u*e1.X + v*e2.X) }
			c.Tri(a0, b0.Sub(a0), a1.Sub(a0), func(u, v float64) color.RGBA { return fill(u, v, a0, b0.Sub(a0), a1.Sub(a0)) })
			c.Tri(b1, a1.Sub(b1), b0.Sub(b1), func(u, v float64) color.RGBA { return fill(u, v, b1, a1.Sub(b1), b0.Sub(b1)) })
			c.Line(a1, b1, 1.2, Shade(kt.tunic, 0.7))
			switch {
			case kt.robe:
				sd := 3.2
				if f.Y < -0.2 {
					sd = -3.2
				}
				ap := f.Mul(sd)
				c.Capsule(B(ap, 13), P(ap.Add(sway), 4), 1.5, lit(team))
				c.Line(B(r.Mul(-4.5), 13.5), B(r.Mul(4.5), 13.5), 1.4, RGB(220, 200, 150))
			case f.Y > -0.35:
				ap := f.Mul(3.2)
				c.Capsule(B(ap, 12), P(ap.Add(sway), 5.5), 2.4, lit(RGB(238, 230, 210)))
			}
		})
	}

	if kt.quiver {
		q := f.Mul(-3.2).Add(r.Mul(-1))
		k.add(q.Y-0.01, func() {
			c.Line(B(q.Add(r.Mul(1.5)), 25), B(q.Add(r.Mul(-1)), 14), 3, RGB(110, 70, 40))
			for i := range 3 {
				p := B(q.Add(r.Mul(1.2+0.6*float64(i))), 26.5)
				c.Blend(int(p.X), int(p.Y), RGB(240, 236, 220))
				c.Blend(int(p.X), int(p.Y)-1, Shade(team, 1.1))
			}
		})
	}

	bw := 5.5 + kt.bulk
	if kt.female {
		bw = 4.8
	}
	k.add(0, func() {
		c.Ellipse(B(V{}, 19), bw, 6.5+kt.bulk*0.5, lit(kt.tunic))
		if kt.female && f.Y > -0.35 {
			c.Ellipse(B(f.Mul(2.4), 17.5), 2.4, 3.2, lit(RGB(238, 230, 210)))
		}
		if kt.robe {
			sd := 1.0
			if f.Y < -0.2 {
				sd = -1
			}
			c.Capsule(B(f.Mul(2.6*sd).Add(r.Mul(-1.4)), 24), B(f.Mul(2.6*sd).Add(r.Mul(-1.4)), 14), 1.2, lit(team))
			c.Capsule(B(f.Mul(2.6*sd).Add(r.Mul(1.4)), 24), B(f.Mul(2.6*sd).Add(r.Mul(1.4)), 14), 1.2, lit(team))
			c.Ellipse(B(V{}, 24.5), 4.2, 2, lit(Shade(kt.tunic, 0.9)))
		}
		if kt.tabard {
			sd := 1.0
			if f.Y < -0.2 {
				sd = -1
			}
			c.Ellipse(B(f.Mul(2.2*sd), 18.5), 3+kt.bulk*0.3, 5.2, lit(team))
		} else if kt.head == headHood || kt.weapon == wNone || mounted {
			c.Ellipse(B(V{}, 21), bw-1.5, 3, lit(Shade(kt.tunic, 1.02)))
		}
		if kt.weapon == wSpear {
			c.Line(B(r.Mul(-4).Add(f.Mul(1)), 24), B(r.Mul(4).Add(f.Mul(1)), 15), 1.5, RGB(110, 76, 44))
		}
		c.Line(B(r.Mul(-bw+1), 13.5), B(r.Mul(bw-1), 13.5), 1.5, RGB(70, 45, 25))
	})

	two := kt.weapon == wGreat
	var grip V
	var gripH float64
	if two {
		switch act {
		case 0:
			grip, gripH = r.Mul(2.5).Add(f.Mul(-1.5)), 28
		case 1:
			grip, gripH = f.Mul(6), 15
		default:
			grip, gripH = f.Mul(4), 15
		}
	}

	tp := poseTool(kt.tool, act, swing, f, r, r.Mul(bw-0.9))
	if kt.tool == toolBasket {
		bp, bh, bd := tp.lhand.Add(f.Mul(0.5)), tp.lh-3, tp.lhand.Y+0.03
		if act >= 0 {
			bp, bh, bd = f.Mul(3.5).Add(r.Mul(-4.5)), 0.5, f.Mul(3.5).Add(r.Mul(-4.5)).Y
		}
		k.add(bd, func() { basket(c, P(bp, bh)) })
	}

	for _, side := range []float64{-1, 1} {
		sh := r.Mul((bw - 0.9) * side)
		hand, hh := sh.Add(f.Mul(-2.5*swing*side)), 13.0
		if mounted {
			hand, hh = sh.Add(f.Mul(2)), 16
		}
		right := side > 0
		switch kt.weapon {
		case wTool, wSword:
			if right && act == 0 {
				hand, hh = sh.Add(f.Mul(-1)), 29
			} else if right && act == 1 {
				hand, hh = sh.Add(f.Mul(6)), 16
			} else if !right && kt.shield {
				hand, hh = sh.Add(f.Mul(2.5)), 15
			}
		case wLance:
			if right {
				switch act {
				case 0:
					hand, hh = sh.Add(f.Mul(-2)), 17
				case 1:
					hand, hh = sh.Add(f.Mul(4)), 17
				default:
					hand, hh = sh.Add(f.Mul(1)), 15
				}
			} else {
				hand, hh = sh.Add(f.Mul(2.5)), 16
			}
		case wSpear:
			switch {
			case right && act == 0:
				hand, hh = sh.Add(f.Mul(-2.5)), 19
			case right && act == 1:
				hand, hh = sh.Add(f.Mul(5)), 18
			case right:
				hand, hh = sh.Add(f.Mul(1)), 14
			case act >= 0:
				hand, hh = sh.Mul(0.3).Add(f.Mul(4+float64(act)*3)), 18
			}
		case wBow:
			switch {
			case !right && act >= 0:
				hand, hh = sh.Mul(0.6).Add(f.Mul(7)), 22
			case !right:
				hand, hh = sh.Mul(1.2).Add(f.Mul(1.5)), 15
			case act == 0:
				hand, hh = r.Mul(1.5).Add(f.Mul(0.3)), 22
			case act == 1:
				hand, hh = r.Mul(3.5).Add(f.Mul(-2)), 23
			}
		case wGreat:
			hand, hh = grip.Add(r.Mul(0.7*side)), gripH+side*0.8
		case wXbow:
			switch {
			case right && act >= 0:
				hand, hh = r.Mul(1.8).Add(f.Mul(0.5-0.6*float64(act))), 22
			case right:
				hand, hh = sh.Add(f.Mul(1.5)), 15
			case act >= 0:
				hand, hh = r.Mul(-0.3).Add(f.Mul(5)), 22
			default:
				hand, hh = sh.Mul(0.4).Add(f.Mul(4.5)), 16
			}
		case wStaff:
			switch act {
			case 0:
				hand, hh = sh.Mul(0.9).Add(f.Mul(3)), 28
			case 1:
				hand, hh = sh.Mul(1.35).Add(f.Mul(2)), 31
			default:
				if right {
					hand, hh = sh.Add(f.Mul(1.5-1.2*swing)), 15
				}
			}
		}
		switch {
		case kt.tool == toolNone:
		case right && tp.ok:
			hand, hh = tp.grip, tp.gh
		case !right && tp.two:
			hand, hh = tp.grip.Add(tp.tip.Sub(tp.grip).Mul(0.4)), tp.gh+(tp.th-tp.gh)*0.4
		case !right && tp.left:
			hand, hh = tp.lhand, tp.lh
		}
		hand, hh, sh, side := hand, hh, sh, side
		depth := hand.Y + 0.01*side
		if right && (kt.weapon == wSpear || kt.weapon == wLance || kt.weapon == wGreat || (tp.ok && kt.tool != toolBasket)) {
			depth = math.Max(depth, 0.5)
			if f.Y < -0.3 {
				depth = math.Min(hand.Y, -0.5)
			}
		}
		k.add(depth, func() {
			c.Line(B(sh, 22), P(hand, hh), 2.6, kt.sleeve)
			h := P(hand, hh)
			switch {
			case right && tp.ok && kt.tool != toolBasket:
				d := tp.tip.Sub(tp.grip)
				drawTool(c, kt.tool, P(tp.grip.Sub(d.Mul(0.25)), tp.gh-(tp.th-tp.gh)*0.25), P(tp.tip, tp.th), V{f.X, f.Y * 0.5})
			case right && (kt.weapon == wSword || (kt.weapon == wTool && act >= 0)):
				blade := kt.weapon == wSword
				tip, th := hand.Add(f.Mul(4)), hh+8
				if blade && mounted && act < 0 {
					tip, th = hand.Add(f.Mul(3)), hh+9
				}
				switch act {
				case 0:
					tip, th = hand.Add(f.Mul(-3)), hh+10
				case 1:
					tip, th = hand.Add(f.Mul(10)), hh-3
					if !blade {
						tip, th = hand.Add(f.Mul(5)), hh-7
					}
				}
				if blade {
					c.Line(h, P(tip, th), 2, RGB(205, 210, 220))
					c.Line(P(hand.Add(r.Mul(-1.5)), hh+1), P(hand.Add(r.Mul(1.5)), hh+1), 1.5, RGB(90, 70, 40))
				} else {
					c.Line(h, P(tip, th), 1.5, RGB(120, 85, 50))
					c.Ellipse(P(tip, th), 2, 2, lit(RGB(160, 160, 170)))
				}
			case right && kt.weapon == wXbow:
				crossbow(c, P, hand, hh, f, r, act, kt.elite)
			case right && kt.weapon == wStaff:
				top := hh + 22
				if act >= 0 {
					top = hh + 10
				}
				c.Line(P(hand.Add(f.Mul(-0.3)), hh-13+float64(act+1)*3), P(hand.Add(f.Mul(0.6)), top), 2.2, RGB(206, 166, 104))
				c.Ellipse(P(hand.Add(f.Mul(0.6)), top), 2, 2, lit(RGB(226, 186, 110)))
			case right && kt.weapon == wSpear:
				reach := math.Max(1, kt.reach)
				butt, bh, tip, tz := hand.Add(f.Mul(-1)), hh-9, hand.Add(f.Mul(2+5*(reach-1)/0.35)).Add(r.Mul(1)), hh+27+2*(reach-1)/0.35
				switch act {
				case 0:
					butt, bh, tip, tz = hand.Add(f.Mul(-7)), hh+1, hand.Add(f.Mul(10*reach)), hh+3
				case 1:
					butt, bh, tip, tz = hand.Add(f.Mul(-4)), hh+2, hand.Add(f.Mul(14*reach)), hh-1
				}
				c.Line(P(butt, bh), P(tip, tz), 1.5, RGB(130, 92, 52))
				d := tip.Sub(butt)
				t0 := 1 - 3.5/math.Hypot(math.Hypot(d.X, d.Y), tz-bh)
				c.Line(P(butt.Add(d.Mul(t0)), bh+(tz-bh)*t0), P(tip, tz), 2.4, RGB(215, 218, 228))
			case right && kt.weapon == wLance:
				butt, bh, tip, tz := hand.Add(f.Mul(-3)), hh-3, hand.Add(f.Mul(9)), hh+15
				switch act {
				case 0:
					butt, bh, tip, tz = hand.Add(f.Mul(-5)), hh+1, hand.Add(f.Mul(12)), hh+3
				case 1:
					butt, bh, tip, tz = hand.Add(f.Mul(-3)), hh, hand.Add(f.Mul(17)), hh-1
				}
				c.Line(P(butt, bh), P(tip, tz), 2, RGB(150, 110, 70))
				c.Ellipse(P(tip, tz), 1.2, 1.2, solid(RGB(220, 222, 230)))
				pa := butt.Add(tip.Sub(butt).Mul(0.8))
				ph := bh + (tz-bh)*0.8
				pb := butt.Add(tip.Sub(butt).Mul(0.62))
				pbh := bh + (tz-bh)*0.62
				q0, q1 := P(pa, ph), P(pb, pbh)
				c.Tri(q0, q1.Sub(q0), V{-2 * f.X, 5}, solid(team))
			case right && kt.weapon == wBow && act >= 0:
				// The bow is drawn with the left hand.
			case !right && kt.weapon == wBow:
				bow(k, P, hand, hh, act, r.Mul(1.5).Add(f.Mul(0.3)), kt.bowScale)
			case !right && kt.shield:
				p := P(hand.Add(f.Mul(1.2)), hh+1)
				c.Ellipse(p, 4.3, 5.3, lit(RGB(150, 150, 155)))
				c.Ellipse(p, 3.4, 4.4, lit(team))
				c.Line(p.Add(V{0, -3}), p.Add(V{0, 3}), 1, Shade(Mix(team, RGB(255, 230, 120), 0.6), 1))
			case right && two:
				tip, th := grip.Add(f.Mul(2.5)), gripH+17
				switch act {
				case 0:
					tip, th = grip.Add(f.Mul(-7)).Add(r.Mul(3)), gripH+7
				case 1:
					tip, th = grip.Add(f.Mul(14)), 1
				}
				base := grip.Add(tip.Sub(grip).Mul(0.15))
				bh := gripH + (th-gripH)*0.15
				c.Line(P(base, bh), P(tip, th), 2.6, RGB(215, 220, 232))
				c.Line(P(base, bh), P(tip, th), 0.8, RGB(250, 250, 255))
				c.Line(P(base.Add(r.Mul(-2.5)), bh), P(base.Add(r.Mul(2.5)), bh), 1.6, goldCol)
				c.Line(P(grip.Sub(tip.Sub(grip).Mul(0.12)), gripH-(th-gripH)*0.12), P(base, bh), 1.6, RGB(90, 60, 35))
			}
			c.Ellipse(h, 1.6, 1.6, solid(skin))
			if kt.head == headGreat {
				c.Ellipse(h, 1.7, 1.7, lit(RGB(140, 142, 150)))
			}
		})
	}

	k.add(50, func() { head(k, kt, team, B, f, r) })
	if kt.weapon == wStaff && act >= 0 {
		k.add(60, func() {
			ctr := B(f.Mul(2.5), 27+float64(act)*2)
			rad := 9.0 + 3*float64(act)
			c.Ellipse(ctr, rad, rad*0.9, func(nx, ny float64) color.RGBA {
				d := 1 - (nx*nx + ny*ny)
				return color.RGBA{255, 240, 150, uint8(160 * d * d)}
			})
			for i := range 4 {
				a := float64(i)*math.Pi/2 + float64(act)*0.6
				p := ctr.Add(V{math.Cos(a) * rad * 0.65, math.Sin(a) * rad * 0.55})
				c.Blend(int(p.X), int(p.Y), RGB(255, 252, 220))
			}
		})
	}
}

// bow draws a bow held at hand; while drawing, the string goes to pull.
func bow(k *rig, P func(V, float64) V, hand V, hh float64, act int, pull V, scale float64) {
	c, f := k.c, k.f
	if scale == 0 {
		scale = 1
	}
	wood := RGB(176, 122, 58)
	var prev V
	top, bot := V{}, V{}
	for i := 0; i <= 10; i++ {
		t := float64(i)/5 - 1
		p := P(hand.Add(f.Mul(-3.2*scale*t*t)), hh+10*scale*t)
		if i > 0 {
			c.Line(prev, p, 2, wood)
		}
		if i == 0 {
			bot = p
		}
		top = p
		prev = p
	}
	str := RGB(230, 225, 210)
	if act == 0 {
		ph := P(pull, 22)
		c.Line(top, ph, 0.7, str)
		c.Line(bot, ph, 0.7, str)
		c.Line(ph, P(hand.Add(f.Mul(3)), hh+0.5), 1, RGB(150, 110, 70))
		c.Ellipse(P(hand.Add(f.Mul(3)), hh+0.5), 0.9, 0.9, solid(RGB(200, 200, 210)))
		return
	}
	c.Line(top, bot, 0.7, str)
}

func head(k *rig, kt kit, team color.RGBA, P func(V, float64) V, f, r V) {
	c := k.c
	eyes := func(h float64) {
		if f.Y > -0.3 {
			for _, side := range []float64{-1, 1} {
				e := f.Mul(3).Add(r.Mul(1.6 * side))
				if e.Y > -1 {
					p := P(e, h)
					c.Blend(int(p.X), int(p.Y), RGB(40, 30, 30))
				}
			}
		}
	}
	switch kt.head {
	case headGreat:
		c.Ellipse(P(V{}, 28.5), 4.2, 4.6, lit(RGB(176, 180, 192)))
		if f.Y > -0.4 {
			c.Line(P(f.Mul(3.4).Add(r.Mul(-2)), 28.8), P(f.Mul(3.4).Add(r.Mul(2)), 28.8), 1, RGB(30, 30, 36))
			c.Line(P(f.Mul(3.8), 28), P(f.Mul(3.8), 25.5), 0.8, RGB(120, 122, 130))
		}
		c.Ellipse(P(f.Mul(-1.2), 32.3), 1.8, 2.1, lit(team))
		c.Line(P(f.Mul(-1.2), 33), P(f.Mul(-3.5), 31), 1.6, team)
		if kt.elite {
			c.Line(P(r.Mul(-3.8), 29.5), P(r.Mul(3.8), 29.5), 1.2, goldCol)
			c.Ellipse(P(f.Mul(-2), 32.5), 2.2, 2.2, lit(Shade(team, 1.1)))
		}
		return
	case headHood:
		hood := Shade(team, 0.8)
		c.Line(P(f.Mul(-2.5), 29), P(f.Mul(-4.5), 25), 2, hood)
		c.Ellipse(P(V{}, 28.3), 4.5, 4.5, lit(hood))
		if f.Y > -0.5 {
			c.Ellipse(P(f.Mul(1.8), 27.8), 2.8, 3, lit(skin))
			eyes(28)
		}
		return
	}
	c.Ellipse(P(V{}, 28), 4, 4, lit(skin))
	eyes(28)
	switch kt.head {
	case headHelm:
		c.Ellipse(P(f.Mul(-0.4), 29.8), 4.4, 3.3, lit(RGB(135, 138, 148)))
		c.Line(P(f.Mul(3.6), 30), P(f.Mul(3.8), 26.5), 1, RGB(100, 100, 110))
	case headCap:
		c.Ellipse(P(f.Mul(-0.3), 30), 4.3, 3.1, lit(RGB(150, 154, 166)))
		c.Line(P(V{}, 31.5), P(V{}, 34.5), 1.8, RGB(150, 154, 166))
		c.Line(P(f.Mul(3.7), 30), P(f.Mul(3.9), 26.8), 1, RGB(110, 112, 122))
	case headLeather:
		c.Ellipse(P(f.Mul(-0.6), 29.8), 4.2, 3, lit(RGB(130, 92, 56)))
		c.Line(P(f.Mul(-0.6).Add(r.Mul(-4)), 29), P(f.Mul(-0.6).Add(r.Mul(4)), 29), 1, team)
	case headTonsure:
		c.Ellipse(P(f.Mul(-0.6), 29), 4.2, 2.8, lit(RGB(96, 66, 40)))
		c.Ellipse(P(f.Mul(-0.9), 30.8), 2.6, 1.6, lit(skin))
	default:
		if kt.female {
			hair := RGB(150, 88, 42)
			c.Ellipse(P(f.Mul(-1.1), 29.4), 4.3, 3.3, lit(hair))
			c.Ellipse(P(f.Mul(-3.4), 29.5), 2.3, 2.3, lit(Shade(hair, 0.95)))
			c.Line(P(f.Mul(1.2).Add(r.Mul(-3.6)), 30.5), P(f.Mul(1.2).Add(r.Mul(3.6)), 30.5), 1.2, RGB(238, 230, 210))
			return
		}
		c.Ellipse(P(f.Mul(-1.3), 29.6), 3.9, 2.9, lit(RGB(110, 70, 40)))
	}
}

// deadPerson lies on its back with the head at +b.
func deadPerson(k *rig, kt kit, team color.RGBA, off V) {
	c := k.c
	b := k.r.Sub(k.f).Mul(1 / math.Sqrt2)
	sd := V{-b.Y, b.X}
	at := func(t, s float64) V { return off.Add(b.Mul(t)).Add(sd.Mul(s)) }
	P := k.P
	switch kt.weapon {
	case wSword, wTool:
		c.Line(P(at(-2, 6), 1), P(at(6, 7.5), 1), 1.8, RGB(205, 210, 220))
	case wSpear, wLance:
		c.Line(P(at(-9, 6), 1), P(at(10, 7), 1), 1.5, RGB(130, 92, 52))
		c.Line(P(at(8, 6.9), 1), P(at(10.5, 7), 1), 2.2, RGB(215, 218, 228))
	case wBow, wXbow:
		c.Line(P(at(-3, 7), 1), P(at(5, 8), 1), 1.6, RGB(130, 84, 40))
	case wStaff:
		c.Line(P(at(-6, 6), 1), P(at(10, 7), 1), 2.2, RGB(206, 166, 104))
	case wGreat:
		c.Line(P(at(-6, 6.5), 1), P(at(9, 7.5), 1), 2.4, RGB(215, 220, 232))
		c.Line(P(at(-3, 5), 1), P(at(-3, 8), 1), 1.5, goldCol)
	}
	switch kt.tool {
	case toolNone:
	case toolBasket:
		basket(c, P(at(1, -9), 0.5))
	default:
		drawTool(c, kt.tool, P(at(-4, 6.5), 1), P(at(7, 8), 1), V{0, 1})
	}
	if kt.shield {
		c.Ellipse(P(at(1, -8), 1), 4.4, 2.8, lit(RGB(150, 150, 155)))
		c.Ellipse(P(at(1, -8), 1.2), 3.5, 2.1, lit(team))
	}
	for _, s := range []float64{-1, 1} {
		c.Line(P(at(-6, 1.6*s), 1.5), P(at(-1, 1.8*s), 2), 3, kt.legs)
		c.Ellipse(P(at(-6.8, 1.7*s), 1.5), 1.6, 1.6, solid(kt.boots))
	}
	if kt.female || kt.robe {
		c.Capsule(P(at(-5.5, 0), 2), P(at(-0.5, 0), 2.5), 4.6, lit(kt.tunic))
	}
	c.Capsule(P(at(-1, 0), 2.5), P(at(3.5, 0), 2.5), 4.2+kt.bulk*0.5, lit(kt.tunic))
	if kt.tabard {
		c.Capsule(P(at(-0.5, 0), 3.5), P(at(3, 0), 3.5), 2.2, lit(team))
	}
	c.Line(P(at(3.5, 4), 2), P(at(1, 7.5), 1.5), 2.6, kt.sleeve)
	c.Line(P(at(3.5, -4), 2), P(at(5.5, -7), 1.5), 2.6, kt.sleeve)
	c.Ellipse(P(at(1, 7.5), 1.5), 1.5, 1.5, solid(skin))
	c.Ellipse(P(at(5.5, -7), 1.5), 1.5, 1.5, solid(skin))
	hp := P(at(7.3, 0), 2.5)
	switch kt.head {
	case headGreat:
		c.Ellipse(hp, 3.8, 3.6, lit(RGB(176, 180, 192)))
		c.Ellipse(P(at(9.5, 1), 1.5), 1.8, 1.5, lit(team))
	case headHood:
		c.Ellipse(hp, 3.9, 3.7, lit(Shade(team, 0.8)))
		c.Ellipse(hp.Add(V{0, 0.5}), 2.2, 2.2, lit(skin))
	default:
		c.Ellipse(hp, 3.5, 3.4, lit(skin))
		switch kt.head {
		case headHelm, headCap:
			c.Ellipse(P(at(8.6, 0), 2.5), 3.2, 2.6, lit(RGB(140, 142, 152)))
		case headLeather:
			c.Ellipse(P(at(8.6, 0), 2.5), 3, 2.5, lit(RGB(130, 92, 56)))
		default:
			c.Ellipse(P(at(8.6, 0), 2.5), 3, 2.5, lit(RGB(110, 70, 40)))
		}
	}
}

// gait returns the forward foot offset of each leg for a four-legged walk.
func gait(a anim, stride float64) (fl, fr, bl, br float64) {
	if a.frame < 1 || a.frame > 4 {
		return 0, 0, 0, 0
	}
	s := math.Sin(float64(a.frame-1)*math.Pi/2 + math.Pi/4)
	c := math.Cos(float64(a.frame-1)*math.Pi/2 + math.Pi/4)
	return stride * s, -stride * s, -stride * c, stride * c
}

type legSpec struct {
	along, side, swing float64
}

func (k *rig) legs(off V, specs []legSpec, hipH, w float64, col, hoof color.RGBA, nearBias float64, under bool) {
	for _, l := range specs {
		hip := off.Add(k.at(l.along, l.side))
		foot := off.Add(k.at(l.along+l.swing, l.side*1.1))
		lift := math.Max(0, math.Abs(l.swing)*0.6-0.5)
		knee := hip.Add(foot).Mul(0.5).Add(k.f.Mul(0.6))
		d := hip.Y + nearBias
		if k.r.Mul(l.side).Y <= 0.05 {
			d = hip.Y - 20
		}
		if under {
			d = -100
		}
		k.add(d, func() {
			k.limb(hip, hipH, knee, hipH*0.5+lift, w, col)
			k.limb(knee, hipH*0.5+lift, foot, 1+lift, w*0.85, col)
			k.c.Ellipse(k.P(foot, 0.8+lift), w*0.6, w*0.45, solid(hoof))
		})
	}
}

func horse(k *rig, a anim, coat, mane, saddle color.RGBA, capar *color.RGBA, off V) {
	f := k.f
	fl, fr, bl, br := gait(a, 3.5)
	k.legs(off, []legSpec{{5.5, -1.8, fl}, {5.5, 1.8, fr}, {-6, -1.8, bl}, {-6, 1.8, br}}, 14, 2.1, Shade(coat, 0.9), RGB(50, 40, 32), 0, capar != nil)
	g := func(x float64) V { return off.Add(f.Mul(x)) }
	k.add(g(-8).Y-0.02, func() {
		k.capsule(g(-7.5), 17.5, g(-10), 9, 1.4, mane)
	})
	k.add(0, func() {
		k.capsule(g(-6), 15, g(6.5), 15.5, 4.6, coat)
		if capar != nil {
			team := *capar
			k.c.Capsule(k.P(g(-6.8), 14.5), k.P(g(7.2), 15), 5, lit(team))
			k.c.Capsule(k.P(g(-6.8), 11), k.P(g(7.2), 11), 3.3, lit(Shade(team, 0.85)))
			k.limb(g(-6.8).Add(k.r.Mul(2.2)), 8, g(7.2).Add(k.r.Mul(2.2)), 8, 1.2, goldCol)
			k.c.Capsule(k.P(g(-1.5), 19.5), k.P(g(1.5), 19.5), 2.5, lit(RGB(100, 70, 40)))
		} else {
			k.c.Capsule(k.P(g(-2.5), 18.5), k.P(g(2), 18.5), 3.4, lit(saddle))
			k.limb(g(-2.5).Add(k.r.Mul(3.3)), 15.5, g(2).Add(k.r.Mul(3.3)), 15.5, 1.2, goldCol)
			k.c.Capsule(k.P(g(-1.5), 19.8), k.P(g(1.5), 19.8), 2.2, lit(RGB(110, 70, 40)))
		}
	})
	headUp := 0.0
	if a.act == 0 {
		headUp = 2
	}
	k.add(g(8).Y+0.01, func() {
		neckCol := coat
		if capar != nil {
			neckCol = *capar
		}
		k.capsule(g(5.5), 17, g(9.5), 22+headUp, 2.7, neckCol)
		k.limb(g(5), 20, g(9), 25+headUp, 1.6, mane)
	})
	k.add(g(11).Y+0.02, func() {
		hc := coat
		if capar != nil {
			hc = RGB(176, 180, 192)
		}
		k.capsule(g(10), 23.5+headUp, g(13.5), 19.5+headUp, 2.2, hc)
		k.c.Ellipse(k.P(g(13.8), 19+headUp), 1.6, 1.4, lit(Shade(coat, 0.7)))
		k.limb(g(9.6), 25.5+headUp, g(9.2), 28+headUp, 1.2, Shade(coat, 0.8))
		if f.Y > -0.5 {
			for _, s := range []float64{-1, 1} {
				e := k.P(g(11.2).Add(k.r.Mul(1.5*s)), 23+headUp)
				if k.r.Mul(s).Y > -0.6 {
					k.c.Blend(int(e.X), int(e.Y), RGB(25, 20, 18))
				}
			}
		}
	})
}

func deadHorse(k *rig, coat, mane color.RGBA, capar *color.RGBA, off V) {
	f, r := k.f, k.r
	g := func(a, b float64) V { return off.Add(k.at(a, b)) }
	legs := func() {
		for _, a := range []float64{5, -5} {
			for _, d := range []float64{0, 2} {
				k.limb(g(a+d*0.5, 1), 4, g(a+d+1.5, 9), 2, 2.2, Shade(coat, 0.9))
				k.c.Ellipse(k.P(g(a+d+1.5, 9.5), 2), 1.4, 1.1, solid(RGB(50, 40, 32)))
			}
		}
	}
	_ = f
	if r.Y < 0 {
		legs()
	}
	k.capsule(g(-9, -1), 4, g(-12, 1), 1.5, 1.3, mane)
	body := coat
	if capar != nil {
		body = *capar
	}
	k.capsule(g(-6, 0), 5, g(6, 0), 5, 5, body)
	if capar != nil {
		k.limb(g(-6, 4), 2, g(6, 4), 2, 1.2, goldCol)
	}
	k.capsule(g(6, 0), 5, g(10, -1), 3, 2.6, body)
	k.capsule(g(10.5, -1), 3, g(14, -1.5), 2.2, 2.1, coat)
	if r.Y >= 0 {
		legs()
	}
}

func deer(k *rig, a anim, off V) {
	f := k.f
	coat := RGB(168, 116, 66)
	g := func(x float64) V { return off.Add(f.Mul(x)) }
	if a.dead {
		k.c.Ellipse(k.o, 10, 3, softShadow)
		deadBeast(k, off, coat, 3.6, 4.2, 1.5, true)
		return
	}
	k.c.Ellipse(k.o, 10, 3.5, softShadow)
	fl, fr, bl, br := gait(a, 2.5)
	k.legs(off, []legSpec{{3.6, -1.4, fl}, {3.6, 1.4, fr}, {-4, -1.4, bl}, {-4, 1.4, br}}, 10, 1.5, Shade(coat, 0.85), RGB(50, 38, 30), 0, false)
	k.add(g(-5).Y-0.01, func() {
		k.c.Ellipse(k.P(g(-5.2), 13), 1.4, 1.6, lit(RGB(245, 240, 230)))
	})
	k.add(0, func() {
		k.capsule(g(-4.5), 11.5, g(4), 12, 3.6, coat)
		k.c.Capsule(k.P(g(-3.5), 9.5), k.P(g(3), 9.8), 1.6, lit(RGB(225, 205, 175)))
	})
	hx, hy, nx, ny := 8.5, 18.5, 6.0, 19.0
	if a.act >= 0 {
		hx, hy, nx, ny = 7.8, 2.5-float64(a.act), 6.8, 5.0-float64(a.act)
	}
	k.add(g(7).Y+0.01, func() {
		k.capsule(g(3.5), 13, g(nx), ny, 1.8, coat)
		k.capsule(g(nx+0.3), ny+1, g(hx), hy, 1.6, coat)
		k.c.Ellipse(k.P(g(hx+0.3), hy-0.3), 1, 1, solid(RGB(40, 30, 25)))
		ant := RGB(225, 205, 170)
		for _, s := range []float64{-1, 1} {
			b := g(nx + 0.4).Add(k.r.Mul(0.8 * s))
			tipH := ny + 7
			if a.act >= 0 {
				tipH = ny + 4
			}
			t := b.Add(k.r.Mul(2 * s)).Add(f.Mul(-1))
			k.limb(b, ny+2, t, tipH, 1, ant)
			k.limb(b.Add(t.Sub(b).Mul(0.5)), (ny+2+tipH)/2, b.Add(f.Mul(1.5)), tipH-1, 0.9, ant)
		}
	})
	k.flush()
}

func boar(k *rig, a anim, off V) {
	f := k.f
	coat := RGB(92, 70, 54)
	if a.dead {
		k.c.Ellipse(k.o, 9, 3, softShadow)
		deadBeast(k, off, coat, 4.4, 3.5, 1.2, false)
		return
	}
	shift, hd := 0.0, 0.0
	switch a.act {
	case 0:
		shift, hd = -1.5, -1.5
	case 1:
		shift, hd = 2.5, 0.8
	}
	off = off.Add(f.Mul(shift))
	g := func(x float64) V { return off.Add(f.Mul(x)) }
	k.c.Ellipse(k.o.Add(V{f.X * shift, f.Y * shift * 0.5}), 9, 3.5, softShadow)
	fl, fr, bl, br := gait(a, 2)
	if a.act == 1 {
		fl, fr, bl, br = 1.5, 1.5, -2, -2
	}
	k.legs(off, []legSpec{{3, -1.8, fl}, {3, 1.8, fr}, {-3.5, -1.8, bl}, {-3.5, 1.8, br}}, 8.5, 2.2, Shade(coat, 0.8), RGB(40, 32, 26), 0, false)
	k.add(g(-6).Y-0.01, func() { k.limb(g(-5.5), 11.5, g(-7), 8, 1, coat) })
	k.add(0, func() {
		k.capsule(g(-4.5), 10.5, g(2.5), 11, 4.3, coat)
		k.limb(g(-4.5), 14.6, g(3), 15, 1.8, RGB(60, 46, 36))
	})
	k.add(g(6).Y+0.01, func() {
		k.capsule(g(3), 10.5, g(6.5), 8.5+hd, 3.1, Shade(coat, 0.95))
		k.c.Ellipse(k.P(g(7.6), 8+hd), 1.6, 1.5, lit(RGB(170, 130, 120)))
		for _, s := range []float64{-1, 1} {
			b := g(6.8).Add(k.r.Mul(1.2 * s))
			k.limb(b, 7+hd, b.Add(f.Mul(0.8)).Add(k.r.Mul(0.4*s)), 9.8+hd, 1, RGB(240, 236, 220))
			k.limb(g(3.5).Add(k.r.Mul(1.8*s)), 13+hd*0.5, g(3).Add(k.r.Mul(2.2*s)), 15+hd*0.5, 1.4, RGB(70, 52, 40))
		}
		if f.Y > -0.5 {
			e := k.P(g(5.5).Add(k.r.Mul(1.5)), 11+hd)
			k.c.Blend(int(e.X), int(e.Y), RGB(220, 60, 40))
		}
	})
	k.flush()
}

// deadBeast lies on its side with the legs stretched out.
func deadBeast(k *rig, off V, coat color.RGBA, rad, length, legW float64, antlers bool) {
	g := func(a, b float64) V { return off.Add(k.at(a, b)) }
	legs := func() {
		for _, a := range []float64{length * 0.8, -length * 0.8} {
			for _, d := range []float64{0, 1.5} {
				k.limb(g(a+d*0.5, 1), 3, g(a+d+1, rad+4), 1.5, legW, Shade(coat, 0.85))
			}
		}
	}
	if k.r.Y < 0 {
		legs()
	}
	k.capsule(g(-length, 0), rad, g(length, 0), rad, rad, coat)
	k.capsule(g(length, 0), rad, g(length+4, -1), 2, rad*0.55, coat)
	if antlers {
		k.limb(g(length+3.5, -1.5), 2.5, g(length+5, -5), 2, 1, RGB(225, 205, 170))
		k.limb(g(length+3.5, -1.5), 2.5, g(length+2.5, -5.5), 2, 1, RGB(225, 205, 170))
	} else {
		k.limb(g(length+3.6, -1.2), 1.5, g(length+4.6, -0.5), 3, 1, RGB(240, 236, 220))
	}
	if k.r.Y >= 0 {
		legs()
	}
}

// Carcass is a dead animal lying on the ground, as a size-1 resource.
func Carcass(u Unit) *Canvas {
	c := New(64, 48)
	k := newRig(c, V{32, 34}, 1)
	c.Ellipse(V{33, 35}, 13, 4, solid(color.RGBA{120, 20, 20, 110}))
	switch u {
	case Boar:
		deadBeast(k, V{}, RGB(92, 70, 54), 4.4, 3.5, 1.2, false)
	default:
		deadBeast(k, V{}, RGB(168, 116, 66), 3.6, 4.2, 1.5, true)
	}
	finish(c)
	return c
}
