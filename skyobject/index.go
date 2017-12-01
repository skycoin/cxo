package skyobject

import (
	"sort"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

type indexRoot struct {
	data.Root
	sync bool // sync with DB (AccessTime)
	hold int  // holds
}

// Index is internal and used by Container.
// The Index can't be creaed and used outside
type Index struct {
	mx    sync.Mutex
	feeds map[cipher.PubKey]map[uint64][]*indexRoot
}

func (i *Index) load(db data.IdxDB) (err error) {
	i.feeds = make(map[cipher.PubKey]map[uint64][]*indexRoot)

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
						Root: *dr,  // copy
						sync: true, // synchronous
						hold: 0,    // not held
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

			i.feeds[pk] = feedMap

			return
		})

	})

	return
}

// call under lock
func (i *Index) selectRoot(
	pk cipher.PubKey,
	nonce uint64,
	seq uint64,
) (
	ir *indexRoot,
	err error,
) {

	var hs, ok = i.feeds[pk]

	if ok == true {
		return nil, data.ErrNoSuchFeed
	}

	var rs []*indexRoot
	if rs, ok = hs[nonce]; ok == false {
		return nil, data.ErrNoSuchHead
	}

	var k = sort.Search(len(rs), func(i int) bool {
		rs[i].Seq >= seq
	})

	ir = rs[k]

	if ir.Seq != sqe {
		return nil, data.ErrNotFound
	}

	return

}

// HoldRoot used to avoid removing of a
// Root object with all related objects.
// A Root can be used for end-user needs
// and also to share it with other nodes.
// Thus, in some cases a Root can't be
// removed, and should be held. It's
// impossible to hold a Root if it
// doesn't exist in IdxDB
func (i *Index) HoldRoot(
	pk cipher.PubKey, // : feed of the Root
	nonce uint64, //     : nonce of head of the Root
	seq uint64, //       : seq of the Root to hold
) (
	err error, //        : an error
) {

	i.mx.Lock()
	defer i.mx.Unlock()

	var ir *indexRoot
	if ir, err = i.selectRoot(pk, nonce, seq); err != nil {
		return
	}

	ir.hold++
	return

}

// UnholdRoot used to unhold previously
// held Root object. The UnholdRoot
// returns error if Root doesn't exist
// or was not held
func (i *Index) UnholdRoot(
	pk cipher.PubKey, // :
	nonce uint64, //     :
	seq uint64, //       :
) (
	err error,
) {

	i.mx.Lock()
	defer i.mx.Unlock()

	var ir *indexRoot
	if ir, err = i.selectRoot(pk, nonce, seq); err != nil {
		return
	}

	if ir.hold == 0 {
		return ErrRootIsNotHeld
	}

	ir.hold--
	return

}

// IsRootHeld returns true if Root with
// given feed, head, seq is held. The
// IsRootHeld method doesn't returns
// error if the Root doesn't exist
func (i *Index) IsRootHeld(
	pk cipher.PubKey, // :
	nonce uint64, //     :
	seq uint64, //       :
) (
	held bool, //        :
) {

	i.mx.Lock()
	defer i.mx.Unlock()

	var (
		ir  *indexRoot
		err error
	)

	if ir, err = i.selectRoot(pk, nonce, seq); err != nil {
		return // false
	}

	return ir.hold > 0
}
