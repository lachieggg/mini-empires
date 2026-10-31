package music

import "math/rand/v2"

var (
	ionian     = [7]int{0, 2, 4, 5, 7, 9, 11}
	dorian     = [7]int{0, 2, 3, 5, 7, 9, 10}
	phrygian   = [7]int{0, 1, 3, 5, 7, 8, 10}
	mixolydian = [7]int{0, 2, 4, 5, 7, 9, 10}
	aeolian    = [7]int{0, 2, 3, 5, 7, 8, 10}
)

// Chord roots are scale degrees. fixChord moves any root that makes a
// diminished triad in the chosen mode.
var (
	progsA = [][8]int{
		{0, 6, 0, 4, 0, 3, 6, 0},
		{0, 0, 6, 6, 0, 3, 4, 0},
		{0, 3, 0, 6, 0, 3, 6, 0},
		{0, 6, 3, 0, 1, 3, 6, 0},
	}
	progsB = [][8]int{
		{3, 3, 0, 0, 6, 6, 3, 4},
		{3, 6, 0, 1, 3, 6, 3, 4},
		{6, 6, 3, 3, 0, 1, 3, 4},
	}
	progsC = [][8]int{
		{1, 3, 6, 0, 1, 3, 6, 4},
		{3, 1, 6, 3, 0, 6, 3, 4},
	}
)

const (
	degMin = -3
	degMax = 9
)

type note struct {
	at, dur int // eighth notes
	deg     int
	vel     float64
}

// motif is a two-bar melodic idea: a rhythm plus the scale steps between notes.
type motif struct {
	rhythm []int
	steps  []int
}

type section struct {
	kind     byte
	start    int // eighth notes
	chords   [8]int
	melody   []note
	luteLead bool
	harmony  bool
	pad      bool
	arp      []int
	drums    int // 0 none, 1 light, 2 full
}

type piece struct {
	st       *style
	sections []section
	eighths  int
}

func fixChord(mode [7]int, root int) int {
	if (mode[(root+4)%7]-mode[root]+12)%12 == 6 {
		return (root + 5) % 7
	}
	return root
}

func (st *style) newMotif(rng *rand.Rand, bias int) motif {
	var m motif
	m.rhythm = append(m.rhythm, st.meter.busy[rng.IntN(len(st.meter.busy))]...)
	m.rhythm = append(m.rhythm, st.meter.calm[rng.IntN(len(st.meter.calm))]...)
	m.steps = make([]int, len(m.rhythm))
	half := len(m.rhythm) / 2
	for i := 1; i < len(m.steps); i++ {
		dir := bias
		if i > half {
			dir = -bias
		}
		if rng.IntN(3) == 0 {
			dir = -dir
		}
		size := 1
		switch r := rng.IntN(10); {
		case r < 1:
			size = 0
		case r < 3:
			size = 2
		case r < 4:
			size = 3
		}
		// A leap is followed by a step back, as in plainchant.
		if prev := m.steps[i-1]; prev >= 3 || prev <= -3 {
			size, dir = 1, -sign(prev)
		}
		m.steps[i] = dir * size
	}
	return m
}

func (m motif) vary(rng *rand.Rand) motif {
	v := motif{rhythm: append([]int(nil), m.rhythm...), steps: append([]int(nil), m.steps...)}
	switch rng.IntN(3) {
	case 0:
		i := len(v.steps) - 1
		v.steps[i] = -v.steps[i]
	case 1:
		for i, d := range v.rhythm {
			if d >= 4 && d%2 == 0 {
				v.rhythm = append(v.rhythm[:i+1], v.rhythm[i:]...)
				v.rhythm[i], v.rhythm[i+1] = d/2, d/2
				v.steps = append(v.steps[:i+1], v.steps[i:]...)
				v.steps[i+1] = 1 - 2*rng.IntN(2)
				break
			}
		}
	default:
		if len(v.steps) > 2 {
			i := 1 + rng.IntN(len(v.steps)-2)
			v.steps[i] += 1 - 2*rng.IntN(2)
		}
	}
	return v
}

func sign(x int) int {
	if x < 0 {
		return -1
	}
	return 1
}

func clampDeg(d int) int {
	for d > degMax {
		d -= 2
	}
	for d < degMin {
		d += 2
	}
	return d
}

func chordTones(root int) [3]int { return [3]int{root, root + 2, root + 4} }

// nearestChordTone returns the chord tone (in any octave) closest to d.
func nearestChordTone(d, root int) int {
	best, bestDist := d, 99
	for oct := -14; oct <= 14; oct += 7 {
		for _, t := range chordTones(root) {
			c := t + oct
			dist := c - d
			if dist < 0 {
				dist = -dist
			}
			if dist < bestDist && c >= degMin && c <= degMax {
				best, bestDist = c, dist
			}
		}
	}
	return best
}

func (mt *meter) velocity(rng *rand.Rand, pos int) float64 {
	v := 0.82
	if pos%mt.bar == 0 {
		v = 1
	} else if mt.strong(pos) {
		v = 0.92
	}
	return v * (0.95 + 0.1*rng.Float64())
}

// phrase realizes an eight-bar phrase: three motif cells and a cadence on target.
func (st *style) phrase(rng *rand.Rand, start int, chords [8]int, cells [3]motif, shifts [3]int, startDeg, target int) []note {
	mt := &st.meter
	bar := mt.bar
	var out []note
	last := startDeg
	for c, m := range cells {
		deg := nearestChordTone(last+shifts[c], chords[2*c])
		pos := 0
		for i, d := range m.rhythm {
			if i > 0 {
				deg = clampDeg(deg + m.steps[i])
			}
			if mt.strong(pos) {
				if t := nearestChordTone(deg, chords[2*c+pos/bar]); t-deg <= 1 && deg-t <= 1 {
					deg = t
				}
			}
			out = append(out, note{at: start + 2*bar*c + pos, dur: d, deg: deg, vel: mt.velocity(rng, pos)})
			pos += d
		}
		last = deg
	}

	if target == 0 && last > 3 {
		target = 7
	}
	approach := mt.calm[rng.IntN(len(mt.calm))]
	if rng.IntN(2) == 0 {
		approach = mt.busy[rng.IntN(len(mt.busy))]
	}
	dir := 1
	if last < target {
		dir = -1
	}
	pos := 6 * bar
	for j, d := range approach {
		deg := clampDeg(target + dir*(len(approach)-j))
		out = append(out, note{at: start + pos, dur: d, deg: deg, vel: mt.velocity(rng, pos)})
		pos += d
	}
	out = append(out, note{at: start + 7*bar, dur: bar, deg: target, vel: 1})
	return out
}

func pick[T any](rng *rand.Rand, xs []T) T { return xs[rng.IntN(len(xs))] }

func compose(st *style, rng *rand.Rand) piece {
	p := piece{st: st}
	fix := func(prog [8]int) [8]int {
		for i, r := range prog {
			prog[i] = fixChord(st.mode, r)
		}
		return prog
	}
	progA := fix(pick(rng, progsA))
	progB := fix(pick(rng, progsB))
	progC := fix(pick(rng, progsC))

	mA, nA := st.newMotif(rng, 1), st.newMotif(rng, -1)
	mA2, nA2 := mA.vary(rng), nA.vary(rng)
	mA3 := mA2.vary(rng)
	mB, nB := st.newMotif(rng, -1), st.newMotif(rng, 1)
	mC := st.newMotif(rng, 1)
	mC2 := mC.vary(rng)
	startA := []int{0, 2, 4}[rng.IntN(3)]
	arpA := pick(rng, st.meter.arps)
	arpB := pick(rng, st.meter.arps)

	sectionBars := 8 * st.meter.bar
	aCount := 0
	for i, k := range st.form {
		s := section{kind: byte(k), start: sectionBars * i, drums: 2, pad: contains(st.padKinds, byte(k))}
		switch k {
		case 'A':
			s.chords, s.arp = progA, arpA
			cells := [3]motif{mA, mA2, nA}
			if aCount%2 == 1 {
				cells = [3]motif{mA, mA3, nA2}
			}
			s.melody = st.phrase(rng, s.start, progA, cells, [3]int{0, 0, 0}, startA, 0)
			if i == 0 && st.introLute {
				s.luteLead, s.arp, s.drums = true, st.meter.thinArp, 1
			}
			s.harmony = st.harmony != nil && aCount >= 2
			aCount++
		case 'B':
			s.chords, s.arp = progB, arpB
			s.melody = st.phrase(rng, s.start, progB, [3]motif{mB, mB, nB}, [3]int{0, 2, 0}, 5, 4)
		case 'C':
			s.chords, s.arp, s.drums = progC, st.meter.thinArp, 1
			s.melody = st.phrase(rng, s.start, progC, [3]motif{mC, mC2, mC}, [3]int{0, 0, -2}, 2, 4)
		}
		if i == 0 && st.quietIntro {
			s.drums = 1
		}
		s.drums = min(s.drums, st.drums)
		p.sections = append(p.sections, s)
	}
	p.eighths = sectionBars * len(st.form)
	return p
}

func contains(s string, b byte) bool {
	for i := range len(s) {
		if s[i] == b {
			return true
		}
	}
	return false
}
