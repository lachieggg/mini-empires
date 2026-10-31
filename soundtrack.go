package main

import (
	"bytes"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"mini-empires/internal/music"
)

const sampleRate = 44100

type rendered struct {
	track int
	pcm   []byte
}

// soundtrack plays the tracks in a shuffled order, one after another. The
// next track is rendered in the background while the current one plays.
type soundtrack struct {
	ctx     *audio.Context
	player  *audio.Player
	order   []int
	pos     int
	current int
	on      bool
	ready   chan rendered
	pending bool
}

func newSoundtrack() *soundtrack {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	s := &soundtrack{
		ctx:     audio.NewContext(sampleRate),
		order:   rng.Perm(music.NumTracks),
		on:      true,
		current: -1,
		ready:   make(chan rendered, 1),
	}
	s.render(s.order[0])
	return s
}

func (s *soundtrack) render(track int) {
	s.pending = true
	go func() { s.ready <- rendered{track, music.Track(sampleRate, track)} }()
}

// update starts the next track when the current one has finished.
func (s *soundtrack) update() {
	select {
	case r := <-s.ready:
		s.pending = false
		p, err := s.ctx.NewPlayer(bytes.NewReader(r.pcm))
		if err != nil {
			return
		}
		if s.player != nil {
			s.player.Close()
		}
		s.player, s.current = p, r.track
		p.SetVolume(0.45)
		if s.on {
			p.Play()
		}
	default:
	}
	if s.on && s.player != nil && !s.player.IsPlaying() && !s.pending {
		s.skip()
	}
}

func (s *soundtrack) skip() {
	if s.pending {
		return
	}
	s.pos = (s.pos + 1) % len(s.order)
	s.render(s.order[s.pos])
}

func (s *soundtrack) toggle() {
	s.on = !s.on
	if s.player == nil {
		return
	}
	if s.on {
		s.player.Play()
	} else {
		s.player.Pause()
	}
}

// nowPlaying names the current track, or "" when none is playing.
func (s *soundtrack) nowPlaying() string {
	if !s.on || s.current < 0 {
		return ""
	}
	return music.TrackName(s.current)
}
