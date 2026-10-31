package main

import (
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"mini-empires/internal/rl"
)

// netPolicy plays with a learned network. get is called on every decision,
// so a watched game picks up new weights as soon as the learner publishes them.
type netPolicy struct {
	get    func() *rl.Net
	rng    *rand.Rand
	greedy bool
	probs  []float64 // last decision's action probabilities, when show is set
	show   bool
}

func (p *netPolicy) Name() string { return "Learned AI" }

func (p *netPolicy) Choose(g *Game, team int, obs []float64, mask []bool) (int, float64, float64) {
	n := p.get()
	if p.show {
		p.probs = n.Probs(obs, mask)
	}
	if p.greedy {
		return n.Greedy(obs, mask), 0, 0
	}
	return n.Act(obs, mask, p.rng)
}

const (
	batchSteps = 6000
	historyMax = 400
)

type gameResult struct {
	eps      []rl.Episode
	eval     bool
	evalWin  float64 // 1 win, 0.5 draw, 0 loss for the learner against the scripted AI
	score    float64 // mean total score of the learning players
	duration float64
}

type HistPoint struct {
	Games   int
	Updates int
	EvalWin float64
	Score   float64
	Entropy float64
}

// Trainer runs self-play games on worker goroutines and trains one network
// with PPO on a learner goroutine.
type Trainer struct {
	net     atomic.Pointer[rl.Net]
	learner *rl.Learner
	path    string

	mu        sync.Mutex
	running   bool
	stop      chan struct{}
	wg        sync.WaitGroup
	games     int
	evalWins  []float64
	scores    []float64
	durations []float64
	last      rl.UpdateStats
	history   []HistPoint
	startT    time.Time
	startG    int
}

func policyPath() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, ".mini-empires", "policy.json")
}

func NewTrainer(path string) *Trainer {
	t := &Trainer{path: path}
	net := rl.NewNet(rl.DefaultConfig(numObs, numActions), time.Now().UnixNano())
	if f, err := os.Open(path); err == nil {
		if n, err := rl.Load(f); err == nil && n.Config().Obs == numObs && n.Config().Actions == numActions {
			net = n
		}
		f.Close()
	}
	t.learner = rl.NewLearner(net)
	t.net.Store(t.learner.Snapshot())
	return t
}

func (t *Trainer) Net() *rl.Net { return t.net.Load() }

func (t *Trainer) Reset() {
	t.Stop()
	t.mu.Lock()
	defer t.mu.Unlock()
	net := rl.NewNet(rl.DefaultConfig(numObs, numActions), time.Now().UnixNano())
	t.learner = rl.NewLearner(net)
	t.net.Store(t.learner.Snapshot())
	t.games, t.evalWins, t.scores, t.durations, t.history = 0, nil, nil, nil, nil
	t.last = rl.UpdateStats{}
	os.Remove(t.path)
}

func (t *Trainer) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running {
		return
	}
	t.running = true
	t.stop = make(chan struct{})
	t.startT, t.startG = time.Now(), t.games
	results := make(chan gameResult, 16)
	workers := max(1, runtime.NumCPU()-2)
	for i := range workers {
		t.wg.Add(1)
		go t.worker(i, results)
	}
	t.wg.Add(1)
	go t.learn(results)
}

func (t *Trainer) Stop() {
	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return
	}
	t.running = false
	close(t.stop)
	t.mu.Unlock()
	t.wg.Wait()
	t.save()
}

func (t *Trainer) Running() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

func (t *Trainer) stopped() bool {
	select {
	case <-t.stop:
		return true
	default:
		return false
	}
}

// playTrainingGame runs one headless game. Every fourth game is against
// the scripted AI, so progress can be measured against a fixed opponent.
func playTrainingGame(seed int64, net func() *rl.Net, eval bool, stop func() bool) (gameResult, bool) {
	rng := rand.New(rand.NewSource(seed))
	snap := net()
	get := func() *rl.Net { return snap }
	opts := GameOptions{Seed: seed, Human: -1, Record: true, TimeLimit: trainTimeLimit()}
	opts.Policies[0] = &netPolicy{get: get, rng: rng}
	if eval {
		opts.Policies[1] = &scripted{rng: rng}
	} else {
		opts.Policies[1] = &netPolicy{get: get, rng: rng}
	}
	g := NewGame(opts)
	if eval {
		g.brains[1].record = false
	}
	for !g.over() {
		for range 600 {
			g.step()
		}
		if stop() {
			return gameResult{}, false
		}
	}
	res := gameResult{eval: eval, duration: g.time}
	learners := 0
	for team, b := range g.brains {
		if !b.record || len(b.steps) == 0 {
			continue
		}
		ep := rl.Episode{Done: true, Steps: make([]rl.Step, len(b.steps))}
		for i, s := range b.steps {
			ep.Steps[i] = rl.Step{Obs: s.Obs, Mask: s.Mask, Action: s.Action, LogP: s.LogP, Value: s.Value, Reward: s.Reward}
		}
		res.eps = append(res.eps, ep)
		res.score += g.score(team).Total
		learners++
	}
	if learners > 0 {
		res.score /= float64(learners)
	}
	switch g.winner {
	case 0:
		res.evalWin = 1
	case 2:
		res.evalWin = 0.5
	}
	return res, true
}

func (t *Trainer) worker(id int, out chan<- gameResult) {
	defer t.wg.Done()
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)*7919))
	for n := 0; ; n++ {
		res, ok := playTrainingGame(rng.Int63(), t.Net, (n+id)%4 == 3, t.stopped)
		if !ok {
			return
		}
		select {
		case out <- res:
		case <-t.stop:
			return
		}
	}
}

func appendCapped(s []float64, v float64, n int) []float64 {
	s = append(s, v)
	if len(s) > n {
		s = s[len(s)-n:]
	}
	return s
}

func mean(s []float64) float64 {
	if len(s) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range s {
		sum += v
	}
	return sum / float64(len(s))
}

func (t *Trainer) learn(in <-chan gameResult) {
	defer t.wg.Done()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var batch []rl.Episode
	steps := 0
	for {
		var res gameResult
		select {
		case res = <-in:
		case <-t.stop:
			return
		}
		t.mu.Lock()
		t.games++
		if res.eval {
			t.evalWins = appendCapped(t.evalWins, res.evalWin, 40)
		}
		t.scores = appendCapped(t.scores, res.score, 40)
		t.durations = appendCapped(t.durations, res.duration, 40)
		t.mu.Unlock()

		batch = append(batch, res.eps...)
		for _, e := range res.eps {
			steps += len(e.Steps)
		}
		if steps < batchSteps {
			continue
		}
		st := t.learner.Update(batch, rng)
		t.net.Store(t.learner.Snapshot())
		batch, steps = nil, 0

		t.mu.Lock()
		t.last = st
		t.history = append(t.history, HistPoint{Games: t.games, Updates: st.Updates, EvalWin: mean(t.evalWins), Score: mean(t.scores), Entropy: st.Entropy})
		if len(t.history) > historyMax {
			t.history = t.history[len(t.history)-historyMax:]
		}
		t.mu.Unlock()
		if st.Updates%5 == 0 {
			t.save()
		}
	}
}

func (t *Trainer) save() {
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		log.Print(err)
		return
	}
	tmp := t.path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		log.Print(err)
		return
	}
	if err := t.Net().Save(f); err != nil {
		f.Close()
		log.Print(err)
		return
	}
	f.Close()
	os.Rename(tmp, t.path)
}

type TrainerStats struct {
	Running   bool
	Games     int
	Updates   int
	GamesPerS float64
	EvalWin   float64
	EvalN     int
	Score     float64
	Duration  float64
	Last      rl.UpdateStats
	History   []HistPoint
}

func (t *Trainer) Stats() TrainerStats {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := TrainerStats{
		Running: t.running, Games: t.games, Updates: t.Net().Updates(), EvalWin: mean(t.evalWins), EvalN: len(t.evalWins),
		Score: mean(t.scores), Duration: mean(t.durations), Last: t.last,
		History: append([]HistPoint(nil), t.history...),
	}
	if t.running {
		if el := time.Since(t.startT).Seconds(); el > 0 {
			s.GamesPerS = float64(t.games-t.startG) / el
		}
	}
	return s
}

// trainTimeLimit is how long a training game may run, in seconds. Larger
// maps get longer games because armies take longer to cross them.
func trainTimeLimit() float64 { return 20 * 60 * max(1, float64(MapW)/96) }
