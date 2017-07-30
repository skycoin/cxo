package skyobject

import (
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

// A Stat represents Container statistic
type Stat struct {
	Registries int           // amount of unpacked registries
	Save       time.Duration // avg time of pack.Save() call
	CleanUp    time.Duration // avg time of c.CleanUp() call
}

// rolling average of duration
type rollavg func(time.Duration) time.Duration

// statistic
type stat struct {
	mx sync.Mutex

	Registries int // amount of unpacked registries

	Save    time.Duration // avg time of pack.Save() call
	CleanUp time.Duration // avg time of c.CleanUp() call

	//
	// rolling averages
	//

	packSave rollavg // avg time of pack.Save() call
	cleanUp  rollavg // avg time of c.CleanUp() call (GC)
}

func (s *stat) init(samples int) {
	if samples <= 0 {
		samples = StatSamples // default
	}
	s.packSave = rolling(samples)
	s.cleanUp = rolling(samples)
}

func (s *stat) addPackSave(dur time.Duration) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.Save = s.packSave(dur)
}

func (s *stat) addCleanUp(dur time.Duration) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.CleanUp = s.cleanUp(dur)
}

func (s *stat) addRegistry(delta int) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.Registries += delta
}

// rolling returns function that calculates rolling
// averaage (moving average) using n samples
func rolling(n int) rollavg {

	var (
		bins    = make([]float64, n)
		average float64
		i, c    int
	)

	return func(d time.Duration) time.Duration {
		if c < n {
			c++
		}
		x := float64(d)
		average += (x - bins[i]) / float64(c)
		bins[i] = x
		i = (i + 1) % n
		return time.Duration(average)
	}

}

// informational copy of the stat
func (s *stat) Stat() (cp Stat) {
	s.mx.Lock()
	defer s.mx.Unlock()

	cp.Registries = s.Registries
	cp.Save = s.Save
	cp.CleanUp = s.CleanUp
	return
}

// Stat of Container
func (c *Container) Stat() Stat {
	return c.stat.Stat()
}
