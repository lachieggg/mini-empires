package rl

import (
	"bytes"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"testing"
)

func randObs(rng *rand.Rand, n int) []float64 {
	o := make([]float64, n)
	for i := range o {
		o[i] = rng.Float64()*3 - 1
	}
	return o
}

func randMask(rng *rand.Rand, n int) []bool {
	m := make([]bool, n)
	for i := range m {
		m[i] = rng.Intn(3) > 0
	}
	return m
}

func TestGradientCheck(t *testing.T) {
	cfg := DefaultConfig(7, 5)
	cfg.Hidden = []int{6, 4}
	cfg.EntropyCoef = 0.05
	net := NewNet(cfg, 1)
	rng := rand.New(rand.NewSource(2))
	// Larger policy weights give non-trivial probabilities and entropy terms.
	for i := net.pol.w; i < net.pol.b; i++ {
		net.params[i] = rng.NormFloat64() * 0.5
	}
	l := NewLearner(net)

	var batch []sample
	for range 12 {
		obs := randObs(rng, cfg.Obs)
		mask := randMask(rng, cfg.Actions)
		mask[rng.Intn(cfg.Actions)] = true
		a, logp, _ := net.Act(obs, mask, rng)
		batch = append(batch, sample{
			obs: obs, mask: mask, action: a,
			// Offsets push some ratios outside the clip range.
			oldLogP: logp + rng.NormFloat64()*0.3,
			adv:     rng.NormFloat64(),
			ret:     rng.NormFloat64(),
		})
	}

	w := l.workers[0]
	loss := func() float64 {
		w.run(net, batch, 1/float64(len(batch)))
		return w.st.loss / float64(len(batch))
	}
	loss()
	analytic := append([]float64(nil), w.grad...)

	const eps = 1e-6
	checked := 0
	for i := range net.params {
		orig := net.params[i]
		net.params[i] = orig + eps
		lp := loss()
		net.params[i] = orig - eps
		lm := loss()
		net.params[i] = orig
		num := (lp - lm) / (2 * eps)
		a := analytic[i]
		denom := math.Max(math.Abs(a)+math.Abs(num), 1e-7)
		if rel := math.Abs(a-num) / denom; rel > 1e-4 {
			t.Errorf("param %d: analytic %.10g numeric %.10g rel %.3g", i, a, num, rel)
		}
		checked++
	}
	t.Logf("checked %d of %d params", checked, len(net.params))
}

func TestMasking(t *testing.T) {
	cfg := DefaultConfig(10, 8)
	cfg.Hidden = []int{16}
	net := NewNet(cfg, 3)
	rng := rand.New(rand.NewSource(4))
	for i := net.pol.w; i < net.pol.b; i++ {
		net.params[i] = rng.NormFloat64()
	}
	for range 2000 {
		obs := randObs(rng, cfg.Obs)
		mask := randMask(rng, cfg.Actions)
		a, logp, _ := net.Act(obs, mask, rng)
		probs := net.Probs(obs, mask)
		anyValid := false
		var sum float64
		for i, p := range probs {
			if !mask[i] && p != 0 {
				t.Fatalf("masked action %d has prob %g", i, p)
			}
			anyValid = anyValid || mask[i]
			sum += p
		}
		if math.Abs(sum-1) > 1e-12 {
			t.Fatalf("probs sum to %g", sum)
		}
		if !anyValid {
			if a != 0 || logp != 0 {
				t.Fatalf("all masked: got action %d logp %g", a, logp)
			}
			continue
		}
		if !mask[a] {
			t.Fatalf("Act returned masked action %d", a)
		}
		if math.Abs(math.Exp(logp)-probs[a]) > 1e-12 {
			t.Fatalf("logp %g does not match prob %g", logp, probs[a])
		}
		if g := net.Greedy(obs, mask); !mask[g] {
			t.Fatalf("Greedy returned masked action %d", g)
		}
	}
	none := make([]bool, cfg.Actions)
	obs := randObs(rng, cfg.Obs)
	if a, _, _ := net.Act(obs, none, rng); a != 0 {
		t.Errorf("all masked Act = %d, want 0", a)
	}
	if g := net.Greedy(obs, none); g != 0 {
		t.Errorf("all masked Greedy = %d, want 0", g)
	}
}

func TestSaveLoad(t *testing.T) {
	cfg := DefaultConfig(12, 6)
	cfg.Hidden = []int{20, 10}
	l := NewLearner(NewNet(cfg, 5))
	rng := rand.New(rand.NewSource(6))
	ep := Episode{Done: true}
	for range 50 {
		obs := randObs(rng, cfg.Obs)
		a, logp, v := l.Net().Act(obs, nil, rng)
		ep.Steps = append(ep.Steps, Step{Obs: obs, Action: a, LogP: logp, Value: v, Reward: rng.NormFloat64()})
	}
	l.Update([]Episode{ep}, rng)
	net := l.Snapshot()

	var buf bytes.Buffer
	if err := net.Save(&buf); err != nil {
		t.Fatal(err)
	}
	got, err := Load(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Config(), net.Config()) {
		t.Errorf("config mismatch: %+v vs %+v", got.Config(), net.Config())
	}
	if got.Updates() != 1 {
		t.Errorf("Updates = %d, want 1", got.Updates())
	}
	for range 20 {
		obs := randObs(rng, cfg.Obs)
		l1, v1 := net.Forward(obs)
		l2, v2 := got.Forward(obs)
		if v1 != v2 || !reflect.DeepEqual(l1, l2) {
			t.Fatalf("outputs differ after load")
		}
	}
}

func TestConcurrentAct(t *testing.T) {
	cfg := DefaultConfig(60, 24)
	net := NewNet(cfg, 7)
	var wg sync.WaitGroup
	for g := range 16 {
		wg.Go(func() {
			rng := rand.New(rand.NewSource(int64(g)))
			for range 300 {
				obs := randObs(rng, cfg.Obs)
				mask := randMask(rng, cfg.Actions)
				a, _, _ := net.Act(obs, mask, rng)
				if a != 0 && !mask[a] {
					t.Errorf("masked action %d", a)
				}
				net.Greedy(obs, mask)
				net.Probs(obs, mask)
				net.Forward(obs)
			}
		})
	}
	wg.Wait()
}

// env is a toy episodic environment for the learning tests.
type env interface {
	reset(rng *rand.Rand) (obs []float64, mask []bool)
	step(a int) (obs []float64, mask []bool, r float64, done bool)
}

// delayedBandit shows a context only on the first step; the action taken
// there decides a terminal reward of +1 or -1 that arrives T-1 steps later.
// Filler steps pay a small immediate reward for action 5, which is masked
// on the first step.
type delayedBandit struct {
	k, actions, horizon int
	ctx, t              int
	correct             bool
}

func (e *delayedBandit) obs() ([]float64, []bool) {
	o := make([]float64, e.k+2)
	m := make([]bool, e.actions)
	if e.t == 0 {
		o[e.ctx] = 1
		o[e.k] = 1
		for i := range e.k {
			m[i] = true
		}
	} else {
		for i := range m {
			m[i] = true
		}
	}
	o[e.k+1] = float64(e.t) / float64(e.horizon)
	return o, m
}

func (e *delayedBandit) reset(rng *rand.Rand) ([]float64, []bool) {
	e.ctx, e.t, e.correct = rng.Intn(e.k), 0, false
	return e.obs()
}

func (e *delayedBandit) step(a int) ([]float64, []bool, float64, bool) {
	var r float64
	if e.t == 0 {
		e.correct = a == e.ctx
	} else if a == 5 {
		r = 0.1
	}
	e.t++
	if e.t == e.horizon {
		if e.correct {
			r++
		} else {
			r--
		}
		return nil, nil, r, true
	}
	o, m := e.obs()
	return o, m, r, false
}

// corridor starts at cell 0 and pays +1 for reaching the last cell, with a
// small cost per step. Actions 0 and 1 move left and right; 2 and 3 are
// always masked. Episodes are truncated after maxSteps.
type corridor struct {
	n, maxSteps, pos, t int
}

func (e *corridor) obs() ([]float64, []bool) {
	o := make([]float64, e.n)
	o[e.pos] = 1
	return o, []bool{true, true, false, false}
}

func (e *corridor) reset(*rand.Rand) ([]float64, []bool) {
	e.pos, e.t = 0, 0
	return e.obs()
}

func (e *corridor) step(a int) ([]float64, []bool, float64, bool) {
	if a == 1 {
		e.pos++
	} else if e.pos > 0 {
		e.pos--
	}
	e.t++
	if e.pos == e.n-1 {
		return nil, nil, 1 - 0.01, true
	}
	o, m := e.obs()
	return o, m, -0.01, false
}

func rollout(net *Net, e env, rng *rand.Rand, maxSteps int) (Episode, float64) {
	var ep Episode
	var ret float64
	obs, mask := e.reset(rng)
	for range maxSteps {
		a, logp, v := net.Act(obs, mask, rng)
		nobs, nmask, r, done := e.step(a)
		ep.Steps = append(ep.Steps, Step{Obs: obs, Mask: mask, Action: a, LogP: logp, Value: v, Reward: r})
		ret += r
		if done {
			ep.Done = true
			return ep, ret
		}
		obs, mask = nobs, nmask
	}
	_, ep.BootstrapValue = net.Forward(obs)
	return ep, ret
}

// train runs parallel collection on snapshots and returns the mean return
// of each iteration.
func train(t *testing.T, cfg Config, newEnv func() env, iters, episodes, maxSteps int) []float64 {
	l := NewLearner(NewNet(cfg, 11))
	rng := rand.New(rand.NewSource(12))
	var curve []float64
	const workers = 4
	for it := range iters {
		snap := l.Snapshot()
		eps := make([]Episode, episodes)
		rets := make([]float64, episodes)
		var wg sync.WaitGroup
		for w := range workers {
			wg.Go(func() {
				wrng := rand.New(rand.NewSource(int64(it*workers + w)))
				e := newEnv()
				for i := w; i < episodes; i += workers {
					eps[i], rets[i] = rollout(snap, e, wrng, maxSteps)
				}
			})
		}
		wg.Wait()
		var mean float64
		for _, r := range rets {
			mean += r
		}
		mean /= float64(episodes)
		curve = append(curve, mean)
		st := l.Update(eps, rng)
		if it%5 == 0 || it == iters-1 {
			t.Logf("iter %2d  return %+.3f  ent %.3f  kl %.4f  clip %.2f  ev %+.2f",
				it, mean, st.Entropy, st.ApproxKL, st.ClipFrac, st.ExplainedVar)
		}
	}
	return curve
}

func toyConfig(obs, actions int) Config {
	cfg := DefaultConfig(obs, actions)
	cfg.Hidden = []int{32, 32}
	cfg.LR = 3e-3
	cfg.Gamma = 0.99
	cfg.Minibatch = 128
	return cfg
}

func tail(c []float64, n int) float64 {
	var s float64
	for _, v := range c[len(c)-n:] {
		s += v
	}
	return s / float64(n)
}

func TestLearnDelayedBandit(t *testing.T) {
	// Random play returns about -0.5 + 7*0.1/6 = -0.38; optimal is 1.7.
	newEnv := func() env { return &delayedBandit{k: 4, actions: 6, horizon: 8} }
	curve := train(t, toyConfig(6, 6), newEnv, 60, 64, 8)
	if curve[0] > 0 {
		t.Errorf("initial return %.3f is not near random", curve[0])
	}
	if final := tail(curve, 5); final < 1.5 {
		t.Errorf("final return %.3f, want >= 1.5 (optimal 1.7)", final)
	}
}

func TestLearnCorridor(t *testing.T) {
	// Optimal return is 1 - 0.01*7 = 0.93; random play rarely reaches the goal.
	newEnv := func() env { return &corridor{n: 8} }
	curve := train(t, toyConfig(8, 4), newEnv, 40, 32, 40)
	if curve[0] > 0.3 {
		t.Errorf("initial return %.3f is not near random", curve[0])
	}
	if final := tail(curve, 5); final < 0.85 {
		t.Errorf("final return %.3f, want >= 0.85 (optimal 0.93)", final)
	}
}

func BenchmarkForward(b *testing.B) {
	net := NewNet(DefaultConfig(60, 24), 1)
	obs := randObs(rand.New(rand.NewSource(1)), 60)
	logits := make([]float64, 24)
	b.ReportAllocs()
	for b.Loop() {
		net.ForwardInto(obs, logits)
	}
}

func BenchmarkAct(b *testing.B) {
	net := NewNet(DefaultConfig(60, 24), 1)
	rng := rand.New(rand.NewSource(1))
	obs := randObs(rng, 60)
	mask := randMask(rng, 24)
	b.ReportAllocs()
	for b.Loop() {
		net.Act(obs, mask, rng)
	}
}

func BenchmarkActParallel(b *testing.B) {
	net := NewNet(DefaultConfig(60, 24), 1)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(1))
		obs := randObs(rng, 60)
		mask := randMask(rng, 24)
		for pb.Next() {
			net.Act(obs, mask, rng)
		}
	})
}

// BenchmarkUpdate measures one PPO update on 16 episodes of 400 steps.
func BenchmarkUpdate(b *testing.B) {
	cfg := DefaultConfig(60, 24)
	l := NewLearner(NewNet(cfg, 1))
	rng := rand.New(rand.NewSource(1))
	var eps []Episode
	for range 16 {
		ep := Episode{Done: true}
		for range 400 {
			obs := randObs(rng, cfg.Obs)
			mask := randMask(rng, cfg.Actions)
			a, logp, v := l.Net().Act(obs, mask, rng)
			ep.Steps = append(ep.Steps, Step{Obs: obs, Mask: mask, Action: a, LogP: logp, Value: v, Reward: rng.NormFloat64() * 0.1})
		}
		eps = append(eps, ep)
	}
	for b.Loop() {
		l.Update(eps, rng)
	}
}
