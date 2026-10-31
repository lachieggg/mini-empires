package music

// meter describes a time signature in eighth notes.
type meter struct {
	name     string
	bar      int   // eighths per bar
	beat     int   // eighths per counted beat
	strongAt []int // strong positions within a bar
	busy     [][]int
	calm     [][]int
	arps     [][]int
	thinArp  []int
	drumBars [][]int
	drumThin []int
	drumFill []int
}

func (mt *meter) strong(pos int) bool {
	p := pos % mt.bar
	for _, s := range mt.strongAt {
		if p == s {
			return true
		}
	}
	return false
}

// Arp entries index the chord tones root, third, fifth and octave; -1 is a rest.
// Drum entries: 2 is a center thump, 1 a rim tap.
var (
	common = meter{
		name: "4/4", bar: 8, beat: 2, strongAt: []int{0, 4},
		busy:     [][]int{{2, 2, 2, 2}, {3, 1, 2, 2}, {2, 1, 1, 2, 2}, {2, 2, 1, 1, 2}, {3, 1, 3, 1}},
		calm:     [][]int{{4, 2, 2}, {2, 2, 4}, {3, 1, 4}, {6, 2}, {4, 4}},
		arps:     [][]int{{0, 2, 3, 2, 1, 2, 3, 2}, {0, -1, 2, 3, 1, -1, 2, -1}, {0, 2, 1, 2, 0, 2, 1, 2}},
		thinArp:  []int{0, -1, -1, -1, 2, -1, -1, -1},
		drumBars: [][]int{{2, 0, 1, 0, 2, 0, 1, 1}, {2, 0, 0, 1, 2, 0, 1, 0}},
		drumThin: []int{2, 0, 0, 0, 1, 0, 0, 0},
		drumFill: []int{2, 1, 1, 1, 2, 1, 2, 2},
	}
	march = meter{
		name: "4/4", bar: 8, beat: 2, strongAt: []int{0, 4},
		busy:     [][]int{{3, 1, 2, 2}, {2, 2, 2, 2}, {3, 1, 3, 1}, {2, 1, 1, 4}},
		calm:     [][]int{{4, 4}, {2, 2, 4}, {6, 2}, {4, 2, 2}},
		arps:     [][]int{{0, -1, 2, -1, 0, -1, 2, -1}, {0, -1, 3, -1, 2, -1, 3, -1}},
		thinArp:  []int{0, -1, -1, -1, 2, -1, -1, -1},
		drumBars: [][]int{{2, 0, 1, 1, 2, 0, 1, 1}, {2, 0, 1, 0, 2, 1, 1, 1}},
		drumThin: []int{2, 0, 0, 0, 2, 0, 0, 0},
		drumFill: []int{2, 1, 1, 1, 2, 1, 1, 1},
	}
	battle = meter{
		name: "4/4", bar: 8, beat: 2, strongAt: []int{0, 4},
		busy:     [][]int{{1, 1, 2, 1, 1, 2}, {2, 1, 1, 2, 2}, {1, 1, 1, 1, 2, 2}, {3, 1, 2, 2}},
		calm:     [][]int{{2, 2, 4}, {3, 1, 4}, {4, 2, 2}},
		arps:     [][]int{{0, 0, 2, 0, 3, 0, 2, 0}, {0, 0, 1, 0, 2, 0, 1, 0}},
		thinArp:  []int{0, -1, 0, -1, 0, -1, 2, -1},
		drumBars: [][]int{{2, 1, 2, 1, 2, 1, 2, 1}, {2, 0, 2, 1, 2, 0, 2, 2}},
		drumThin: []int{2, 0, 0, 0, 2, 0, 2, 0},
		drumFill: []int{2, 2, 2, 2, 2, 1, 2, 2},
	}
	waltz = meter{
		name: "3/4", bar: 6, beat: 2, strongAt: []int{0},
		busy:     [][]int{{2, 2, 2}, {2, 1, 1, 2}, {3, 1, 2}, {1, 1, 2, 2}},
		calm:     [][]int{{4, 2}, {6}, {2, 4}, {3, 3}},
		arps:     [][]int{{0, 2, 3, 2, 1, 2}, {0, -1, 2, -1, 3, -1}},
		thinArp:  []int{0, -1, 2, -1, -1, -1},
		drumBars: [][]int{{2, 0, 1, 0, 1, 0}, {2, 0, 1, 1, 1, 0}},
		drumThin: []int{2, 0, 0, 0, 0, 0},
		drumFill: []int{2, 1, 1, 1, 2, 2},
	}
	jig = meter{
		name: "6/8", bar: 6, beat: 3, strongAt: []int{0, 3},
		busy:     [][]int{{1, 1, 1, 1, 1, 1}, {2, 1, 2, 1}, {1, 1, 1, 2, 1}, {2, 1, 1, 1, 1}},
		calm:     [][]int{{3, 3}, {2, 1, 3}, {3, 2, 1}, {6}},
		arps:     [][]int{{0, 2, 3, 0, 2, 3}, {0, 2, 1, 3, 2, 1}},
		thinArp:  []int{0, -1, -1, 2, -1, -1},
		drumBars: [][]int{{2, 0, 1, 2, 1, 1}, {2, 0, 1, 1, 0, 1}},
		drumThin: []int{2, 0, 0, 1, 0, 0},
		drumFill: []int{2, 1, 1, 2, 1, 2},
	}
	slow = meter{
		name: "4/4", bar: 8, beat: 2, strongAt: []int{0, 4},
		busy:     [][]int{{4, 2, 2}, {3, 1, 4}, {2, 2, 4}},
		calm:     [][]int{{8}, {6, 2}, {4, 4}},
		arps:     [][]int{{0, -1, 2, -1, 3, -1, 2, -1}, {0, -1, -1, 2, -1, -1, 3, -1}},
		thinArp:  []int{0, -1, -1, -1, -1, -1, -1, -1},
		drumThin: []int{2, 0, 0, 0, 0, 0, 0, 0},
	}
)

// voice is the timbre of a sustained melodic line.
type voice struct {
	table    *wavetable
	breath   float64
	vibDepth float64
	vibRate  float64
	attack   float64
	gain     float64
	pan      float64
	cutoff   float64 // lowpass corner in Hz; 0 leaves the line unfiltered
}

var (
	recorder  = voice{table: fluteTable, breath: 0.3, vibDepth: 0.006, vibRate: 5.3, attack: 0.035, gain: 0.2, pan: 0.2}
	softFlute = voice{table: fluteTable, breath: 0.45, vibDepth: 0.004, vibRate: 4.6, attack: 0.09, gain: 0.2, pan: 0.15}
	fiddle    = voice{table: bowTable, breath: 0.05, vibDepth: 0.008, vibRate: 6, attack: 0.02, gain: 0.15, pan: 0.2}
	shawm     = voice{table: shawmTable, breath: 0.1, vibDepth: 0.003, vibRate: 5, attack: 0.015, gain: 0.13, pan: 0.15}
)

type style struct {
	name, modeName string
	mode           [7]int
	meter          meter
	bpm            int     // counted beats per minute
	tonic          float64 // Hz of the lute register tonic
	leadMul        float64 // lead register relative to tonic
	droneRef       float64 // drone voices sit at droneRef/4, 3/8 and /2
	droneGain      float64
	form           string
	lead           voice
	harmony        *voice // a gymel line a third below, on later A sections
	introLute      bool
	quietIntro     bool
	padKinds       string
	drums          int // highest drum level any section may use
	thumpGain      float64
	arpGain        float64
	wet            float64
	seed           uint64
}

const (
	d4Hz = 293.6648
	e3Hz = 164.8138
	e4Hz = 329.6276
	f4Hz = 349.2282
	g3Hz = 195.9977
	a3Hz = 220.0
)

// generateStyle is the piece Generate makes: bpm and mode come from the seed.
func generateStyle(bpm int, mode [7]int) *style {
	st := &style{
		name: "Castle Green", modeName: "D Dorian", mode: mode, meter: common, bpm: bpm,
		tonic: d4Hz, leadMul: 2, droneRef: d4Hz, droneGain: 0.05,
		form: "AABACABA", lead: recorder, introLute: true, padKinds: "BC", drums: 2,
		thumpGain: 0.45, arpGain: 1, wet: 0.08,
	}
	if mode == mixolydian {
		st.modeName = "D Mixolydian"
	}
	// Eight bars per section keeps every tempo in the 120-180 s window.
	if bpm < 88 {
		st.form = "AABACBA"
	}
	return st
}

var flute2 = voice{table: fluteTable, breath: 0.25, vibDepth: 0.005, vibRate: 5.0, attack: 0.04, gain: 0.12, pan: -0.25}
var fiddle2 = voice{table: bowTable, breath: 0.03, vibDepth: 0.007, vibRate: 5.6, attack: 0.03, gain: 0.09, pan: -0.3}
var shawm2 = voice{table: shawmTable, breath: 0.08, vibDepth: 0.003, vibRate: 4.8, attack: 0.02, gain: 0.08, pan: -0.3}

// Track 0 is Generate(sampleRate, 1); the rest are fixed styles.
var tracks = []*style{
	nil,
	{
		name: "Pastoral Morning", modeName: "G Ionian", mode: ionian, meter: common, bpm: 84,
		tonic: g3Hz, leadMul: 2, droneRef: 2 * g3Hz, droneGain: 0.045,
		form: "AABACBA", lead: recorder, harmony: &flute2, introLute: true, padKinds: "C", drums: 1,
		thumpGain: 0.35, arpGain: 1, wet: 0.09, seed: 11,
	},
	{
		name: "Tavern Jig", modeName: "D Dorian", mode: dorian, meter: jig, bpm: 112,
		tonic: d4Hz, leadMul: 2, droneRef: d4Hz, droneGain: 0.035,
		form: "AABBAACCAABB", lead: fiddle, harmony: &flute2, padKinds: "", drums: 2,
		thumpGain: 0.4, arpGain: 1.1, wet: 0.05, seed: 22,
	},
	{
		name: "Mystic Grove", modeName: "A Aeolian", mode: aeolian, meter: slow, bpm: 64,
		tonic: a3Hz, leadMul: 2, droneRef: 2 * a3Hz, droneGain: 0.07,
		form: "ABACA", lead: softFlute, padKinds: "ABC", drums: 1,
		thumpGain: 0.3, arpGain: 0.8, wet: 0.15, seed: 33,
	},
	{
		name: "March of Banners", modeName: "D Mixolydian", mode: mixolydian, meter: march, bpm: 100,
		tonic: d4Hz, leadMul: 2, droneRef: d4Hz, droneGain: 0.05,
		form: "AABACABA", lead: shawm, harmony: &shawm2, quietIntro: true, padKinds: "C", drums: 2,
		thumpGain: 0.55, arpGain: 0.9, wet: 0.07, seed: 44,
	},
	{
		name: "Harvest Dance", modeName: "E Dorian", mode: dorian, meter: waltz, bpm: 138,
		tonic: e4Hz, leadMul: 2, droneRef: e4Hz, droneGain: 0.04,
		form: "AABBACCAABBA", lead: recorder, harmony: &fiddle2, padKinds: "C", drums: 2,
		thumpGain: 0.4, arpGain: 1, wet: 0.06, seed: 55,
	},
	{
		name: "Clash of Steel", modeName: "E Phrygian", mode: phrygian, meter: battle, bpm: 118,
		tonic: e3Hz, leadMul: 2, droneRef: 2 * e3Hz, droneGain: 0.06,
		form: "AABACABA", lead: shawm, padKinds: "ABC", drums: 2,
		thumpGain: 0.6, arpGain: 1.1, wet: 0.06, seed: 66,
	},
	{
		name: "Evening Embers", modeName: "F Ionian", mode: ionian, meter: waltz, bpm: 76,
		tonic: f4Hz, leadMul: 1, droneRef: f4Hz, droneGain: 0.05,
		form: "AABACA", lead: softFlute, harmony: &fiddle2, introLute: true, padKinds: "B", drums: 0,
		arpGain: 0.9, wet: 0.11, seed: 77,
	},
}

// NumTracks is the number of distinct pieces in the soundtrack: the
// medieval set, then the General-MIDI-style set from gmTracks.
const NumTracks = 16

// TrackName returns a short display name for piece i.
func TrackName(i int) string {
	switch {
	case i == 0:
		return "Castle Green"
	case i >= len(tracks):
		return gmTracks[i-len(tracks)].name
	}
	return tracks[i].name
}
