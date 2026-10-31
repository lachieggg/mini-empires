package music

import "math"

// Timbres for the General-MIDI-style soundtrack. Each is a fixed harmonic
// recipe, which is roughly what a small wavetable soundfont plays.
var (
	oboeTable     = newTable(0.6, 1, 0.9, 0.7, 0.5, 0.35, 0.25, 0.15, 0.1)
	clarinetTable = newTable(1, 0.05, 0.6, 0.05, 0.35, 0.03, 0.2, 0, 0.1)
	trumpetTable  = newTable(1, 0.85, 0.7, 0.55, 0.45, 0.35, 0.25, 0.18, 0.12, 0.08)
	hornTable     = newTable(1, 0.55, 0.3, 0.15, 0.08, 0.04)
	accordionTbl  = newTable(1, 0.7, 0.6, 0.45, 0.4, 0.3, 0.25, 0.2, 0.15, 0.1, 0.08)
	organTable    = newTable(1, 0.6, 0.35, 0.3, 0, 0.2, 0, 0.12)
	choirTable    = newTable(0.6, 1, 0.7, 0.35, 0.15, 0.08)
	stringTable   = newTable(1, 0.6, 0.45, 0.35, 0.28, 0.22, 0.17, 0.13, 0.1, 0.08)
)

// pad renders one sustained note with linear attack and release, for
// string, choir and organ chords.
func (m *mixer) pad(s, e int, freq, vel, pan float64, tbl *wavetable, attack, release, vib float64) {
	sr := m.sr
	att, rel := math.Max(1, attack*sr), math.Max(1, release*sr)
	end := min(e+int(rel), len(m.l))
	gl, gr := panGains(pan)
	ph, vph := 0.0, 0.0
	inc := freq / sr
	vinc := 5.2 / sr
	for i := s; i < end; i++ {
		env := min(1, float64(i-s)/att)
		if i >= e {
			env *= 1 - float64(i-e)/rel
		}
		vph += vinc
		ph += inc * (1 + vib*sineTable.at(vph))
		if ph >= 1 {
			ph--
		}
		v := vel * env * tbl.at(ph)
		m.l[i] += v * gl
		m.r[i] += v * gr
	}
}

// bell renders a glockenspiel or celesta strike from a few inharmonic partials.
func (m *mixer) bell(s int, freq, vel, pan, decay float64, ratios, amps []float64) {
	sr := m.sr
	total := min(int(4*decay*sr), len(m.l)-s)
	gl, gr := panGains(pan)
	for p, ratio := range ratios {
		f := freq * ratio
		if f > min(12000, 0.45*sr) {
			continue
		}
		inc := f / sr
		k := math.Exp(-1 / (decay / (1 + float64(p)) * sr))
		amp := vel * amps[p]
		ph := 0.0
		attack := 0.002 * sr
		for j := range total {
			v := amp * sineTable.at(ph) * min(1, float64(j)/attack)
			if rem := total - j; rem < 200 {
				v *= float64(rem) / 200
			}
			m.l[s+j] += v * gl
			m.r[s+j] += v * gr
			ph += inc
			amp *= k
		}
	}
}

// timpani renders a tuned kettle-drum stroke.
func (m *mixer) timpani(s int, freq, vel, pan, decay float64) {
	sr := m.sr
	total := min(int(3*decay*sr), len(m.l)-s)
	gl, gr := panGains(pan)
	kTone, kNoise, kBend := math.Exp(-1/(decay*sr)), math.Exp(-1/(0.03*sr)), math.Exp(-1/(0.05*sr))
	eTone, eNoise, eBend := 1.0, 1.0, 1.0
	ph, lp := 0.0, 0.0
	attack := 0.002 * sr
	for j := range total {
		ph += freq * (1 + 0.04*eBend) / sr
		lp += (m.noise() - lp) * 0.1
		v := sineTable.at(ph)*eTone + 0.4*sineTable.at(ph*1.5)*eTone*eTone + 0.5*lp*eNoise
		v *= vel * min(1, float64(j)/attack)
		if rem := total - j; rem < 200 {
			v *= float64(rem) / 200
		}
		m.l[s+j] += v * gl
		m.r[s+j] += v * gr
		eTone, eNoise, eBend = eTone*kTone, eNoise*kNoise, eBend*kBend
	}
}

// noiseHit renders filtered noise with an exponential decay, for snare,
// tambourine and cymbal. bright sets how much of the low band is removed.
func (m *mixer) noiseHit(s int, vel, pan, decay, bright, toneHz float64) {
	sr := m.sr
	total := min(int(4*decay*sr), len(m.l)-s)
	gl, gr := panGains(pan)
	k := math.Exp(-1 / (decay * sr))
	kTone := math.Exp(-1 / (0.04 * sr))
	env, eTone, lp, ph := 1.0, 1.0, 0.0, 0.0
	attack := 0.001 * sr
	for j := range total {
		x := m.noise()
		lp += (x - lp) * 0.25
		v := (x - bright*lp) * env
		if toneHz > 0 {
			ph += toneHz / sr
			v += 0.6 * sineTable.at(ph) * eTone
			eTone *= kTone
		}
		v *= vel * min(1, float64(j)/attack)
		if rem := total - j; rem < 200 {
			v *= float64(rem) / 200
		}
		m.l[s+j] += v * gl
		m.r[s+j] += v * gr
		env *= k
	}
}

// chorus adds a slowly modulated delayed copy, the thickening most 2000s
// soundfont synths applied by default. The delay reads wrap around the
// loop and the LFO makes whole cycles per loop, so the seam stays clean.
func chorus(l, r []float64, sr, mix float64) {
	n := len(l)
	dl := append([]float64(nil), l...)
	dr := append([]float64(nil), r...)
	cycles := math.Max(1, math.Round(0.6*float64(n)/sr))
	base, depth := 0.014*sr, 0.003*sr
	read := func(buf []float64, pos float64) float64 {
		i := int(math.Floor(pos))
		f := pos - float64(i)
		a, b := ((i%n)+n)%n, (((i+1)%n)+n)%n
		return buf[a] + f*(buf[b]-buf[a])
	}
	for i := range n {
		ph := float64(i) * cycles / float64(n)
		l[i] += mix * read(dl, float64(i)-base-depth*sineTable.at(ph))
		r[i] += mix * read(dr, float64(i)-base-depth*sineTable.at(ph+0.25))
	}
}
