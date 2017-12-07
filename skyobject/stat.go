package skyobject

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject/statutil"
)

// A Stat represents statistic of the Container
type Stat struct {

	// CXDS is statistic of CXDS key-value store
	CXDS ReadWriteStat
	// Cache contains read-write statistic of the
	// Cache
	Cache ReadWriteStat
	// CacheCleaning is average pause of Cache
	// for cleaning
	CacheCleaning time.Duration

	// Amount is amount of obejcts in the CXDS
	// key-value store
	Amount statutil.Amount
	// Volume is total volume of values (not keys,
	// and values) in the CXDS key-vlaue store
	Volume statutil.Volume

	// RootsPerSecond is average vlaue of new
	// Root obejcts per second
	RootsPerSecond float64

	// Feeds contains statistic of feeds
	Feeds map[cipher.PubKey]FeedStat
}

// A ReadWriteStat represents read-write statistic
type ReadWriteStat struct {
	RPS float64 // reads per second
	WPS float64 // writes per second
}

// A FeedsTat represents statistic
// of a feed
type FeedStat struct {
	// Hads contains statistic of heads
	Heads map[uint64]HeadStat
}

// A HeadStat represents statistic of
// a head
type HeadStat struct {
	// Roots is total amount of Root
	// obejcts of this head
	Roots int

	// First is timestamp of first
	// Root in the head
	First time.Time
	// Last is timestamp of last
	// Root in the head
	Last time.Time
}

// Stat retusn statistic of the Container
func (c *Container) Stat() (s *Stat) {

	s = new(Stat)

	s.CXDS.RPS = c.Cache.stat.dbRPS()
	s.CXDS.WPS = c.Cache.stat.dbWPS()

	s.Cache.RPS = c.Cache.stat.cRPS()
	s.Cache.WPS = c.Cache.stat.cWPS()

	s.CacheCleaning = c.Cache.stat.cacheCleaning()

	s.Amount = statutil.Amount(c.Cache.stat.amount)
	s.Volume = statutil.Volume(c.Cache.stat.volume)

	s.RootsPerSecond = c.Index.stat.rootsPerSecond()

	s.Feeds = c.Index.feedsStat()

	return
}

func (i *Index) feedsStat() (s map[cipher.PubKey]FeedStat) {

	i.mx.Lock()
	defer i.mx.Unlock()

	s = make(map[cipher.PubKey]FeedStat)

	for pk, hs := range i.feeds {

		var sf FeedStat

		sf.Heads = make(map[uint64]HeadStat)

		for nonce, rs := range hs.h {

			var sh HeadStat

			for i, r := range rs {

				sh.Roots++

				if i == 0 {

					sh.First = time.Unix(0, r.Time)

				} else if i == len(rs)-1 {

					sh.Last = time.Unix(0, r.Time)

				}

			}

			sf.Heads[nonce] = sh

		}

		s[pk] = sf

	}

	return

}
