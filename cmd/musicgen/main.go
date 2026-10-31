// Command musicgen writes procedurally generated music loops to WAV files.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"mini-empires/internal/music"
)

func main() {
	out := flag.String("out", "music.wav", "output WAV file, or output directory with -all")
	seed := flag.Int64("seed", 1, "random seed for Generate")
	track := flag.Int("track", -1, fmt.Sprintf("soundtrack piece to render (0-%d) instead of a seeded piece", music.NumTracks-1))
	all := flag.Bool("all", false, "write every soundtrack piece as track-N.wav into the -out directory")
	rate := flag.Int("rate", 44100, "sample rate in Hz")
	flag.Parse()

	switch {
	case *all:
		dir := *out
		if dir == "music.wav" {
			dir = "."
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatal(err)
		}
		for i := range music.NumTracks {
			write(filepath.Join(dir, fmt.Sprintf("track-%d.wav", i)), music.TrackName(i), *rate, func() []byte { return music.Track(*rate, i) })
		}
	case *track >= 0:
		if *track >= music.NumTracks {
			log.Fatalf("track %d out of range [0, %d)", *track, music.NumTracks)
		}
		write(*out, music.TrackName(*track), *rate, func() []byte { return music.Track(*rate, *track) })
	default:
		write(*out, fmt.Sprintf("seed %d", *seed), *rate, func() []byte { return music.Generate(*rate, *seed) })
	}
}

func write(path, label string, rate int, gen func() []byte) {
	start := time.Now()
	pcm := gen()
	elapsed := time.Since(start)
	if err := os.WriteFile(path, wav(pcm, rate), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %s (%s): %.1f s of audio, generated in %v\n", path, label, float64(len(pcm)/4)/float64(rate), elapsed.Round(time.Millisecond))
}

func wav(pcm []byte, rate int) []byte {
	const channels, bits = 2, 16
	h := make([]byte, 44)
	copy(h[0:], "RIFF")
	binary.LittleEndian.PutUint32(h[4:], uint32(36+len(pcm)))
	copy(h[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(h[16:], 16)
	binary.LittleEndian.PutUint16(h[20:], 1)
	binary.LittleEndian.PutUint16(h[22:], channels)
	binary.LittleEndian.PutUint32(h[24:], uint32(rate))
	binary.LittleEndian.PutUint32(h[28:], uint32(rate*channels*bits/8))
	binary.LittleEndian.PutUint16(h[32:], channels*bits/8)
	binary.LittleEndian.PutUint16(h[34:], bits)
	copy(h[36:], "data")
	binary.LittleEndian.PutUint32(h[40:], uint32(len(pcm)))
	return append(h, pcm...)
}
