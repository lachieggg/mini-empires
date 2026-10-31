// Package rl implements a small actor-critic network and a PPO learner for
// training discrete-action agents by self-play, using only the standard library.
package rl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sync"
)

// Config holds the network shape and the PPO hyperparameters.
type Config struct {
	Obs, Actions           int
	Hidden                 []int   // e.g. []int{128, 64}, tanh activations
	LR                     float64 // Adam learning rate, e.g. 3e-4
	Gamma, Lambda          float64 // GAE, e.g. 0.995, 0.95
	Clip                   float64 // PPO clip, e.g. 0.2
	Epochs                 int     // PPO epochs per update, e.g. 4
	Minibatch              int     // e.g. 256
	EntropyCoef, ValueCoef float64 // e.g. 0.01, 0.5
	MaxGradNorm            float64 // e.g. 0.5
}

// DefaultConfig returns a configuration suited to the macro RTS use case.
func DefaultConfig(obs, actions int) Config {
	return Config{
		Obs:         obs,
		Actions:     actions,
		Hidden:      []int{128, 64},
		LR:          3e-4,
		Gamma:       0.995,
		Lambda:      0.95,
		Clip:        0.2,
		Epochs:      4,
		Minibatch:   256,
		EntropyCoef: 0.01,
		ValueCoef:   0.5,
		MaxGradNorm: 0.5,
	}
}

func (c Config) validate() error {
	if c.Obs <= 0 || c.Actions <= 0 {
		return fmt.Errorf("rl: invalid config: Obs=%d Actions=%d", c.Obs, c.Actions)
	}
	for _, h := range c.Hidden {
		if h <= 0 {
			return fmt.Errorf("rl: invalid hidden layer size %d", h)
		}
	}
	return nil
}

// dense is a fully connected layer stored in a flat parameter slice: an
// out x in row-major weight matrix at offset w and a bias vector at offset b.
type dense struct{ in, out, w, b int }

func (d dense) forward(p, x, y []float64) {
	for o := 0; o < d.out; o++ {
		row := d.w + o*d.in
		y[o] = p[d.b+o] + dot(p[row:row+d.in], x)
	}
}

// Net is an actor-critic MLP: shared hidden layers, a policy head (logits per
// action) and a value head. Forward, Act, Greedy and Probs are safe for
// concurrent use; the weights must not change while they run.
type Net struct {
	cfg     Config
	hidden  []dense
	pol     dense
	val     dense
	params  []float64
	updates int
	pool    sync.Pool
}

type scratch struct {
	h      [][]float64
	logits []float64
	probs  []float64
}

func layout(cfg Config) (hidden []dense, pol, val dense, n int) {
	in := cfg.Obs
	add := func(out int) dense {
		d := dense{in: in, out: out, w: n, b: n + in*out}
		n += in*out + out
		in = out
		return d
	}
	for _, h := range cfg.Hidden {
		hidden = append(hidden, add(h))
	}
	last := in
	pol = add(cfg.Actions)
	in = last
	val = add(1)
	return hidden, pol, val, n
}

func newNet(cfg Config) *Net {
	if err := cfg.validate(); err != nil {
		panic(err)
	}
	cfg.Hidden = append([]int(nil), cfg.Hidden...)
	n := &Net{cfg: cfg}
	var size int
	n.hidden, n.pol, n.val, size = layout(cfg)
	n.params = make([]float64, size)
	n.pool.New = func() any {
		s := &scratch{
			h:      make([][]float64, len(n.hidden)),
			logits: make([]float64, cfg.Actions),
			probs:  make([]float64, cfg.Actions),
		}
		for i, d := range n.hidden {
			s.h[i] = make([]float64, d.out)
		}
		return s
	}
	return n
}

// NewNet returns a network with orthogonal weights and zero biases. The policy
// head is scaled by 0.01 so the initial policy is close to uniform.
func NewNet(cfg Config, seed int64) *Net {
	n := newNet(cfg)
	rng := rand.New(rand.NewSource(seed))
	for _, d := range n.hidden {
		orthogonal(n.params[d.w:d.b], d.out, d.in, math.Sqrt2, rng)
	}
	orthogonal(n.params[n.pol.w:n.pol.b], n.pol.out, n.pol.in, 0.01, rng)
	orthogonal(n.params[n.val.w:n.val.b], n.val.out, n.val.in, 1, rng)
	return n
}

// orthogonal fills a rows x cols row-major matrix with a scaled
// (semi-)orthogonal matrix, using modified Gram-Schmidt on Gaussian vectors.
func orthogonal(w []float64, rows, cols int, gain float64, rng *rand.Rand) {
	long, short := rows, cols
	if cols > rows {
		long, short = cols, rows
	}
	vecs := make([][]float64, short)
	for k := range vecs {
		v := make([]float64, long)
		for {
			for i := range v {
				v[i] = rng.NormFloat64()
			}
			for _, u := range vecs[:k] {
				axpy(-dot(u, v), u, v)
			}
			if norm := math.Sqrt(dot(v, v)); norm > 1e-8 {
				for i := range v {
					v[i] /= norm
				}
				break
			}
		}
		vecs[k] = v
	}
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			if rows >= cols {
				w[i*cols+j] = gain * vecs[j][i]
			} else {
				w[i*cols+j] = gain * vecs[i][j]
			}
		}
	}
}

// Clone returns an independent deep copy.
func (n *Net) Clone() *Net {
	c := newNet(n.cfg)
	copy(c.params, n.params)
	c.updates = n.updates
	return c
}

// Config returns a copy of the network configuration.
func (n *Net) Config() Config {
	c := n.cfg
	c.Hidden = append([]int(nil), n.cfg.Hidden...)
	return c
}

// Updates returns how many learner updates produced these weights.
func (n *Net) Updates() int { return n.updates }

// NumParams returns the number of trainable parameters.
func (n *Net) NumParams() int { return len(n.params) }

// forward runs the network into s and returns the value estimate.
func (n *Net) forward(s *scratch, obs []float64) float64 {
	if len(obs) != n.cfg.Obs {
		panic(fmt.Sprintf("rl: obs has length %d, want %d", len(obs), n.cfg.Obs))
	}
	x := obs
	for i, d := range n.hidden {
		y := s.h[i]
		d.forward(n.params, x, y)
		for j, v := range y {
			y[j] = math.Tanh(v)
		}
		x = y
	}
	n.pol.forward(n.params, x, s.logits)
	return n.params[n.val.b] + dot(n.params[n.val.w:n.val.b], x)
}

// Forward returns the raw policy logits and the value estimate.
func (n *Net) Forward(obs []float64) (logits []float64, value float64) {
	logits = make([]float64, n.cfg.Actions)
	return logits, n.ForwardInto(obs, logits)
}

// ForwardInto is Forward without allocation: it writes the logits into
// logits, which must have length Actions.
func (n *Net) ForwardInto(obs, logits []float64) float64 {
	s := n.pool.Get().(*scratch)
	v := n.forward(s, obs)
	copy(logits, s.logits)
	n.pool.Put(s)
	return v
}

// Act samples from the masked softmax. mask[i]==false means action i is
// invalid; a nil mask allows every action. If every action is masked, it
// returns action 0 with logp 0.
func (n *Net) Act(obs []float64, mask []bool, rng *rand.Rand) (action int, logp, value float64) {
	s := n.pool.Get().(*scratch)
	defer n.pool.Put(s)
	value = n.forward(s, obs)
	logZ, ok := maskedSoftmax(s.logits, mask, s.probs)
	if !ok {
		return 0, 0, value
	}
	u := rng.Float64()
	action = -1
	for i, p := range s.probs {
		// Skipping zero-probability actions keeps a rounding shortfall in
		// the cumulative sum from landing on an underflowed action.
		if p == 0 || !valid(mask, i) {
			continue
		}
		action = i
		if u -= p; u < 0 {
			break
		}
	}
	return action, s.logits[action] - logZ, value
}

// Greedy returns the valid action with the highest logit, or 0 if every
// action is masked.
func (n *Net) Greedy(obs []float64, mask []bool) int {
	s := n.pool.Get().(*scratch)
	defer n.pool.Put(s)
	n.forward(s, obs)
	best := -1
	for i, l := range s.logits {
		if valid(mask, i) && (best < 0 || l > s.logits[best]) {
			best = i
		}
	}
	return max(best, 0)
}

// Probs returns the masked action probabilities.
func (n *Net) Probs(obs []float64, mask []bool) []float64 {
	s := n.pool.Get().(*scratch)
	defer n.pool.Put(s)
	n.forward(s, obs)
	out := make([]float64, n.cfg.Actions)
	maskedSoftmax(s.logits, mask, out)
	return out
}

func valid(mask []bool, i int) bool { return mask == nil || mask[i] }

// maskedSoftmax writes the softmax over valid actions into probs and returns
// log of the partition function. If no action is valid, probs is one-hot on
// action 0 and ok is false.
func maskedSoftmax(logits []float64, mask []bool, probs []float64) (logZ float64, ok bool) {
	if mask != nil && len(mask) != len(logits) {
		panic(fmt.Sprintf("rl: mask has length %d, want %d", len(mask), len(logits)))
	}
	m := math.Inf(-1)
	for i, l := range logits {
		if valid(mask, i) && l > m {
			m = l
		}
	}
	if math.IsInf(m, -1) {
		clear(probs)
		probs[0] = 1
		return 0, false
	}
	var sum float64
	for i, l := range logits {
		if valid(mask, i) {
			e := math.Exp(l - m)
			probs[i] = e
			sum += e
		} else {
			probs[i] = 0
		}
	}
	for i := range probs {
		probs[i] /= sum
	}
	return m + math.Log(sum), true
}

type netJSON struct {
	Config  Config    `json:"config"`
	Updates int       `json:"updates"`
	Params  []float64 `json:"params"`
}

// Save writes the configuration, the weights and the update counter as JSON.
func (n *Net) Save(w io.Writer) error {
	return json.NewEncoder(w).Encode(netJSON{Config: n.cfg, Updates: n.updates, Params: n.params})
}

// Load reads a network written by Save.
func Load(r io.Reader) (*Net, error) {
	var j netJSON
	if err := json.NewDecoder(r).Decode(&j); err != nil {
		return nil, fmt.Errorf("rl: decode net: %w", err)
	}
	if err := j.Config.validate(); err != nil {
		return nil, err
	}
	n := newNet(j.Config)
	if len(j.Params) != len(n.params) {
		return nil, errors.New("rl: parameter count does not match config")
	}
	copy(n.params, j.Params)
	n.updates = j.Updates
	return n, nil
}

func dot(a, b []float64) float64 {
	b = b[:len(a)]
	var s0, s1, s2, s3 float64
	i := 0
	for ; i+4 <= len(a); i += 4 {
		x, y := a[i:i+4:i+4], b[i:i+4:i+4]
		s0 += x[0] * y[0]
		s1 += x[1] * y[1]
		s2 += x[2] * y[2]
		s3 += x[3] * y[3]
	}
	for ; i < len(a); i++ {
		s0 += a[i] * b[i]
	}
	return s0 + s1 + s2 + s3
}

// axpy computes y += alpha*x.
func axpy(alpha float64, x, y []float64) {
	y = y[:len(x)]
	for i, v := range x {
		y[i] += alpha * v
	}
}
