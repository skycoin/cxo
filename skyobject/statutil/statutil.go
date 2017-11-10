package statutil

import (
	"sync"
	"time"
)

// RollAvg represents rolling (moving) average of time.Duration
type RollAvg struct {
	mx sync.Mutex

	last time.Duration
	roll func(time.Duration) time.Duration
}

// Value returns current average value
func (r *RollAvg) Value() time.Duration {
	r.mx.Lock()
	defer r.mx.Unlock()

	return r.last
}

// Add a time.Duration to the RollAvg. And get new
// average value back
func (r *RollAvg) Add(d time.Duration) (avg time.Duration) {
	r.mx.Lock()
	defer r.mx.Unlock()

	r.last = r.roll(d)
	return r.last
}

// AddStartTime is Add(time.Now().Sub(tp))
func (r *RollAvg) AddStartTime(tp time.Time) (avg time.Duration) {
	return r.Add(time.Now().Sub(tp))
}

// NewRollAvg creates new RollAvg using
// given amount of samples. It panics if
// the amount is zero or less
func NewRollAvg(n int) (ra RollAvg) {

	if n <= 0 {
		panic("number of samples is too small")
	}

	var (
		bins    = make([]float64, n)
		average float64
		i, c    int
		mx      sync.Mutex
	)

	ra.roll = func(d time.Duration) time.Duration {
		mx.Lock()
		defer mx.Unlock()

		if c < n {
			c++
		}

		x := float64(d)

		average += (x - bins[i]) / float64(c)
		bins[i] = x
		i = (i + 1) % n

		return time.Duration(average)
	}

	return
}
