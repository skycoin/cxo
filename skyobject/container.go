package skyobject

import (
	"errors"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/data/cxds"
	"github.com/skycoin/cxo/data/idxdb"
	"github.com/skycoin/cxo/skyobject/registry"
)

var (
	ErrRootIsHeld    = errors.New("Root is held")
	ErrRootIsNotHeld = errors.New("Root is not held")
)

// A Container represents
type Container struct {
	Cache // cache of the Container

	db  *data.DB // database
	idx          // memory mapped IdxDB

	conf Config // ccopy of given

	// human readable (used by node for debugging)
	cxPath, idxPath string
}

// HumanCXDSPath returns human readable path
// to CXDS. It used by the node package for
// dbug logs
func (c *Container) HumanCXDSPath() string {
	return c.cxPath
}

// HumanIdxDBPath returns human readable path
// to IdxDB. It used by the node package for
// dbug logs
func (c *Container) HumanIdxDBPath() string {
	return c.idxPath
}

// NewContainer creates Container instance using provided
// Config. If the Config is nil, then default config is
// used (see NewConfig function)
func NewContainer(conf *Config) (c *Container, err error) {

	if conf == nil {
		conf = NewConfig() // default
	}

	c = new(Container)

	c.conf = *conf // copy

	if err = c.createDB(conf); err != nil {
		return
	}

	// init DB, repair if need, get stat, and initialize cache
	if err = c.initDB(); err != nil {
		c.db.Close()         // ignore error
		c.Cache.stat.Close() // release resources
		return
	}

	return // done
}

func (c *Container) createDB(conf *Config) (err error) {

	if conf.DataDir != "" {
		if err = mkdirp(conf.DataDir); err != nil {
			return
		}
	}

	var db *data.DB

	if conf.DB != nil {

		c.cxPath, c.idxPath = "<used provided DB>", "<used provided DB>"
		db = conf.DB

	} else if conf.InMemoryDB == true {

		c.cxPath, c.idxPath = "<in memory>", "<in memory>"
		db = data.NewDB(cxds.NewMemoryCXDS(), idxdb.NewMemeoryDB())

	} else {

		if conf.DBPath == "" {
			c.cxPath = filepath.Join(conf.DataDir, CXDS)
			c.idxPath = filepath.Join(conf.DataDir, IdxDB)
		} else {
			c.cxPath = conf.DBPath + ".cxds"
			c.idxPath = conf.DBPath + ".idx"
		}

		var cx data.CXDS
		var idx data.IdxDB

		if cx, err = cxds.NewDriveCXDS(cxPath); err != nil {
			return
		}

		if idx, err = idxdb.NewDriveIdxDB(idxPath); err != nil {
			cx.Close()
			return
		}

		db = data.NewDB(cx, idx)

	}

	c.db = db

	return
}

type rcs struct {
	rc uint32 // saved rc (DB)
	cc uint32 // correct rc (determined by walking)
}

type cxdsRCs struct {
	hr map[cipher.SHA256]rcs // hash -> {rc, actual rc}

	amount uint32 // stat
	volume uint32 // stat
}

func (c *Container) getHashRCs() (cr *cxdsRCs, err error) {
	cr = new(cxdsRCs)
	cr.hr = make(map[cipher.SHA256]rcs)

	err = c.db.CXDS().Iterate(func(hash cipher.SHA256, rc uint32) (err error) {
		var val []byte

		if val, _, err = c.db.CXDS().Get(hash, 0); err != nil {
			return
		}

		cr.amount++                   // stat
		cr.volume += uint32(len(val)) // stat

		cr.hr[hash] = rcs{rc: rc}
		return
	})

	if err != nil {
		return
	}

	return
}

func (c *Container) getPack(reg *registry.Registry) (pack *Pack) {
	pack = new(Pack)

	pack.deg = c.conf.Degree
	pack.reg = reg
	pack.c = c

	return
}

func (c *Container) initRoot(cr *cxdsRCs, hash cipher.SHA256) (err error) {

	var val []byte
	if val, err = c.Get(rootHash, 0); err != nil {
		return
	}

	var r *registry.Root
	if r, err = registry.DecodeRoot(val); err != nil {
		return
	}

	if val, err = c.Get(cipher.SHA256(r.Reg), 0); err != nil {
		return
	}

	var reg *registry.Registry
	if reg, err = registry.DecodeRegistry(val); err != nil {
		return
	}

	var pack = c.getPack(reg)

	err = r.Walk(pack,
		func(
			hash cipher.SHA256,
			_ int,
			_ ...cipher.SHA256,
		) (
			deepper bool,
			err error,
		) {

			var rc, ok = cr.hr[hash]
			if ok == false {
				err = data.ErrNotFound
				return
			}
			rc.cc++
			deepper = rc.cc == 1 // go deepper for first look
			cr.hr[hash] = rc
			return

		})

	return

}

func (c *Container) initDB() (err error) {

	if c.db.IdxDB().IsClosedSafely() == true {

		// get stat from the CXDS
		var amount, volume uint32
		if amount, volume, err = c.db.CXDS().Stat(); err != nil {
			return
		}

		// create cache
		c.Cache.initialize(c.db.CXDS(), &c.conf, amount, volume)

		return // everything is ok
	}

	// try to repair DB

	// (pre-)create cache for the Pack(s)
	c.Cache.initialize(c.db.CXDS(), &c.conf, 0, 0)
	defer func() {
		if err != nil {
			c.Cache.stat.Close() // release resources
		}
	}()

	// so, we have to walk through IdxDB and all
	// Root obejcts setting correct rcs for CXDS

	// collect hash->{existing rc, actual rc}
	var cr *cxdsRCs

	if cr, err = c.getHashRCs(); err != nil {
		return
	}

	// determine actual rcs
	err = c.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {

		err = feeds.Iterate(func(pk cipher.PubKey) (err error) {

			var hs data.Heads
			if hs, err = feeds.Heads(pk); err != nil {
				return
			}

			err = hs.Iterate(func(nonce uint64) (err error) {

				var rs data.Roots
				if rs, err = hs.Roots(nonce); err != nil {
					return
				}

				err = rs.Ascend(func(r *data.Root) (err error) {
					return c.initRoot(cr, r.Hash)
				})

				return
			})

			return
		})

		return
	})

	// correct rcs in DB
	for key, rc := range hrcs {
		if rc.rc == rc.cc {
			continue // actual value
		}
		var inc = int(rc.cc) - int(rc.rc)
		// correct
		if _, err = c.db.CXDS().Inc(key, inc); err != nil {
			return
		}
	}

	// save stat
	if err = c.db.CXDS().SetStat(cr.amount, cr.volume); err != nil {
		return
	}

	// and clear cache (recreate the cache with CXDS stat)
	c.Cache.reset()
	c.Cache.initialize(c.db.CXDS(), &c.conf, int(cr.amount), int(cr.volume))

	return
}

// Close the Container and related DB. The DB
// will be closed, even if the Container created
// with user-provided DB.
func (c *Container) Close() (err error) {

	// the Cache.Close closes CXDS
	if err = c.Cache.Close(); err == nil {
		err = c.db.Close()
	} else {
		c.db.Close() // ignore error
	}

	return

}

//
//
//

// AddFeed adds feed
func (c *Container) AddFeed(pk cipher.PubKey) (err error) {
	return c.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {
		return feeds.Add(pk)
	})
}

// AddHead adds head
func (c *Container) AddHead(pk cipher.PubKey, nonce uint64) (err error) {
	return c.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {
		var hs data.Heads
		if hs, err = feeds.Heads(pk); err != nil {
			return
		}
		_, err = hs.Add(nonce)
		return
	})
}

// ReceivedRoot called by the node package to
// check a recived root. The method verify hash
// and signature of the Root. The method also
// check database, may be DB already has this
// root
func (c *Container) ReceivedRoot(
	sig cipher.Sig,
	val []byte,
) (
	r *registry.Root,
	err error,
) {

	var hash = cipher.SumSHA256(val)
	if err = cipher.VerifySignature(r.Pub, sig, hash); err != nil {
		return
	}

	if r, err = registry.DecodeRoot(val); err != nil {
		return
	}

	r.Hash = hash // set the hash
	r.Sig = sig   // set the signature

	var alreadyHave bool

	err = c.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {
		var hs data.Heads
		if hs, err = feeds.Heads(r.Pub); err != nil {
			return
		}
		var rs data.Roots
		if rs, err = hs.Add(r.Nonce); err != nil {
			return
		}
		alreadyHave, err = rs.Has(r.Seq)
		return
	})

	r.IsFull = alreadyHave
	return
}

// AddRoot to DB. The method doesn;t create feed of the root
// but if head of the root doesn't exist, then the method
// creates the head. The method never return already have error
// and the method never save the Root inside CXDS. E.g. the
// method adds the Root to index (that is necessary)
func (c *Container) AddRoot(r *registry.Root) (err error) {

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

	err = c.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {
		var hs data.Heads
		if hs, err = feeds.Heads(r.Pub); err != nil {
			return
		}
		var rs data.Roots
		if rs, err = hs.Add(r.Nonce); err != nil {
			return
		}
		var dr = new(data.Root)
		dr.Seq = r.Seq
		dr.Prev = r.Prev
		dr.Hash = r.Hash
		dr.Sig = r.Sig
		return rs.Set(dr)
	})

	return
}

// LastRoot
func (c *Container) LastRoot(pk cipher.PubKey) (r *registry.Root, err error) {

}

// DelFeed deletes feed. If one or more Root objects of the feed
// are held, then the method removes all Root obejcts before and
// returns error
func (c *Container) DelFeed(pk cipher.PubKey) (err error) {
	//
}

// DelHead deletes given head. It can't remove a held Root
func (c *Container) DelHead(pk cipher.PubKey, nonce uint64) (err error) {
	//
}

// DelRoot deletes Root. It's can't remove held Root
func (c *Container) DelRoot(pk cipher.PubKey, nonce, seq uint64) (err error) {

	if c.IsRootHeld(pk, nonce, seq) == true {
		return ErrRootIsHeld
	}

	// so, a Root can be held here, but removed, so fucking stupid

}
