package rl

import (
	"math"
	"math/rand"
	"runtime"
	"sync"
)

// Step is one decision of an agent.
type Step struct {
	Obs    []float64
	Mask   []bool
	Action int
	LogP   float64 // log-prob under the behaviour policy at collection time
	Value  float64 // value estimate at collection time
	Reward float64 // reward received after taking Action
}

// Episode is one agent's trajectory. Done=false means it was truncated, and
// BootstrapValue is used as the value of the state after the last step.
type Episode struct {
	Steps          []Step
	Done           bool
	BootstrapValue float64
}

// UpdateStats reports averages over every minibatch of one Update.
type UpdateStats struct {
	PolicyLoss, ValueLoss, Entropy, ApproxKL, ClipFrac, ExplainedVar float64
	Steps, Updates                                                   int
}

const (
	adamBeta1 = 0.9
	adamBeta2 = 0.999
	adamEps   = 1e-5
	maxWorker = 8
)

// Learner owns the trainable network and Adam state. Not safe for concurrent use.
type Learner struct {
	net     *Net
	m, v    []float64
	grad    []float64
	t       int
	workers []*worker
}

type sample struct {
	obs     []float64
	mask    []bool
	action  int
	oldLogP float64
	adv     float64
	ret     float64
}

type accum struct {
	loss, policy, value, entropy, kl, clipped float64
	n                                         int
}

func (a *accum) add(b accum) {
	a.loss += b.loss
	a.policy += b.policy
	a.value += b.value
	a.entropy += b.entropy
	a.kl += b.kl
	a.clipped += b.clipped
	a.n += b.n
}

// worker holds per-goroutine buffers for the backward pass.
type worker struct {
	acts    [][]float64 // acts[0] is the observation, acts[i+1] hidden layer i
	logits  []float64
	probs   []float64
	dlogits []float64
	dh, dx  []float64
	grad    []float64
	st      accum
}

// NewLearner takes ownership of net; callers must not use net afterwards
// except through the Learner.
func NewLearner(net *Net) *Learner {
	p := len(net.params)
	l := &Learner{
		net:  net,
		m:    make([]float64, p),
		v:    make([]float64, p),
		grad: make([]float64, p),
	}
	nw := min(runtime.GOMAXPROCS(0), maxWorker)
	for range nw {
		l.workers = append(l.workers, newWorker(net))
	}
	return l
}

func newWorker(n *Net) *worker {
	w := &worker{
		acts:    make([][]float64, len(n.hidden)+1),
		logits:  make([]float64, n.cfg.Actions),
		probs:   make([]float64, n.cfg.Actions),
		dlogits: make([]float64, n.cfg.Actions),
		grad:    make([]float64, len(n.params)),
	}
	width := 1
	for i, d := range n.hidden {
		w.acts[i+1] = make([]float64, d.out)
		width = max(width, d.out)
	}
	w.dh = make([]float64, width)
	w.dx = make([]float64, width)
	return w
}

// Net returns the network being trained. It changes during Update, so give
// workers a Snapshot instead.
func (l *Learner) Net() *Net { return l.net }

// Snapshot returns an independent copy of the current weights for workers.
func (l *Learner) Snapshot() *Net { return l.net.Clone() }

// Update runs PPO on the episodes and returns stats.
func (l *Learner) Update(eps []Episode, rng *rand.Rand) UpdateStats {
	cfg := l.net.cfg
	samples, values := buildSamples(eps, cfg.Gamma, cfg.Lambda)
	if len(samples) == 0 {
		return UpdateStats{Updates: l.net.updates}
	}
	ev := explainedVariance(values, samples)
	normalizeAdvantages(samples)

	mb := cfg.Minibatch
	if mb <= 0 || mb > len(samples) {
		mb = len(samples)
	}
	batch := make([]sample, 0, mb)
	var total accum
	for range max(cfg.Epochs, 1) {
		perm := rng.Perm(len(samples))
		for start := 0; start < len(perm); start += mb {
			batch = batch[:0]
			for _, i := range perm[start:min(start+mb, len(perm))] {
				batch = append(batch, samples[i])
			}
			total.add(l.gradient(batch))
			clipNorm(l.grad, cfg.MaxGradNorm)
			l.adam(cfg.LR)
		}
	}
	l.net.updates++

	n := float64(total.n)
	return UpdateStats{
		PolicyLoss:   total.policy / n,
		ValueLoss:    total.value / n,
		Entropy:      total.entropy / n,
		ApproxKL:     total.kl / n,
		ClipFrac:     total.clipped / n,
		ExplainedVar: ev,
		Steps:        len(samples),
		Updates:      l.net.updates,
	}
}

// buildSamples computes GAE advantages and returns targets. It also returns
// the collection-time values, aligned with the samples.
func buildSamples(eps []Episode, gamma, lambda float64) ([]sample, []float64) {
	var n int
	for _, e := range eps {
		n += len(e.Steps)
	}
	samples := make([]sample, n)
	values := make([]float64, n)
	off := 0
	for _, e := range eps {
		next := e.BootstrapValue
		if e.Done {
			next = 0
		}
		var gae float64
		for t := len(e.Steps) - 1; t >= 0; t-- {
			s := &e.Steps[t]
			delta := s.Reward + gamma*next - s.Value
			gae = delta + gamma*lambda*gae
			samples[off+t] = sample{
				obs: s.Obs, mask: s.Mask, action: s.Action, oldLogP: s.LogP,
				adv: gae, ret: gae + s.Value,
			}
			values[off+t] = s.Value
			next = s.Value
		}
		off += len(e.Steps)
	}
	return samples, values
}

func explainedVariance(values []float64, samples []sample) float64 {
	var mr, md float64
	for i, s := range samples {
		mr += s.ret
		md += s.ret - values[i]
	}
	k := float64(len(samples))
	mr /= k
	md /= k
	var vr, vd float64
	for i, s := range samples {
		vr += (s.ret - mr) * (s.ret - mr)
		d := s.ret - values[i] - md
		vd += d * d
	}
	if vr == 0 {
		return 0
	}
	return 1 - vd/vr
}

func normalizeAdvantages(samples []sample) {
	var mean float64
	for _, s := range samples {
		mean += s.adv
	}
	mean /= float64(len(samples))
	var v float64
	for _, s := range samples {
		v += (s.adv - mean) * (s.adv - mean)
	}
	std := math.Sqrt(v/float64(len(samples))) + 1e-8
	for i := range samples {
		samples[i].adv = (samples[i].adv - mean) / std
	}
}

// gradient writes the mean loss gradient over batch into l.grad, splitting
// the batch across workers. Chunks are summed in a fixed order, so the
// result does not depend on scheduling.
func (l *Learner) gradient(batch []sample) accum {
	nw := min(len(l.workers), (len(batch)+31)/32)
	chunk := (len(batch) + nw - 1) / nw
	scale := 1 / float64(len(batch))
	var wg sync.WaitGroup
	for i := range nw {
		lo, hi := i*chunk, min((i+1)*chunk, len(batch))
		w := l.workers[i]
		wg.Go(func() { w.run(l.net, batch[lo:hi], scale) })
	}
	wg.Wait()
	clear(l.grad)
	var st accum
	for _, w := range l.workers[:nw] {
		axpy(1, w.grad, l.grad)
		st.add(w.st)
	}
	return st
}

// run accumulates scale times the gradient of the PPO loss over batch into
// w.grad, which it clears first, and records unscaled loss terms in w.st.
func (w *worker) run(n *Net, batch []sample, scale float64) {
	clear(w.grad)
	w.st = accum{}
	cfg := n.cfg
	p, g := n.params, w.grad
	for i := range batch {
		s := &batch[i]
		value := w.forward(n, s.obs)
		logZ, ok := maskedSoftmax(w.logits, s.mask, w.probs)

		var logp, ent float64
		if ok {
			logp = w.logits[s.action] - logZ
			for j, pj := range w.probs {
				if pj > 0 {
					ent -= pj * (w.logits[j] - logZ)
				}
			}
		}
		ratio := math.Exp(logp - s.oldLogP)
		clipped := math.Max(1-cfg.Clip, math.Min(1+cfg.Clip, ratio))
		s1, s2 := ratio*s.adv, clipped*s.adv
		var pl, dLogp float64
		if s1 <= s2 {
			pl, dLogp = -s1, -s.adv*ratio
		} else {
			pl = -s2
		}
		dv := value - s.ret
		vl := 0.5 * dv * dv
		w.st.loss += pl + cfg.ValueCoef*vl - cfg.EntropyCoef*ent
		w.st.policy += pl
		w.st.value += vl
		w.st.entropy += ent
		w.st.kl += (ratio - 1) - (logp - s.oldLogP)
		if math.Abs(ratio-1) > cfg.Clip {
			w.st.clipped++
		}
		w.st.n++

		// d(-ent)/dz_j = p_j*(log p_j + H); d(logp)/dz_j = 1[j=a] - p_j.
		for j, pj := range w.probs {
			if !ok || pj == 0 {
				w.dlogits[j] = 0
				continue
			}
			ind := 0.0
			if j == s.action {
				ind = 1
			}
			lpj := w.logits[j] - logZ
			w.dlogits[j] = scale * (dLogp*(ind-pj) + cfg.EntropyCoef*pj*(lpj+ent))
		}
		dvs := scale * cfg.ValueCoef * dv

		hL := w.acts[len(n.hidden)]
		pol, val := n.pol, n.val
		for o, d := range w.dlogits {
			if d == 0 {
				continue
			}
			g[pol.b+o] += d
			axpy(d, hL, g[pol.w+o*pol.in:])
		}
		g[val.b] += dvs
		axpy(dvs, hL, g[val.w:])
		if len(n.hidden) == 0 {
			continue
		}

		dh := w.dh[:len(hL)]
		for k := range dh {
			dh[k] = dvs * p[val.w+k]
		}
		for o, d := range w.dlogits {
			if d != 0 {
				axpy(d, p[pol.w+o*pol.in:pol.w+(o+1)*pol.in], dh)
			}
		}
		for li := len(n.hidden) - 1; li >= 0; li-- {
			d := n.hidden[li]
			h, x := w.acts[li+1], w.acts[li]
			dpre := dh[:d.out]
			for o, hv := range h {
				dpre[o] *= 1 - hv*hv
			}
			var dx []float64
			if li > 0 {
				dx = w.dx[:d.in]
				clear(dx)
			}
			for o, dp := range dpre {
				if dp == 0 {
					continue
				}
				row := d.w + o*d.in
				g[d.b+o] += dp
				axpy(dp, x, g[row:row+d.in])
				if dx != nil {
					axpy(dp, p[row:row+d.in], dx)
				}
			}
			if dx != nil {
				w.dh, w.dx = w.dx, w.dh
				dh = dx
			}
		}
	}
}

// forward is Net.forward that keeps every activation for the backward pass.
func (w *worker) forward(n *Net, obs []float64) float64 {
	w.acts[0] = obs
	x := obs
	for i, d := range n.hidden {
		y := w.acts[i+1]
		d.forward(n.params, x, y)
		for j, v := range y {
			y[j] = math.Tanh(v)
		}
		x = y
	}
	n.pol.forward(n.params, x, w.logits)
	return n.params[n.val.b] + dot(n.params[n.val.w:n.val.b], x)
}

func clipNorm(g []float64, maxNorm float64) {
	if maxNorm <= 0 {
		return
	}
	norm := math.Sqrt(dot(g, g))
	if norm > maxNorm {
		s := maxNorm / norm
		for i := range g {
			g[i] *= s
		}
	}
}

func (l *Learner) adam(lr float64) {
	l.t++
	c1 := 1 - math.Pow(adamBeta1, float64(l.t))
	c2 := 1 - math.Pow(adamBeta2, float64(l.t))
	p := l.net.params
	for i, g := range l.grad {
		l.m[i] = adamBeta1*l.m[i] + (1-adamBeta1)*g
		l.v[i] = adamBeta2*l.v[i] + (1-adamBeta2)*g*g
		p[i] -= lr * (l.m[i] / c1) / (math.Sqrt(l.v[i]/c2) + adamEps)
	}
}
