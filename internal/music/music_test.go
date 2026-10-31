package music

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

const testRate = 44100

func samples(pcm []byte) []float64 {
	out := make([]float64, len(pcm)/2)
	for i := range out {
		out[i] = float64(int16(binary.LittleEndian.Uint16(pcm[2*i:]))) / 32768
	}
	return out
}

func checkPiece(t *testing.T, pcm []byte, minSecs, maxSecs float64) {
	t.Helper()
	if len(pcm)%4 != 0 {
		t.Fatalf("length %d is not a whole number of stereo frames", len(pcm))
	}
	secs := float64(len(pcm)/4) / testRate
	if secs < minSecs || secs > maxSecs {
		t.Errorf("length %.1f s, want %.0f-%.0f s", secs, minSecs, maxSecs)
	}

	s := samples(pcm)
	top := 0.0
	for _, v := range s {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatal("non-finite sample")
		}
		top = max(top, math.Abs(v))
	}
	if top > 0.851 {
		t.Errorf("peak %.3f, want <= 0.85", top)
	}
	if top < 0.5 {
		t.Errorf("peak %.3f, mix is unexpectedly quiet", top)
	}

	// The last frame must flow into the first one.
	n := len(s)
	for ch := range 2 {
		if d := math.Abs(s[n-2+ch] - s[ch]); d >= 0.05 {
			t.Errorf("channel %d: loop seam jump %.4f, want < 0.05", ch, d)
		}
	}
}

func TestGenerate(t *testing.T) {
	for _, seed := range []int64{1, 2, 3, 42} {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			checkPiece(t, Generate(testRate, seed), 120, 180)
		})
	}
}

func TestDeterministic(t *testing.T) {
	a, b := Generate(testRate, 7), Generate(testRate, 7)
	if !bytes.Equal(a, b) {
		t.Fatal("same seed produced different output")
	}
	if bytes.Equal(a, Generate(testRate, 8)) {
		t.Fatal("different seeds produced identical output")
	}
}

func TestTracks(t *testing.T) {
	names := map[string]bool{}
	for i := range NumTracks {
		name := TrackName(i)
		if name == "" || names[name] {
			t.Errorf("track %d: empty or duplicate name %q", i, name)
		}
		names[name] = true
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			pcm := Track(testRate, i)
			checkPiece(t, pcm, 90, 180)
			if !bytes.Equal(pcm, Track(testRate, i)) {
				t.Error("same track produced different output")
			}
		})
	}
}

func TestTrackCount(t *testing.T) {
	if got := len(tracks) + len(gmTracks); got != NumTracks {
		t.Fatalf("NumTracks is %d but %d tracks are defined", NumTracks, got)
	}
}

func TestTrackZeroIsGenerate(t *testing.T) {
	if !bytes.Equal(Track(testRate, 0), Generate(testRate, 1)) {
		t.Fatal("track 0 differs from Generate seed 1")
	}
}

func BenchmarkGenerate(b *testing.B) {
	for b.Loop() {
		Generate(48000, 1)
	}
}

func BenchmarkTrack(b *testing.B) {
	for i := range NumTracks {
		b.Run(TrackName(i), func(b *testing.B) {
			for b.Loop() {
				Track(48000, i)
			}
		})
	}
}
