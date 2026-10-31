package music

import (
	"math"
	"math/rand/v2"
)

var (
	harmonicMinor     = [7]int{0, 2, 3, 5, 7, 8, 11}
	phrygianDominant  = [7]int{0, 1, 4, 5, 7, 8, 10}
	glockRatios       = []float64{1, 2.76, 5.4}
	glockAmps         = []float64{1, 0.35, 0.12}
	celestaRatios     = []float64{1, 2, 3.9}
	celestaAmps       = []float64{1, 0.3, 0.08}
	accompHarp        = 0
	accompGuitar      = 1
	accompHarpsichord = 2
	accompAccordion   = 3
)

// gmVoice is a lead or countermelody instrument. frac is how much of each
// note sounds: 1 is legato, lower values detach the notes.
type gmVoice struct {
	v      voice
	frac   float64
	double bool // add a slightly detuned copy, as accordion reeds do
}

var (
	gmFlute    = gmVoice{v: voice{table: fluteTable, breath: 0.12, vibDepth: 0.005, vibRate: 5.2, attack: 0.04, gain: 0.2, pan: 0.1, cutoff: 5000}, frac: 0.95}
	gmOboe     = gmVoice{v: voice{table: oboeTable, breath: 0.04, vibDepth: 0.006, vibRate: 5.5, attack: 0.03, gain: 0.12, pan: -0.15, cutoff: 3500}, frac: 0.95}
	gmClarinet = gmVoice{v: voice{table: clarinetTable, breath: 0.03, vibDepth: 0.003, vibRate: 5, attack: 0.03, gain: 0.16, pan: -0.2, cutoff: 3000}, frac: 1}
	gmTrumpet  = gmVoice{v: voice{table: trumpetTable, breath: 0.02, vibDepth: 0.004, vibRate: 5.5, attack: 0.02, gain: 0.11, pan: 0.15, cutoff: 4500}, frac: 0.8}
	gmHorn     = gmVoice{v: voice{table: hornTable, breath: 0.02, vibDepth: 0.003, vibRate: 5, attack: 0.06, gain: 0.16, pan: -0.25, cutoff: 2500}, frac: 1}
	gmViolin   = gmVoice{v: voice{table: stringTable, breath: 0.02, vibDepth: 0.008, vibRate: 6, attack: 0.05, gain: 0.1, pan: 0.25, cutoff: 4000}, frac: 1}
	gmAccord   = gmVoice{v: voice{table: accordionTbl, vibRate: 1, attack: 0.02, gain: 0.08, pan: 0.2, cutoff: 4000}, frac: 0.9, double: true}
	gmOrgan    = gmVoice{v: voice{table: organTable, vibRate: 1, attack: 0.03, gain: 0.1, pan: -0.1, cutoff: 3500}, frac: 1}
)

type gmTrack struct {
	name, key, leads string
	mode             [7]int
	meter            meter
	bpm              int
	base             float64 // lead register tonic in Hz
	form             string  // I is a four-bar intro; A, B and C are eight bars
	intro            [4]int
	progA            [8]int
	progB            [8]int
	progC            [8]int
	leadA, leadB     gmVoice
	counter          gmVoice
	fanfare          bool // lead plays a call over the intro
	sparkle          int  // 1 glockenspiel doubles the returning tune, 2 celesta arpeggios
	accomp           int
	accompPat        []int
	bass             int // 0 pizzicato, 1 plucked bass, 2 oom-pah horn
	pad              *wavetable
	padGain          float64
	drone            float64
	perc             string
	wet, chorus      float64
	seed             uint64
}

var gmTracks = []gmTrack{
	{
		name: "Market Square", key: "C major", leads: "flute, oboe; violin countermelody",
		mode: ionian, meter: common, bpm: 108, base: 523.2511, form: "IAABACABA",
		intro: [4]int{0, 3, 1, 4}, progA: [8]int{0, 3, 4, 0, 5, 3, 4, 0},
		progB: [8]int{3, 4, 2, 5, 1, 4, 0, 4}, progC: [8]int{5, 2, 3, 0, 1, 5, 1, 4},
		leadA: gmFlute, leadB: gmOboe, counter: gmViolin, sparkle: 1,
		accomp: accompHarp, accompPat: []int{0, 2, 3, 4, 3, 2, 1, 2}, bass: 0,
		pad: stringTable, padGain: 0.025, perc: "town", wet: 0.1, chorus: 0.25, seed: 101,
	},
	{
		name: "Saltwind Shanty", key: "D Dorian", leads: "accordion; violin countermelody",
		mode: dorian, meter: jig, bpm: 92, base: 587.3295, form: "IAABBAACCAAB",
		intro: [4]int{0, 6, 0, 4}, progA: [8]int{0, 6, 2, 6, 0, 3, 6, 0},
		progB: [8]int{2, 6, 3, 0, 2, 6, 3, 4}, progC: [8]int{3, 3, 0, 0, 6, 6, 3, 4},
		leadA: gmAccord, leadB: gmAccord, counter: gmViolin,
		accomp: accompGuitar, accompPat: []int{0, 2, 3, 1, 2, 3}, bass: 1,
		perc: "shanty", wet: 0.09, chorus: 0.2, seed: 102,
	},
	{
		name: "Sands of Khaleth", key: "E Phrygian dominant", leads: "oboe, clarinet; horn countermelody",
		mode: phrygianDominant, meter: common, bpm: 96, base: 659.2551, form: "IAABACBA",
		intro: [4]int{0, 1, 0, 1}, progA: [8]int{0, 1, 0, 6, 3, 1, 1, 0},
		progB: [8]int{3, 3, 0, 0, 6, 6, 1, 0}, progC: [8]int{6, 3, 1, 0, 6, 3, 1, 0},
		leadA: gmOboe, leadB: gmClarinet, counter: gmHorn,
		accomp: accompHarp, accompPat: []int{0, -1, 2, 3, 1, -1, 2, 1}, bass: 1,
		drone: 0.04, perc: "bazaar", wet: 0.12, chorus: 0.2, seed: 103,
	},
	{
		name: "Banner of the Brave", key: "D major", leads: "trumpet; French horn countermelody",
		mode: ionian, meter: march, bpm: 112, base: 587.3295, form: "IAABACABA",
		intro: [4]int{0, 3, 0, 4}, progA: [8]int{0, 4, 5, 3, 0, 3, 4, 0},
		progB: [8]int{3, 0, 3, 4, 5, 3, 1, 4}, progC: [8]int{5, 2, 3, 0, 5, 1, 4, 4},
		leadA: gmTrumpet, leadB: gmTrumpet, counter: gmHorn, fanfare: true,
		accomp: accompHarp, accompPat: []int{0, -1, 2, -1, 3, -1, 2, -1}, bass: 0,
		pad: stringTable, padGain: 0.03, perc: "march", wet: 0.1, chorus: 0.2, seed: 104,
	},
	{
		name: "Willowmere Waltz", key: "G major", leads: "flute, clarinet; violin countermelody",
		mode: ionian, meter: waltz, bpm: 132, base: 783.9909, form: "IAABBACCABA",
		intro: [4]int{0, 3, 0, 4}, progA: [8]int{0, 0, 3, 0, 1, 4, 4, 0},
		progB: [8]int{3, 3, 0, 5, 1, 1, 4, 4}, progC: [8]int{5, 5, 3, 3, 0, 1, 4, 4},
		leadA: gmFlute, leadB: gmClarinet, counter: gmViolin, sparkle: 1,
		accomp: accompHarp, accompPat: []int{0, 2, 3, 4, 3, 2}, bass: 0,
		pad: stringTable, padGain: 0.022, perc: "waltz", wet: 0.12, chorus: 0.25, seed: 105,
	},
	{
		name: "Crypt of Whispers", key: "A harmonic minor", leads: "clarinet, organ; choir pad, celesta",
		mode: harmonicMinor, meter: slow, bpm: 70, base: 440, form: "IABACA",
		intro: [4]int{0, 5, 3, 4}, progA: [8]int{0, 5, 3, 4, 0, 5, 4, 0},
		progB: [8]int{5, 3, 5, 4, 0, 2, 3, 4}, progC: [8]int{3, 0, 5, 2, 3, 5, 4, 4},
		leadA: gmClarinet, leadB: gmOrgan, counter: gmOrgan, sparkle: 2,
		accomp: accompHarp, accompPat: []int{0, -1, -1, 2, -1, -1, 3, -1}, bass: 1,
		pad: choirTable, padGain: 0.03, perc: "crypt", wet: 0.18, chorus: 0.3, seed: 106,
	},
	{
		name: "Carnival of Lights", key: "F major", leads: "glockenspiel and flute; accordion countermelody",
		mode: ionian, meter: common, bpm: 120, base: 698.4565, form: "IAABACABA",
		intro: [4]int{0, 4, 0, 4}, progA: [8]int{0, 0, 4, 4, 0, 3, 4, 0},
		progB: [8]int{3, 3, 0, 0, 1, 4, 1, 4}, progC: [8]int{5, 1, 4, 0, 5, 1, 4, 4},
		leadA: gmFlute, leadB: gmOboe, counter: gmAccord, sparkle: 1,
		accomp: accompAccordion, accompPat: []int{-1, -1, 0, -1, -1, -1, 0, -1}, bass: 2,
		perc: "carnival", wet: 0.09, chorus: 0.25, seed: 107,
	},
	{
		name: "Lonely Moor", key: "E Aeolian", leads: "French horn, oboe; flute countermelody",
		mode: aeolian, meter: slow, bpm: 76, base: 659.2551, form: "IAABCA",
		intro: [4]int{0, 5, 2, 6}, progA: [8]int{0, 5, 2, 6, 0, 5, 6, 0},
		progB: [8]int{3, 0, 5, 2, 3, 5, 6, 4}, progC: [8]int{5, 6, 0, 0, 5, 6, 4, 4},
		leadA: gmHorn, leadB: gmOboe, counter: gmFlute,
		accomp: accompHarp, accompPat: []int{0, 2, 3, 4, 3, 2, 3, 2}, bass: 1,
		pad: stringTable, padGain: 0.028, perc: "moor", wet: 0.15, chorus: 0.25, seed: 108,
	},
}

type gmSection struct {
	kind    byte
	start   int // eighths
	bars    int
	chords  []int
	melody  []note
	counter []note
	lead    gmVoice
	cvoice  gmVoice
	glock   bool
}

// counterLine writes a slower second voice on chord tones below the tune,
// moving by the smallest step and avoiding the tune's own pitch.
func counterLine(rng *rand.Rand, mt *meter, start int, chords []int, mel []note) []note {
	slots := mt.strongAt
	melAt := func(pos int) int {
		d := 0
		for _, nt := range mel {
			if nt.at > pos {
				break
			}
			d = nt.deg
		}
		return d
	}
	var out []note
	prev := -3
	for bar, root := range chords {
		for k, slot := range slots {
			end := mt.bar
			if k+1 < len(slots) {
				end = slots[k+1]
			}
			pos := start + bar*mt.bar + slot
			md := melAt(pos)
			best, bestScore := prev, math.Inf(1)
			for oct := -14; oct <= 7; oct += 7 {
				for _, t := range chordTones(root) {
					c := t + oct
					if c < -7 || c > 3 || c >= md-1 {
						continue
					}
					score := math.Abs(float64(c-prev)) + 0.5*rng.Float64()
					if ((c-md)%7+7)%7 == 0 {
						score += 2
					}
					if score < bestScore {
						best, bestScore = c, score
					}
				}
			}
			prev = best
			out = append(out, note{at: pos, dur: end - slot, deg: best, vel: 0.85})
		}
	}
	return out
}

func gmCompose(tr *gmTrack, rng *rand.Rand) []gmSection {
	st := &style{meter: tr.meter, mode: tr.mode}
	mA, nA := st.newMotif(rng, 1), st.newMotif(rng, -1)
	mA2, nA2 := mA.vary(rng), nA.vary(rng)
	mB, nB := st.newMotif(rng, -1), st.newMotif(rng, 1)
	mC := st.newMotif(rng, 1)
	mC2 := mC.vary(rng)
	startA := []int{0, 2, 4}[rng.IntN(3)]

	bar := tr.meter.bar
	var out []gmSection
	pos, aCount := 0, 0
	for _, k := range tr.form {
		s := gmSection{kind: byte(k), start: pos, bars: 8}
		switch k {
		case 'I':
			s.bars, s.chords = 4, tr.intro[:]
			if tr.fanfare {
				s.lead = tr.leadA
				s.melody = fanfare(rng, &tr.meter, pos, tr.intro)
			}
		case 'A':
			s.chords, s.lead = tr.progA[:], tr.leadA
			cells := [3]motif{mA, mA, nA}
			if aCount%2 == 1 {
				cells = [3]motif{mA, mA2, nA2}
			}
			s.melody = st.phrase(rng, pos, tr.progA, cells, [3]int{0, 0, 0}, startA, 0)
			if aCount > 0 {
				s.counter, s.cvoice = counterLine(rng, &tr.meter, pos, s.chords, s.melody), tr.counter
			}
			s.glock = tr.sparkle == 1 && aCount >= 2
			aCount++
		case 'B':
			s.chords, s.lead, s.cvoice = tr.progB[:], tr.leadB, tr.counter
			s.melody = st.phrase(rng, pos, tr.progB, [3]motif{mB, mB, nB}, [3]int{0, 2, 0}, 5, 4)
			s.counter = counterLine(rng, &tr.meter, pos, s.chords, s.melody)
		case 'C':
			// The countermelody instrument takes the tune and the lead accompanies.
			s.chords, s.lead, s.cvoice = tr.progC[:], tr.counter, tr.leadA
			s.melody = st.phrase(rng, pos, tr.progC, [3]motif{mC, mC2, mC}, [3]int{0, 0, -2}, 2, 4)
			s.counter = counterLine(rng, &tr.meter, pos, s.chords, s.melody)
		}
		out = append(out, s)
		pos += s.bars * bar
	}
	return out
}

// fanfare writes a rising triadic call over the intro, ending on the fifth.
func fanfare(rng *rand.Rand, mt *meter, start int, chords [4]int) []note {
	var out []note
	for b, root := range chords[:3] {
		rhythm := mt.busy[rng.IntN(len(mt.busy))]
		pos := 0
		for j, d := range rhythm {
			tone := chordTones(root)[j%3]
			if j == len(rhythm)-1 {
				tone = root + 7
			}
			out = append(out, note{at: start + b*mt.bar + pos, dur: d, deg: clampDeg(tone), vel: 0.9})
			pos += d
		}
	}
	return append(out, note{at: start + 3*mt.bar, dur: mt.bar, deg: 4, vel: 1})
}

func gmRender(sampleRate int, tr *gmTrack) []byte {
	rng := rand.New(rand.NewPCG(tr.seed, 0x9e3779b97f4a7c15))
	sections := gmCompose(tr, rng)
	mt := &tr.meter
	sr := float64(sampleRate)
	spe := sr * 60 / float64(tr.bpm) / float64(mt.beat)
	at := func(eighth int) int { return int(math.Round(float64(eighth) * spe)) }
	last := sections[len(sections)-1]
	n := at(last.start + last.bars*mt.bar)
	m := newMixer(sr, n, rng)

	freq := func(deg int, base float64) float64 {
		oct := int(math.Floor(float64(deg) / 7))
		semis := tr.mode[deg-7*oct] + 12*oct
		return base * math.Pow(2, float64(semis)/12)
	}
	jitter := func() int { return int((rng.Float64() - 0.5) * 0.006 * sr) }
	acc := tr.base / 2
	play := func(mel []note, gv gmVoice, gain float64) {
		if len(mel) == 0 {
			return
		}
		line := make([]timedNote, len(mel))
		for i, nt := range mel {
			s := at(nt.at)
			e := s + int(float64(at(nt.at+nt.dur)-s)*gv.frac)
			line[i] = timedNote{s: s, e: e, freq: freq(nt.deg, tr.base), vel: nt.vel}
		}
		l := &line[len(line)-1]
		l.e = l.s + (l.e-l.s)*85/100
		v := gv.v
		v.gain *= gain
		m.line(line, v)
		if gv.double {
			for i := range line {
				line[i].freq *= 1.0045
			}
			v.pan = -v.pan
			v.gain *= 0.7
			m.line(line, v)
		}
	}

	for si, s := range sections {
		play(s.melody, s.lead, 1)
		play(s.counter, s.cvoice, 0.6)
		if s.glock {
			for _, nt := range s.melody {
				if nt.dur >= 2 || mt.strong(nt.at-s.start) {
					m.bell(at(nt.at), freq(nt.deg, tr.base), 0.08*nt.vel, 0.3, 0.35, glockRatios, glockAmps)
				}
			}
		}

		for b, root := range s.chords {
			barStart := s.start + b*mt.bar
			tones := []int{root - 7, root - 5, root - 3, root, root + 2}
			for e, ti := range tr.accompPat {
				if ti < 0 {
					continue
				}
				t := max(0, at(barStart+e)+jitter())
				vel := 0.13 * (0.9 + 0.2*rng.Float64())
				if e == 0 {
					vel *= 1.2
				}
				f := freq(tones[ti], acc)
				switch tr.accomp {
				case accompHarp:
					m.pluck(t, f, 1.5, vel, -0.3, 2.6)
				case accompGuitar:
					m.pluck(t, f, 1.0, vel*1.1, -0.3, 1.6)
				case accompHarpsichord:
					m.pluck(t, f, 0.8, vel, -0.3, 1.0)
					m.pluck(t, 2*f, 0.8, 0.4*vel, -0.3, 0.8)
				case accompAccordion:
					for _, ct := range []int{root - 5, root - 3, root} {
						m.pad(t, t+int(0.7*spe), freq(ct, acc), 0.035, 0.25, accordionTbl, 0.01, 0.06, 0)
					}
				}
			}

			bassAt := []int{0}
			switch {
			case mt.bar == 8:
				bassAt = []int{0, 4}
			case mt.beat == 3:
				bassAt = []int{0, 3}
			}
			for k, p := range bassAt {
				deg := root - 14
				if k == 1 {
					deg = root - 10
				}
				t, f := at(barStart+p), freq(deg, acc)
				switch tr.bass {
				case 0:
					m.pluck(t, f, 0.3, 0.3, 0, 0.5)
				case 1:
					m.pluck(t, f, 1.0, 0.3, 0, 1.4)
				case 2:
					m.pad(t, t+int(1.4*spe), f, 0.09, 0, hornTable, 0.02, 0.08, 0)
				}
			}

			if tr.pad != nil && (s.kind != 'I' || b >= 2) {
				st, en := at(barStart), at(barStart+mt.bar)
				for k, ct := range []int{root - 7, root - 5, root - 3} {
					m.pad(st, en, freq(ct, acc), tr.padGain, float64(k-1)*0.5, tr.pad, 0.3, 0.45, 0.003)
				}
			}

			if tr.sparkle == 2 && b%2 == 0 {
				for k, ct := range []int{root + 7, root + 9, root + 11, root + 14} {
					m.bell(at(barStart+2*k+1), freq(ct, acc), 0.05, 0.4-0.25*float64(k), 0.6, celestaRatios, celestaAmps)
				}
			}

			gmPerc(m, tr, rng, at, mt, barStart, b, s, si == 0, freq(root-14, acc))
		}
	}
	m.fold()
	if tr.drone > 0 {
		m.drone(tr.base/2, tr.drone)
	}
	chorus(m.l, m.r, sr, tr.chorus)
	reverb(m.l, m.r, sr, tr.wet)
	return encode(m.l, m.r)
}

func gmPerc(m *mixer, tr *gmTrack, rng *rand.Rand, at func(int) int, mt *meter, barStart, b int, s gmSection, first bool, rootHz float64) {
	hit := func(e int) int { return at(barStart + e) }
	vary := func() float64 { return 0.85 + 0.3*rng.Float64() }
	tamb := func(e int, v float64) { m.noiseHit(hit(e), 0.05*v*vary(), 0.35, 0.06, 0.9, 0) }
	lastBar := b == s.bars-1
	timp := rootHz
	for timp < 70 {
		timp *= 2
	}
	switch tr.perc {
	case "town":
		if s.kind == 'I' {
			return
		}
		m.drum(hit(0), true, 0.25*vary(), 0)
		tamb(2, 1)
		tamb(6, 1)
		if s.kind != 'C' {
			tamb(5, 0.5)
		}
	case "shanty":
		m.drum(hit(0), true, 0.35*vary(), 0)
		m.drum(hit(3), true, 0.2*vary(), 0)
		if s.kind != 'I' {
			tamb(3, 1)
			tamb(5, 0.6)
		}
	case "bazaar":
		m.drum(hit(0), true, 0.3*vary(), -0.1)
		m.drum(hit(3), true, 0.2*vary(), -0.1)
		for _, e := range []int{2, 5, 6, 7} {
			m.drum(hit(e), false, 0.1*vary(), 0.2)
		}
		if s.kind != 'I' {
			tamb(4, 0.8)
		}
	case "march":
		if s.kind == 'I' {
			for e := range mt.bar {
				m.timpani(hit(e), timp, 0.06*float64(b*mt.bar+e+4)/float64(4*mt.bar), -0.2, 0.3)
			}
			return
		}
		if b == 0 {
			m.noiseHit(hit(0), 0.1, 0.3, 0.8, 0.95, 0)
		}
		m.drum(hit(0), true, 0.35*vary(), 0)
		m.drum(hit(4), true, 0.25*vary(), 0)
		m.timpani(hit(0), timp, 0.18*vary(), -0.2, 0.6)
		snare := []int{2, 6, 7}
		if lastBar {
			snare = []int{0, 1, 2, 3, 4, 5, 6, 7}
		}
		for _, e := range snare {
			m.noiseHit(hit(e), 0.06*vary(), 0.15, 0.09, 0.6, 190)
		}
	case "waltz":
		if s.kind == 'I' {
			return
		}
		tamb(2, 0.7)
		tamb(4, 0.7)
	case "crypt":
		if b == 0 {
			m.timpani(hit(0), timp, 0.25, -0.2, 1.2)
		}
		if lastBar && s.kind != 'I' {
			for e := range 8 {
				m.timpani(hit(e), timp, 0.03+0.015*float64(e), -0.2, 0.35)
			}
		}
	case "carnival":
		m.drum(hit(0), true, 0.3*vary(), 0)
		m.drum(hit(4), true, 0.25*vary(), 0)
		for _, e := range []int{1, 3, 5, 7} {
			tamb(e, 0.8)
		}
		if b == 0 && !first {
			m.noiseHit(hit(0), 0.06, 0.3, 0.6, 0.95, 0)
		}
	case "moor":
		if b == 0 || lastBar {
			m.timpani(hit(0), timp, 0.15, -0.2, 0.9)
		}
	}
}
