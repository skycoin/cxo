package skyobject

import (
	"time"

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

	Save    time.Duration // average saving time
	CleanUp time.Duration // average cleaning time (not implemented yet)
	Stat    time.Duration // duration of the Stat collecting
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

type objStat struct {
	rc  uint32
	vol data.Volume
}

// Stat returns detailed statiscit. The call requires
// iterating over all objects. Thus, the call is slow
func (c *Container) Stat() (s Stat, err error) {

	tp := time.Now()

	// s.Object, s.Shared

	objs := make(map[cipher.SHA256]objStat)

	err = c.DB().CXDS().Iterate(func(key cipher.SHA256, rc uint32) (err error) {
		val, _, _ := c.DB().CXDS().Get(key)
		objs[key] = objStat{rc, data.Volume(len(val))}
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

	s.Feeds = make(map[cipher.PubKey]FeedStat)

	err = c.DB().IdxDB().Tx(func(feeds data.Feeds) (err error) {

		// range over all feeds
		return feeds.Iterate(func(pk cipher.PubKey) (err error) {

			var roots data.Roots
			if roots, err = feeds.Roots(pk); err != nil {
				return
			}

			var fs FeedStat
			if fs, err = c.getFeedStat(roots, objs); err != nil {
				return
			}

			s.Feeds[pk] = fs
			return
		})

	})

	if err != nil {
		return
	}

	s.Save = c.packSave.Value()
	s.CleanUp = c.cleanUp.Value()
	s.Stat = time.Now().Sub(tp)
	return
}

func (c *Container) getFeedStat(roots data.Roots,
	objs map[cipher.SHA256]objStat) (fs FeedStat, err error) {

	fs.Roots = make(map[uint64]RootStat)

	err = roots.Ascend(func(ir *data.Root) (err error) {
		var rs RootStat
		if rs, err = c.getRootStat(ir, objs); err != nil {
			return
		}
		fs.Objects.Amount += rs.Objects.Amount
		fs.Objects.Volume += rs.Objects.Volume
		fs.Shared.Amount += rs.Shared.Amount
		fs.Shared.Volume += rs.Shared.Volume
		return
	})
	if err != nil {
		return
	}
	return
}

func (c *Container) getRootStat(ir *data.Root,
	objs map[cipher.SHA256]objStat) (rs RootStat, err error) {

	err = c.findRefs(ir, func(key cipher.SHA256) (deepper bool, _ error) {

		deepper = true

		o := objs[key]

		rs.Objects.Amount++
		rs.Objects.Volume += o.vol
		if o.rc > 1 {
			rs.Shared.Amount++
			rs.Shared.Volume += o.vol
		}

		return
	})
	return
}
