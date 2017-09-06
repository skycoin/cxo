package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

// A Stat represents detailed DB statistic
type Stat struct {
	// Objects is total amount of all objects
	Objects ObjectsStat
	// Shared objects is part of the total objects
	// that used by many Root obejcts
	Shared ObjectsStat

	// Feeds contains detailed statistic per feed
	Feeds map[cipher.PubKey]FeedStat
}

// An ObjectsStat represents objects DB statistic
type ObjectsStat struct {
	Volume data.Volume // size
	Amount uint32      // amount
}

// A FeedStat represetns detailed
// statistic about a feed
type FeedStat struct {
	// Objects is amount of all obejcts used by
	// this feed
	Objects ObjectsStat
	// Shared objects if amount of objects
	// used by many Root objects of this feed
	Shared ObjectsStat
	// Roots contains detailed statistic
	// for every Root of this feed
	Roots map[uint64]RootStat
}

// A RootStat represents detailed statistic
// of a Root object
type RootStat struct {
	// Total objects used by this Root
	Objects ObjectsStat
	// Shared is part of the total amount.
	// It is amount of objects used by many
	// Root objects of this feed
	Shared ObjectsStat
}

// Stat returns detailed statiscit. The call requires
// iterating over all objects. Thus, the call is slow
func (c *Container) Stat() (s Stat, err error) {

	type object struct {
		rc  uint32
		vol data.Volume
	}

	// s.Object, s.Shared

	objs := make(map[cipher.SHA256]object)

	err = c.DB().CXDS().Iterate(func(key cipher.SHA256, rc uint32) (err error) {
		val, _, _ := c.DB().CXDS().Get(key)
		objs[key] = object{rc, data.Volume(len(vol))}
		return
	})
	if err != nil {
		return
	}

	for _, obj := range objs {
		s.Objects.Amount++
		s.Objects.Volume += obj.vol
		if obj.rc > 1 {
			s.Shared.Amount++
			s.Shared.Volume += obj.vol
		}
	}

	// s.Feeds

	return
}
