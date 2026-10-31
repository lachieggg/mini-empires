package art

import (
	"image/color"
	"math"
)

// toolPose places a villager tool: the right hand at grip (height gh) and
// the tool head at tip (height th). two means the left hand also holds the
// haft; left means the left hand goes to lhand instead (the basket).
type toolPose struct {
	ok, two, left bool
	grip, tip     V
	lhand         V
	gh, th, lh    float64
}

func poseTool(tool, act int, swing float64, f, r, sh V) toolPose {
	p := toolPose{ok: tool != toolNone}
	carry := func() {
		p.grip, p.gh = sh.Add(f.Mul(1-1.5*swing)), 15
		p.tip, p.th = p.grip.Add(f.Mul(-3.5)).Add(r.Mul(-1)), p.gh+13
	}
	switch tool {
	case toolAxe:
		switch act {
		case 0:
			p.grip, p.gh = r.Mul(3).Add(f.Mul(-1.5)), 26
			p.tip, p.th = p.grip.Add(f.Mul(-4)).Add(r.Mul(3)), p.gh+6
			p.two = true
		case 1:
			p.grip, p.gh = f.Mul(5).Add(r.Mul(-1)), 16
			p.tip, p.th = p.grip.Add(f.Mul(8)).Add(r.Mul(-4)), p.gh+1
			p.two = true
		default:
			carry()
		}
	case toolPick:
		switch act {
		case 0:
			p.grip, p.gh = r.Mul(1).Add(f.Mul(-0.5)), 30
			p.tip, p.th = p.grip.Add(f.Mul(-5)), p.gh+5
			p.two = true
		case 1:
			p.grip, p.gh = f.Mul(5.5), 13
			p.tip, p.th = p.grip.Add(f.Mul(7)), 2
			p.two = true
		default:
			carry()
		}
	case toolHoe:
		switch act {
		case 0:
			p.grip, p.gh = f.Mul(3).Add(r.Mul(1.5)), 20
			p.tip, p.th = p.grip.Add(f.Mul(7)), p.gh+10
			p.two = true
		case 1:
			p.grip, p.gh = f.Mul(4).Add(r.Mul(1)), 13
			p.tip, p.th = p.grip.Add(f.Mul(10)), 1
			p.two = true
		default:
			p.grip, p.gh = sh.Add(f.Mul(1-1.5*swing)), 15
			p.tip, p.th = p.grip.Add(f.Mul(1.5)), p.gh+16
		}
	case toolHammer:
		switch act {
		case 0:
			p.grip, p.gh = sh.Add(f.Mul(-0.5)), 28
			p.tip, p.th = p.grip.Add(f.Mul(-3)), p.gh+7
		case 1:
			p.grip, p.gh = sh.Add(f.Mul(5)), 15
			p.tip, p.th = p.grip.Add(f.Mul(7)), p.gh-3
		default:
			p.grip, p.gh = sh.Add(f.Mul(-2.5*swing)), 13
			p.tip, p.th = p.grip.Add(f.Mul(2.5)), p.gh+8
		}
	case toolBasket:
		p.left = true
		lsh := sh.Mul(-1)
		p.lhand, p.lh = lsh.Add(f.Mul(1.5)), 13
		switch act {
		case 0:
			p.grip, p.gh = sh.Mul(0.4).Add(f.Mul(6)), 4
			p.lhand, p.lh = lsh.Add(f.Mul(3)), 9
		case 1:
			p.grip, p.gh = f.Mul(3.5).Add(r.Mul(-3.5)), 5
			p.lhand, p.lh = lsh.Add(f.Mul(3)), 9
		default:
			p.ok = false
		}
	}
	return p
}

var (
	haftCol  = RGB(170, 118, 62)
	toolHead = RGB(196, 202, 214)
)

// drawTool draws a tool in screen space from the haft end a to the head at
// b. fwd is the screen direction the villager faces, used to turn blades.
func drawTool(c *Canvas, tool int, a, b, fwd V) {
	d := b.Sub(a)
	l := math.Max(0.01, math.Hypot(d.X, d.Y))
	d = d.Mul(1 / l)
	n := V{-d.Y, d.X}
	if n.X*fwd.X+n.Y*fwd.Y < 0 || (math.Abs(n.X*fwd.X+n.Y*fwd.Y) < 0.1 && n.Y < 0) {
		n = n.Mul(-1)
	}
	c.Line(a, b, 2, haftCol)
	c.Line(a.Add(V{-0.4, -0.4}), b.Add(V{-0.4, -0.4}), 0.7, Shade(haftCol, 1.25))
	steel := func(lt float64) color.RGBA { return Shade(toolHead, lt) }
	switch tool {
	case toolAxe:
		back, front := b.Sub(d.Mul(1.4)), b.Add(d.Mul(1.8))
		e0, e1 := b.Add(n.Mul(5.5)).Add(d.Mul(3.6)), b.Add(n.Mul(5.5)).Sub(d.Mul(3))
		c.Tri(back, front.Sub(back), e0.Sub(back), solid(steel(1.05)))
		c.Tri(back, e0.Sub(back), e1.Sub(back), solid(steel(0.85)))
		c.Line(e0, e1, 1.3, RGB(250, 252, 255))
		c.Line(b, b.Sub(n.Mul(1.8)), 2.6, steel(0.7))
	case toolPick:
		for _, s := range []float64{-1, 1} {
			prev := b
			for i := 1; i <= 5; i++ {
				t := float64(i) / 5
				p := b.Add(n.Mul(s * 6.5 * t)).Sub(d.Mul(2.6 * t * t))
				c.Line(prev, p, 2.6-1.6*t, steel(1.1-0.25*t))
				prev = p
			}
		}
		c.Ellipse(b, 1.5, 1.5, solid(steel(0.75)))
	case toolHammer:
		c.Line(b.Sub(n.Mul(2.6)), b.Add(n.Mul(3)), 3.6, steel(0.8))
		c.Line(b.Sub(n.Mul(2.4)).Sub(d.Mul(0.8)), b.Add(n.Mul(2.8)).Sub(d.Mul(0.8)), 1.2, steel(1.2))
	case toolHoe:
		blade := b.Add(n.Mul(-0.5))
		c.Quad(blade, n.Mul(-5).Add(d.Mul(-1)), d.Mul(-3.2), solid(steel(1)))
		c.Line(blade.Add(n.Mul(-5)).Add(d.Mul(-1)), blade.Add(n.Mul(-5)).Add(d.Mul(-4.2)), 1.2, RGB(250, 252, 255))
	}
}

// basket draws a woven basket of berries standing at p.
func basket(c *Canvas, p V) {
	wicker := RGB(176, 128, 66)
	c.Quad(p.Add(V{-4, -6}), V{8, 0}, V{0.8, 6}, func(u, v float64) color.RGBA {
		f := 1.12 - 0.35*u
		if int(v*6)%2 == 0 {
			f *= 0.85
		}
		return Shade(wicker, f)
	})
	c.Ellipse(p.Add(V{0.2, -6}), 4.3, 1.8, solid(Shade(wicker, 0.75)))
	for i, o := range []V{{-2, -6.5}, {0.5, -7}, {2.5, -6.2}, {-0.5, -5.8}} {
		col := RGB(205, 35, 70)
		if i%2 == 1 {
			col = RGB(225, 60, 90)
		}
		c.Ellipse(p.Add(o), 1.3, 1.2, smoothLit(col))
	}
}

// crossbow draws a crossbow held at hand, pointing along f. act 0 is
// aimed and loaded, act 1 just loosed.
func crossbow(c *Canvas, P func(V, float64) V, hand V, hh float64, f, r V, act int, heavy bool) {
	span, prodCol := 6.5, RGB(205, 210, 222)
	if heavy {
		span, prodCol = 7.5, RGB(150, 154, 168)
	}
	lift := 0.0
	if act < 0 {
		lift = 2.5
	}
	front := hand.Add(f.Mul(8.5))
	fh := hh + 0.5 + lift
	c.Line(P(hand.Add(f.Mul(-2.5)), hh-1), P(front, fh), 2.8, RGB(110, 72, 38))
	l, rt := front.Add(r.Mul(-span)).Add(f.Mul(-1.4)), front.Add(r.Mul(span)).Add(f.Mul(-1.4))
	c.Line(P(l, fh), P(front, fh), 2.4, prodCol)
	c.Line(P(rt, fh), P(front, fh), 2.4, prodCol)
	nock := hand.Add(f.Mul(2))
	nh := hh + (fh-hh)*0.3
	if act == 1 {
		nock, nh = front.Add(f.Mul(-1.2)), fh
	}
	str := RGB(230, 225, 210)
	c.Line(P(l, fh), P(nock, nh), 0.7, str)
	c.Line(P(rt, fh), P(nock, nh), 0.7, str)
	if act != 1 {
		c.Line(P(nock, nh), P(front.Add(f.Mul(1.5)), fh), 1, RGB(90, 70, 50))
		c.Blend(int(P(front.Add(f.Mul(1.5)), fh).X), int(P(front.Add(f.Mul(1.5)), fh).Y), RGB(220, 222, 230))
	}
}
