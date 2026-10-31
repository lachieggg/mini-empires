package music

import (
	"math"
	"math/rand/v2"
)

const tableSize = 2048

// wavetable holds one cycle of a waveform built from harmonic amplitudes.
type wavetable [tableSize + 1]float64

func newTable(harmonics ...float64) *wavetable {
	var t wavetable
	peak := 0.0
	for j := range tableSize {
		v := 0.0
		for h, a := range harmonics {
			v += a * math.Sin(2*math.Pi*float64((h+1)*j)/tableSize)
		}
		t[j] = v
		peak = max(peak, math.Abs(v))
	}
	for j := range tableSize {
		t[j] /= peak
	}
	t[tableSize] = t[0]
	return &t
}

// at returns the value at phase, measured in cycles.
func (t *wavetable) at(phase float64) float64 {
	p := phase - math.Floor(phase)
	x := p * tableSize
	i := int(x)
	f := x - float64(i)
	return t[i] + f*(t[i+1]-t[i])
}

var (
	sineTable  = newTable(1)
	fluteTable = newTable(1, 0.22, 0.07, 0.02)
	reedTable  = newTable(1, 0.5, 0.3, 0.18, 0.1)
	bowTable   = newTable(1, 0.5, 0.33, 0.25, 0.2, 0.14)
	shawmTable = newTable(1, 0.85, 0.7, 0.55, 0.4, 0.3, 0.2, 0.12)
)

// mixer is a stereo buffer one loop long plus a tail. Sound written into
// the tail is folded back onto the head, so notes ringing past the loop
// point continue at the start.
type mixer struct {
	sr   float64
	n    int
	l, r []float64
	rng  *rand.Rand
}

func newMixer(sr float64, n int, rng *rand.Rand) *mixer {
	tail := min(n, int(10*sr))
	return &mixer{sr: sr, n: n, l: make([]float64, n+tail), r: make([]float64, n+tail), rng: rng}
}

func panGains(pan float64) (float64, float64) {
	a := (pan + 1) * math.Pi / 4
	return math.Cos(a), math.Sin(a)
}

func (m *mixer) fold() {
	for i := m.n; i < len(m.l); i++ {
		m.l[i-m.n] += m.l[i]
		m.r[i-m.n] += m.r[i]
	}
	m.l, m.r = m.l[:m.n], m.r[:m.n]
}

func (m *mixer) noise() float64 { return 2*m.rng.Float64() - 1 }

// pluck renders a Karplus-Strong string.
func (m *mixer) pluck(start int, freq, dur, vel, pan, t60 float64) {
	n := max(2, int(m.sr/freq-0.5))
	raw := make([]float64, n)
	for i := range raw {
		raw[i] = m.noise()
	}
	// The ring repeats every period, so filter it circularly: run twice
	// and keep the second pass, or the wrap point becomes a click.
	ring := make([]float64, n)
	lp1, lp2, mean := 0.0, 0.0, 0.0
	for pass := range 2 {
		for i, x := range raw {
			lp1 += (x - lp1) * 0.2
			lp2 += (lp1 - lp2) * 0.2
			if pass == 1 {
				ring[i] = 3 * lp2
				mean += ring[i]
			}
		}
	}
	mean /= float64(n)
	// Subtracting a delayed copy imitates plucking near the bridge,
	// which gives the hollow lute timbre.
	off := n / 5
	exc := make([]float64, n)
	for i := range ring {
		// The ring is periodic, so the delayed copy wraps around it.
		exc[i] = ring[i] - mean - 0.6*(ring[(i-off+n)%n]-mean)
	}
	ring = exc

	g := math.Pow(10, -3/(t60*freq))
	total := min(int((dur+1.2)*m.sr), int(t60*m.sr), len(m.l)-start)
	attack := int(0.003 * m.sr)
	release := int(0.03 * m.sr)
	gl, gr := panGains(pan)
	p := 0
	for j := range total {
		next := p + 1
		if next == n {
			next = 0
		}
		y := ring[p]
		ring[p] = g * 0.5 * (y + ring[next])
		p = next
		env := vel
		if j < attack {
			env *= float64(j) / float64(attack)
		}
		if rem := total - j; rem < release {
			env *= float64(rem) / float64(release)
		}
		v := y * env
		m.l[start+j] += v * gl
		m.r[start+j] += v * gr
	}
}

type timedNote struct {
	s, e      int
	freq, vel float64
}

// line renders a legato melodic line with one continuous oscillator,
// so pitch glides between notes instead of restarting.
func (m *mixer) line(notes []timedNote, vc voice) {
	if len(notes) == 0 {
		return
	}
	sr := m.sr
	aAtk := 1 - math.Exp(-1/(vc.attack*sr))
	aRel := 1 - math.Exp(-1/(0.08*sr))
	glide := 1 - math.Exp(-1/(0.02*sr))
	tongue := int(0.012 * sr)
	gl, gr := panGains(vc.pan)
	end := min(notes[len(notes)-1].e+int(0.5*sr), len(m.l))

	freq, amp, ph, vph := notes[0].freq, 0.0, 0.0, 0.0
	n1, n2 := 0.0, 0.0
	lp, lpCoef := 0.0, 1-math.Exp(-2*math.Pi*vc.cutoff/sr)
	k := 0
	for i := notes[0].s; i < end; i++ {
		for k+1 < len(notes) && i >= notes[k+1].s {
			k++
		}
		nt := notes[k]
		target := 0.0
		if i < nt.e {
			target = nt.vel
			if i-nt.s < tongue && k > 0 && notes[k-1].e >= nt.s {
				target *= 0.55
			}
		}
		freq += (nt.freq - freq) * glide
		if target > amp {
			amp += (target - amp) * aAtk
		} else {
			amp += (target - amp) * aRel
		}
		t := float64(i-nt.s) / sr
		depth := 0.0
		if t > 0.25 {
			depth = vc.vibDepth * min(1, (t-0.25)/0.5)
		}
		vph += vc.vibRate / sr
		ph += freq * (1 + depth*sineTable.at(vph)) / sr
		if ph >= 1 {
			ph--
		}
		n1 += (m.noise() - n1) * 0.35
		n2 += (n1 - n2) * 0.05
		v := amp * vc.gain * (vc.table.at(ph) + vc.breath*(n1-n2))
		if vc.cutoff > 0 {
			lp += (v - lp) * lpCoef
			v = lp
		}
		m.l[i] += v * gl
		m.r[i] += v * gr
	}
}

// bowed renders a soft hurdy-gurdy-like pad note with slow attack and release.
func (m *mixer) bowed(s, e int, freq, vel, pan float64) {
	sr := m.sr
	attack, release := 0.3*sr, 0.5*sr
	end := min(e+int(release), len(m.l))
	gl, gr := panGains(pan)
	ph := 0.0
	for i := s; i < end; i++ {
		t := float64(i - s)
		env := min(1, t/attack)
		if i >= e {
			env *= 1 - float64(i-e)/release
		}
		ph += freq * (1 + 0.002*sineTable.at(t*0.6/sr)) / sr
		if ph >= 1 {
			ph--
		}
		v := vel * env * bowTable.at(ph)
		m.l[i] += v * gl
		m.r[i] += v * gr
	}
}

// drum renders a frame-drum stroke: a pitch-dropping tone plus a short
// lowpassed noise burst. A low stroke is the center thump, a high one a rim tap.
func (m *mixer) drum(s int, low bool, vel, pan float64) {
	sr := m.sr
	base, bend, toneDecay, noiseDecay, cutoff, noiseAmt := 180.0, 60.0, 0.06, 0.025, 0.25, 0.6
	if low {
		base, bend, toneDecay, noiseDecay, cutoff, noiseAmt = 55, 70, 0.18, 0.04, 0.08, 0.5
	}
	total := min(int(0.5*sr), len(m.l)-s)
	attack := 0.001 * sr
	gl, gr := panGains(pan)
	kBend, kTone, kNoise := math.Exp(-1/(0.03*sr)), math.Exp(-1/(toneDecay*sr)), math.Exp(-1/(noiseDecay*sr))
	eBend, eTone, eNoise := 1.0, 1.0, 1.0
	ph, lp1, lp2 := 0.0, 0.0, 0.0
	for j := range total {
		ph += (base + bend*eBend) / sr
		lp1 += (m.noise() - lp1) * cutoff
		lp2 += (lp1 - lp2) * cutoff
		v := sineTable.at(ph)*eTone + noiseAmt*lp2*eNoise
		eBend, eTone, eNoise = eBend*kBend, eTone*kTone, eNoise*kNoise
		v *= vel * min(1, float64(j)/attack)
		if rem := total - j; rem < 200 {
			v *= float64(rem) / 200
		}
		m.l[s+j] += v * gl
		m.r[s+j] += v * gr
	}
}

// drone renders root and fifth over exactly one loop. Every frequency is
// rounded to a whole number of cycles per loop so the drone wraps without
// a phase jump.
func (m *mixer) drone(root, gain float64) {
	n := float64(m.n)
	cycles := func(f float64) float64 { return math.Max(1, math.Round(f*n/m.sr)) }
	voices := []struct{ f, g, pan, lfo, lfoPh float64 }{
		{root / 4, 1, 0, 9.3, 0},
		{root * 3 / 8, 0.7, 0.25, 11.7, 0.3},
		{root / 2, 0.35, -0.25, 7.1, 0.6},
	}
	for _, v := range voices {
		inc := cycles(v.f) / n
		lfoInc := cycles(1/v.lfo) / n
		gl, gr := panGains(v.pan)
		ph, lph := 0.0, v.lfoPh
		for i := range m.n {
			a := gain * v.g * (0.7 + 0.3*sineTable.at(lph)) * reedTable.at(ph)
			m.l[i] += a * gl
			m.r[i] += a * gr
			ph += inc
			if ph >= 1 {
				ph--
			}
			lph += lfoInc
		}
	}
}
