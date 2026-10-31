// Package music synthesizes loopable medieval-style background music.
package music

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
)

const peak = 0.85

// Generate renders a loopable piece of medieval-style music as 16-bit signed
// little-endian interleaved stereo PCM at sampleRate. The same seed gives the same piece.
func Generate(sampleRate int, seed int64) []byte {
	rng := rand.New(rand.NewPCG(uint64(seed), 0x9e3779b97f4a7c15))
	bpm := 80 + rng.IntN(21)
	mode := dorian
	if rng.IntN(2) == 1 {
		mode = mixolydian
	}
	return render(sampleRate, generateStyle(bpm, mode), rng)
}

// Track renders soundtrack piece i (0 <= i < NumTracks), same PCM format as Generate.
func Track(sampleRate, i int) []byte {
	if i < 0 || i >= NumTracks {
		panic(fmt.Sprintf("music: track %d out of range [0, %d)", i, NumTracks))
	}
	if i == 0 {
		return Generate(sampleRate, 1)
	}
	if i >= len(tracks) {
		return gmRender(sampleRate, &gmTracks[i-len(tracks)])
	}
	st := tracks[i]
	return render(sampleRate, st, rand.New(rand.NewPCG(st.seed, 0x9e3779b97f4a7c15)))
}

func render(sampleRate int, st *style, rng *rand.Rand) []byte {
	p := compose(st, rng)
	mt := &st.meter
	sr := float64(sampleRate)
	spe := sr * 60 / float64(st.bpm) / float64(mt.beat)
	at := func(eighth int) int { return int(math.Round(float64(eighth) * spe)) }
	n := at(p.eighths)
	m := newMixer(sr, n, rng)

	freq := func(deg int, base float64) float64 {
		oct := int(math.Floor(float64(deg) / 7))
		semis := st.mode[deg-7*oct] + 12*oct
		return base * math.Pow(2, float64(semis)/12)
	}
	seconds := func(eighths int) float64 { return float64(eighths) / float64(mt.beat) * 60 / float64(st.bpm) }
	jitter := func() int { return int((rng.Float64() - 0.5) * 0.008 * sr) }
	lineOf := func(mel []note, shift int, base float64) []timedNote {
		line := make([]timedNote, len(mel))
		for i, nt := range mel {
			line[i] = timedNote{s: at(nt.at), e: at(nt.at + nt.dur), freq: freq(nt.deg+shift, base), vel: nt.vel}
		}
		last := &line[len(line)-1]
		last.e = last.s + (last.e-last.s)*85/100
		return line
	}

	var drumBar []int
	if len(mt.drumBars) > 0 {
		drumBar = pick(rng, mt.drumBars)
	}
	lead := st.tonic * st.leadMul
	for _, s := range p.sections {
		if s.luteLead {
			for _, nt := range s.melody {
				m.pluck(at(nt.at), freq(nt.deg, st.tonic), seconds(nt.dur), 0.32*nt.vel, -0.1, 2.8)
			}
		} else {
			m.line(lineOf(s.melody, 0, lead), st.lead)
		}
		if s.harmony {
			m.line(lineOf(s.melody, -2, lead), *st.harmony)
		}

		for bar, root := range s.chords {
			barStart := s.start + mt.bar*bar
			tones := [4]int{root - 7, root - 5, root - 3, root}
			for e, ti := range s.arp {
				if ti < 0 {
					continue
				}
				vel := 0.16
				if e == 0 {
					vel = 0.22
				}
				start := max(0, at(barStart+e)+jitter())
				m.pluck(start, freq(tones[ti], st.tonic), 1.5, st.arpGain*vel*(0.9+0.2*rng.Float64()), -0.35, 2.2)
			}
			if s.pad {
				st0, en := at(barStart), at(barStart+mt.bar)
				m.bowed(st0, en, freq(root-7, st.tonic), 0.05, -0.45)
				m.bowed(st0, en, freq(root-3, st.tonic), 0.04, 0.45)
			}

			if s.drums == 0 {
				continue
			}
			pattern := mt.drumThin
			if s.drums == 2 {
				pattern = drumBar
				if bar == 7 {
					pattern = mt.drumFill
				}
			}
			for e, hit := range pattern {
				if hit == 0 {
					continue
				}
				vel := 0.12 * (0.85 + 0.3*rng.Float64())
				if hit == 2 {
					vel = st.thumpGain * (0.9 + 0.2*rng.Float64())
				}
				if s.drums == 1 {
					vel *= 0.6
				}
				m.drum(max(0, at(barStart+e)+jitter()/2), hit == 2, vel, 0.1)
			}
		}
	}
	m.fold()
	m.drone(st.droneRef, st.droneGain)
	reverb(m.l, m.r, sr, st.wet)
	return encode(m.l, m.r)
}

type comb struct {
	buf         []float64
	p           int
	store, damp float64
	feedback    float64
}

func (c *comb) process(x float64) float64 {
	y := c.buf[c.p]
	c.store = y*(1-c.damp) + c.store*c.damp
	c.buf[c.p] = x + c.store*c.feedback
	if c.p++; c.p == len(c.buf) {
		c.p = 0
	}
	return y
}

type allpass struct {
	buf []float64
	p   int
}

func (a *allpass) process(x float64) float64 {
	b := a.buf[a.p]
	a.buf[a.p] = x + 0.5*b
	if a.p++; a.p == len(a.buf) {
		a.p = 0
	}
	return b - x
}

// reverb applies a small Schroeder reverb in place. It first runs over the
// end of the loop without output, so the reverb state at the start already
// holds the tail of the end and the loop point stays seamless.
func reverb(l, r []float64, sr, wet float64) {
	scale := sr / 44100
	newBank := func(spread int) ([]*comb, []*allpass) {
		var cs []*comb
		for _, d := range []int{1116, 1188, 1277, 1356} {
			cs = append(cs, &comb{buf: make([]float64, int(float64(d+spread)*scale)), damp: 0.3, feedback: 0.84})
		}
		var as []*allpass
		for _, d := range []int{556, 441} {
			as = append(as, &allpass{buf: make([]float64, int(float64(d+spread)*scale))})
		}
		return cs, as
	}
	run := func(cs []*comb, as []*allpass, x float64) float64 {
		y := 0.0
		for _, c := range cs {
			y += c.process(x)
		}
		for _, a := range as {
			y = a.process(y)
		}
		return y
	}
	lc, la := newBank(0)
	rc, ra := newBank(23)
	// The combs fall by about 60 dB per second, so warming up on the last
	// few seconds of the loop gives the same state as a full extra pass.
	for i := max(0, len(l)-int(6*sr)); i < len(l); i++ {
		x := 0.5 * (l[i] + r[i])
		run(lc, la, x)
		run(rc, ra, x)
	}
	for i := range l {
		x := 0.5 * (l[i] + r[i])
		l[i] += wet * run(lc, la, x)
		r[i] += wet * run(rc, ra, x)
	}
}

func encode(l, r []float64) []byte {
	top := 0.0
	for i := range l {
		top = max(top, math.Abs(l[i]), math.Abs(r[i]))
	}
	gain := 0.0
	if top > 0 && !math.IsInf(top, 0) && !math.IsNaN(top) {
		gain = peak / top
	}
	out := make([]byte, 4*len(l))
	put := func(i int, v float64) {
		v *= gain
		if math.IsNaN(v) {
			v = 0
		}
		v = math.Max(-peak, math.Min(peak, v))
		binary.LittleEndian.PutUint16(out[i:], uint16(int16(math.Round(v*32767))))
	}
	for i := range l {
		put(4*i, l[i])
		put(4*i+2, r[i])
	}
	return out
}
