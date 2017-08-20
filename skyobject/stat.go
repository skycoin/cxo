package skyobject

import (
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
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

// A DetailsStat represents detailed DB statistic
type DetailedStat struct {
	// Objects is total amount of all objects
	// including Registries
	Objects ObjectsStat `json:"objects"`
	// Shared objects is part of the total objects
	// that used by many Root obejcts
	Shared ObjectsStat `json:"shared"`
	// Stale objects is part of the total objects
	// that never used  but not removed by
	// CleanUp yet
	Stale ObjectsStat `json:"stale"`

	// Feeds contains detailed statistic per feed
	Feeds map[cipher.PubKey]FeedStat
}

// An ObjectsStat represents objects DB statistic
type ObjectsStat struct {
	Volume data.Volume `json:"volume"` // size
	Amount uint32      `json:"amount"` // amount
}

// A FeedStat represetns detailed
// statistic about a feed
type FeedStat struct {
	// Objects is amount of all obejcts used by
	// this feed
	Objects ObjectsStat `json:"objects"`
	// Shared objects if amount of objects
	// used by many Root objects of this feed
	Shared ObjectsStat `json:"shared"`

	// Volume is total size of all encoded
	// Root obejcts of this feed
	Volume data.Volume `json:"volume"`

	// Roots contains detailed statistic
	// for every Root of this feed
	Roots []RootStat `json:"roots"`
}

// A RootStat represents detaild statistic
// of a Root object
type RootStat struct {
	// Total objects used by this Root
	Objects ObjectsStat `json:"objects"`
	// Shared is part of the total amount.
	// It is amount of objects used by many
	// Root objects of this feed
	Shared ObjectsStat `json:"shared"`

	// Volume is size that this encoded Root
	// fits
	Volume data.Volume `json:"volume"`
}

func (c *Container) DetailedStat() (ds DetailedStat) {

	type Object struct {
		Volume data.Volume
		UsedBy int
	}

	objsStat := make(map[cipher.SHA256]Object)

	c.DB().View(func(tx data.Tv) (_ error) {

		objs := tx.Objects()

		ds.Objects.Amount = objs.Amount()
		ds.Objects.Volume = objs.Volume()

		objs.Ascend(func(key cipher.SHA256, val []byte) (_ error) {
			objsStat[key] = Object{Volume: data.Volume(len(val))}
			return
		})

		feeds := tx.Feeds()

		feeds.Ascend(func(pk cipher.PubKey) (_ error) {
			//
			return
		})

		return
	})
	return
}
