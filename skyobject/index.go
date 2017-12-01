package skyobject

import (
	"sort"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

type indexRoot struct {
	dr   data.Root
	sync bool // sync with DB (AccessTime)
	hold int  // holds
}

type idx struct {
	mx    sync.Mutex
	feeds map[cipher.PubKey]map[uint64][]*indexRoot
}

func (c *Container) loadIdx() (err error) {
	c.idx.feeds = make(map[cipher.PubKey]map[uint64][]*indexRoot)

	err = c.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {

		// range feeds

		return feeds.Iterate(func(pk cipher.PubKey) (err error) {

			var hs data.Heads
			if hs, err = feeds.Heads(pk); err != nil {
				return
			}

			// range heads

			var feedMap = make(map[uint64][]*indexRoot)

			err = hs.Iterate(func(nonce uint64) (err error) {

				var rs data.Roots
				if rs, err = hs.Roots(nonce); err != nil {
					return
				}

				// range roots

				var headSlice = make([]*indexRoot, 0)

				err = rs.Ascend(func(dr *data.Root) (err error) {

					headSlice = append(headSlice, &indexRoot{
						dr:   *dr,
						sync: true,
						hold: 0,
					})

					return
				})

				if err != nil {
					return
				}

				feedMap[nonce] = headSlice

				return
			})

			if err != nil {
				return
			}

			c.idx.feeds[pk] = feedMap

			return
		})

	})

	return
}

func copyDataRoot(dr *data.Root) (cp *data.Root) {
	cp = new(data.Root)
	*cp = *dr
	return
}
