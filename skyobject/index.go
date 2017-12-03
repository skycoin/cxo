package skyobject

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject/registry"
)

type indexRoot struct {
	data.Root
	sync bool // sync with DB (AccessTime)
	hold int  // holds
}

// Index is internal and used by Container.
// The Index can't be creaed and used outside
type Index struct {
	mx sync.Mutex

	c      *Container // back reference (for db.IdxDB and for the Cache)
	feeds  map[cipher.PubKey]map[uint64][]*indexRoot
	feedsl []cipher.PubKey // change on write
}

func (i *Index) load(c *Container) (err error) {

	i.feeds = make(map[cipher.PubKey]map[uint64][]*indexRoot)
	i.c = c

	err = i.c.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {

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
func (i *Index) selectRoots(
	pk cipher.PubKey,
	nonce uint64,
) (
	rs []*indexRoot,
	err error,
) {

	var hs, ok = i.feeds[pk]

	if ok == true {
		return nil, data.ErrNoSuchFeed
	}

	if rs, ok = hs[nonce]; ok == false {
		return nil, data.ErrNoSuchHead
	}

	return
}

func searchRootInSortedSlice(
	rs []*indexRoot, // : the slice
	seq uint64, //      : seq of the wanted Root
) (
	ir *indexRoot, //   : found
	k int, //           : index of the Root in the slice
	err error, //       : not found
) {

	if len(rs) == 0 {
		return nil, 0, data.ErrNotFound
	}

	k = sort.Search(len(rs), func(i int) bool {
		return rs[i].Seq >= seq
	})

	ir = rs[k]

	if ir.Seq != seq {
		return nil, 0, data.ErrNotFound
	}

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

	var rs []*indexRoot
	if rs, err = i.selectRoots(pk, nonce); err != nil {
		return
	}

	ir, _, err = searchRootInSortedSlice(rs, seq)
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

// call it under lock
func (i *Index) isRootHeld(
	pk cipher.PubKey, //
	nonce uint64, //
	seq uint64, //
) (
	held bool, //
) {

	var (
		ir  *indexRoot
		err error
	)

	if ir, err = i.selectRoot(pk, nonce, seq); err != nil {
		return // false
	}

	return ir.hold > 0
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

	return i.isRootHeld(pk, nonce, seq)
}

//
//
//

// AddFeed adds feed
func (i *Index) AddFeed(pk cipher.PubKey) (err error) {

	i.mx.Lock()
	defer i.mx.Unlock()

	if _, ok := i.feeds[pk]; ok {
		return // alrady has
	}

	err = i.c.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {
		return feeds.Add(pk)
	})

	if err != nil {
		return
	}

	i.feedsl = i.feedsl[:0]

	if _, ok := i.feeds[pk]; ok == false {
		i.feeds[pk] = make(map[uint64][]*indexRoot) // add to Index
	}

	return
}

// ReceivedRoot called by the node package to
// check a recived root. The method verify hash
// and signature of the Root. The method also
// check database, may be DB already has this
// root
func (i *Index) ReceivedRoot(
	sig cipher.Sig,
	val []byte,
) (
	r *registry.Root,
	err error,
) {

	i.mx.Lock()
	defer i.mx.Unlock()

	var hash = cipher.SumSHA256(val)
	if err = cipher.VerifySignature(r.Pub, sig, hash); err != nil {
		return
	}

	if r, err = registry.DecodeRoot(val); err != nil {
		return
	}

	r.Hash = hash // set the hash
	r.Sig = sig   // set the signature

	if _, err = i.selectRoot(r.Pub, r.Nonce, r.Seq); err == nil {
		r.IsFull = true
		return
	} else if err == data.ErrNoSuchHead || err == data.ErrNotFound {
		err = nil // just not found (e.g. not full)
		return
	}

	return // data.ErrNoSuchFeed
}

// AddRoot to DB. The method doesn't create feed of the root
// but if head of the root doesn't exist, then the method
// creates the head. The method never return already have error
// and the method never save the Root inside CXDS. E.g. the
// method adds the Root to index (that is necessary)
func (i *Index) AddRoot(r *registry.Root) (err error) {

	i.mx.Lock()
	defer i.mx.Unlock()

	if r.IsFull == false {
		return errors.New("can't add non-full Root: " + r.Short())
	}

	if r.Pub == (cipher.PubKey{}) {
		return errors.New("blank public key of Root: " + r.Short())
	}

	if r.Hash == (cipher.SHA256{}) {
		return errors.New("blank hash of Root: " + r.Short())
	}

	if r.Sig == (cipher.Sig{}) {
		return errors.New("blank signature of Root: " + r.Short())
	}

	var dr *data.Root

	err = i.c.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {

		var hs data.Heads
		if hs, err = feeds.Heads(r.Pub); err != nil {
			return
		}

		var rs data.Roots
		if rs, err = hs.Add(r.Nonce); err != nil {
			return
		}

		// create data.Root

		dr = new(data.Root)
		dr.Seq = r.Seq
		dr.Prev = r.Prev
		dr.Hash = r.Hash
		dr.Sig = r.Sig

		// the Set never return "already exist" error

		return rs.Set(dr)
	})

	if err != nil {
		return
	}

	// if the Root already exist

	if ir, _ := i.selectRoot(r.Pub, r.Nonce, r.Seq); ir != nil {
		ir.AccessTime = time.Now().UnixNano()
		ir.sync = false
		return
	}

	// add to the Index

	var hs, ok = i.feeds[r.Pub]

	if ok == false {
		hs = make(map[uint64][]*indexRoot)
		i.feeds[r.Pub] = hs
	}

	var rs = hs[r.Nonce]

	rs = append(rs, &indexRoot{
		Root: *dr,
		sync: true,
		hold: 0,
	})

	sort.Slice(rs, func(i, j int) bool {
		return rs[i].Seq < rs[j].Seq
	})

	hs[r.Nonce] = rs

	return
}

// Heads returns list of heads of given feed.
// The list is list of nonces. One possible
// error is data.ErrNoSuchFeed
func (i *Index) Heads(pk cipher.PubKey) (heads []uint64, err error) {

	i.mx.Lock()
	defer i.mx.Unlock()

	var hs, ok = i.feeds[pk]

	if ok == false {
		return nil, data.ErrNoSuchFeed
	}

	heads = make([]uint64, 0, len(hs))

	for nonce := range hs {
		heads = append(heads, nonce)
	}

	return
}

// LastRoot returns last Root of given head of given feed.
// The last Root is Root with the greatest seq number. If
// given head of the feed is blank, then the LastRoot
// returns data.ErrNotFound. If the head does not exist
// then the LastRoot returns data.ErrNoSuchFeed
func (i *Index) LastRoot(
	pk cipher.PubKey, // : feed
	nonce uint64, //     : head
) (
	r *registry.Root, // : the Root
	err error, //        : an error
) {

	i.mx.Lock()
	defer i.mx.Unlock()

	var rs []*indexRoot
	if rs, err = i.selectRoots(pk, nonce); err != nil {
		return
	}

	if len(rs) == 0 {
		return nil, data.ErrNotFound
	}

	var ir = rs[len(rs)-1]

	r, err = i.c.RootByHash(ir.Hash) // using Cache
	return
}

// delFeedFromIndex deletes head from IdxDB and from the Index
func (i *Index) delFeedFromIndex(
	pk cipher.PubKey,
) (
	rss [][]*indexRoot,
	err error,
) {

	var hs, ok = i.feeds[pk]

	if ok == false {
		return nil, data.ErrNoSuchFeed
	}

	// a Root can be held

	rss = make([][]*indexRoot, 0, len(hs))

	for _, rs := range hs {

		for _, ir := range rs {
			if ir.hold > 0 {
				return nil, ErrRootIsHeld
			}
		}

		rss = append(rss, rs)

	}

	// delete from IdxDB first

	err = i.c.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {

		var heads data.Heads
		if heads, err = feeds.Heads(pk); err != nil {
			return
		}

		for nonce, rs := range hs {

			var roots data.Roots
			if roots, err = heads.Roots(nonce); err != nil {
				return
			}

			for _, ir := range rs {
				if err = roots.Del(ir.Seq); err != nil {
					return
				}
			}

			if err = heads.Del(nonce); err != nil {
				return
			}

		}

		return feeds.Del(pk)
	})

	if err != nil {
		return nil, err
	}

	// delete from the Index

	delete(i.feeds, pk)
	i.feedsl = i.feedsl[:0] // clear the list

	return
}

// with lock
func (i *Index) delFeedFromIndexLock(
	pk cipher.PubKey,
) (
	rss [][]*indexRoot,
	err error,
) {

	i.mx.Lock()
	defer i.mx.Unlock()

	return i.delFeedFromIndex(pk)
}

// DelFeed deletes feed. It can't remove feed if at least one
// Root of the feed is held, returning ErrRootIsHeld error
func (i *Index) DelFeed(pk cipher.PubKey) (err error) {

	// with lock
	var rss [][]*indexRoot
	if rss, err = i.delFeedFromIndexLock(pk); err != nil {
		return
	}

	// without lock
	for _, rs := range rss {
		for _, ir := range rs {
			if err = i.delRootRelatedValues(ir.Hash); err != nil {
				return
			}
		}
	}

	return
}

// delHeadFromIndex deletes head from IdxDB and from the Index
func (i *Index) delHeadFromIndex(
	pk cipher.PubKey,
	nonce uint64,
) (
	rs []*indexRoot,
	err error,
) {

	var hs, ok = i.feeds[pk]

	if ok == false {
		return nil, data.ErrNoSuchFeed
	}

	if rs, ok = hs[nonce]; ok == false {
		return nil, data.ErrNoSuchHead
	}

	// a Root can be held

	for _, ir := range rs {
		if ir.hold > 0 {
			return nil, ErrRootIsHeld
		}
	}

	// delete from IdxDB first

	err = i.c.db.IdxDB().Tx(func(feed data.Feeds) (err error) {

		var hs data.Heads
		if hs, err = feed.Heads(pk); err != nil {
			return
		}

		var roots data.Roots
		if roots, err = hs.Roots(nonce); err != nil {
			return
		}

		for _, ir := range rs {
			if err = roots.Del(ir.Seq); err != nil {
				return
			}
		}

		return hs.Del(nonce)
	})

	if err != nil {
		return nil, err
	}

	// delete from the Index

	delete(hs, nonce)

	return

}

// with lock
func (i *Index) delHeadFromIndexLock(
	pk cipher.PubKey,
	nonce uint64,
) (
	rs []*indexRoot,
	err error,
) {

	i.mx.Lock()
	defer i.mx.Unlock()

	return i.delHeadFromIndex(pk, nonce)
}

// DelHead deletes given head. It can't remove head if at least one
// Root of the head is held, returning ErrRootIsHeld error
func (i *Index) DelHead(pk cipher.PubKey, nonce uint64) (err error) {

	// with lock

	var rs []*indexRoot
	if rs, err = i.delHeadFromIndexLock(pk, nonce); err != nil {
		return
	}

	// without lock

	for _, ir := range rs {
		if err = i.delRootRelatedValues(ir.Hash); err != nil {
			return
		}
	}

	return
}

// delRoot removes Root from IdxDB and from the Index
// and returns the Root to decrement all related values;
// the delRoot must be called undex lock; but decrementing
// should be performed outside the lock to release the
// Index
func (i *Index) delRootFromIndex(
	pk cipher.PubKey, //       : feed
	nonce uint64, //           : head
	seq uint64, //             : seq
) (
	rootHash cipher.SHA256, // : hash of the Root
	err error, //              : an error
) {

	// don't remove if the Root is held

	if i.isRootHeld(pk, nonce, seq) == true {
		err = ErrRootIsHeld
		return
	}

	// find the Root in the Index first

	var hs, ok = i.feeds[pk]

	if ok == false {
		err = data.ErrNoSuchFeed
		return
	}

	var rs []*indexRoot
	if rs, ok = hs[nonce]; ok == false {
		err = data.ErrNoSuchHead
		return
	}

	var (
		ir *indexRoot
		k  int
	)

	if ir, k, err = searchRootInSortedSlice(rs, seq); err != nil {
		return // not found
	}

	// remove from IdxDB first

	err = i.c.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {

		var hs data.Heads
		if hs, err = feeds.Heads(pk); err != nil {
			return
		}

		var rs data.Roots
		if rs, err = hs.Roots(nonce); err != nil {
			return
		}

		return rs.Del(seq)
	})

	if err != nil {
		return // DB failure
	}

	// remove from the Index

	copy(rs[k:], rs[k+1:])
	rs[len(rs)-1] = nil // GC
	rs = rs[:len(rs)-1]

	if len(rs) == 0 {
		hs[nonce] = nil // delete slice (GC)
	} else {
		hs[nonce] = rs
	}

	return ir.Hash, nil
}

// delRootFromIndexLock is delRootFromIndex with lock
func (i *Index) delRootFromIndexLock(
	pk cipher.PubKey, //       : feed
	nonce uint64, //           : head
	seq uint64, //             : seq
) (
	rootHash cipher.SHA256, // : hash of the Root
	err error, //              : an error
) {

	i.mx.Lock()
	defer i.mx.Unlock()

	return i.delRootFromIndex(pk, nonce, seq)
}

func (i *Index) delPackWalkFunc(
	r *registry.Root, //           : Root
) (
	pack registry.Pack, //         : special Pack for deleting
	walkFunc registry.WalkFunc, // : walk deleting
	err error, //                  : an error
) {

	var dpack *delPack
	if dpack, err = i.c.getDelPack(r); err != nil {
		return
	}

	walkFunc = func(
		hash cipher.SHA256, // : hash of obejct to decrement
		_ int, //              : never used
		_ ...cipher.SHA256, // : never used
	) (
		deepper bool, //       : go deepper
		err error, //          : a DB error
	) {

		var (
			rc  uint32
			val []byte
		)

		// use Get instead of Inc(hash, -1) to get value
		// if it will be deleted by the -1

		if val, rc, err = i.c.Get(hash, -1); err != nil {
			return
		}

		// keep last if it was deleted
		if rc == 0 {
			dpack.last = hash
			dpack.val = val

			deepper = true // and go deepper
		}

		// we are going deepper only if value has been deleted by the Get
		return

	}

	pack = dpack

	return
}

// delRootRelatedValues decrements all values related to
// given Root, including the Root itself and its Registry
func (i *Index) delRootRelatedValues(rootHash cipher.SHA256) (err error) {

	var r *registry.Root
	if r, err = i.c.RootByHash(rootHash); err != nil {
		return
	}

	var (
		pack     registry.Pack
		walkFunc registry.WalkFunc
	)

	if pack, walkFunc, err = i.delPackWalkFunc(r); err != nil {
		return
	}

	return i.c.walkRoot(pack, r, walkFunc)
}

// DelRoot deletes Root. It can't remove a held Root
func (i *Index) DelRoot(pk cipher.PubKey, nonce, seq uint64) (err error) {

	// with lock
	var rootHash cipher.SHA256
	if rootHash, err = i.delRootFromIndexLock(pk, nonce, seq); err != nil {
		return
	}

	// without lock
	return i.delRootRelatedValues(rootHash)
}

// Feeds returns list of feeds
func (i *Index) Feeds() (feeds []cipher.PubKey) {

	i.mx.Lock()
	defer i.mx.Unlock()

	if len(i.feedsl) != len(i.feeds) {
		for pk := range i.feeds {
			i.feedsl = append(i.feedsl, pk)
		}
	}

	return i.feedsl

}

// HasFeed returns true if feed exists
func (i *Index) HasFeed(pk cipher.PubKey) (yep bool) {

	i.mx.Lock()
	defer i.mx.Unlock()

	_, yep = i.feeds[pk]
	return
}

// HasHead returns true if head exists. It
// returns false if given feed doesn't exist
func (i *Index) HasHead(pk cipher.PubKey, nonce uint64) (yep bool) {

	i.mx.Lock()
	defer i.mx.Unlock()

	var hs map[uint64][]*indexRoot

	if hs, yep = i.feeds[pk]; yep == false {
		return
	}

	_, yep = hs[nonce]
	return
}

// AddHead adds head. A head will be added automatically if
// AddRoot called and head of the Root doesn't exist. But
// it's possible to add an empty head to save it in DB.
// It has no practical value. The head will be saved in DB
// and the Heads method will return it even if it empty
func (i *Index) AddHead(pk cipher.PubKey, nonce uint64) (err error) {

	i.mx.Lock()
	defer i.mx.Unlock()

	// check out Index first

	var hs, ok = i.feeds[pk]

	if ok == false {
		return data.ErrNoSuchFeed
	}

	if _, ok = hs[nonce]; ok {
		return // already exists
	}

	// add to DB

	err = i.c.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {
		var heads data.Heads
		if heads, err = feeds.Heads(pk); err != nil {
			return
		}

		_, err = heads.Add(nonce)
		return
	})

	if err != nil {
		return
	}

	// add to the Index

	hs[nonce] = nil
	return

}
