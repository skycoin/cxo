package skyobject

import (
	"log"
	"path/filepath"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/data/cxds"
	"github.com/skycoin/cxo/data/idxdb"
	"github.com/skycoin/cxo/skyobject/registry"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// A Container represents
type Container struct {
	Cache // cache of the Container
	Index // memory mapped IdxDB

	db *data.DB // database

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

	defer func() {
		if err != nil {
			c.db.Close()         // ignore error
			c.Cache.stat.Close() // release resources
		}
	}()

	// check size of obejcts
	if err = c.checkSize(); err != nil {
		return
	}

	// init DB, repair if need, get stat, and initialize cache
	if err = c.initDB(); err != nil {
		return
	}

	if err = c.Index.load(c); err != nil {
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

		if cx, err = cxds.NewDriveCXDS(c.cxPath); err != nil {
			return
		}

		if idx, err = idxdb.NewDriveIdxDB(c.idxPath); err != nil {
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

func (c *Container) initRoot(cr *cxdsRCs, rootHash cipher.SHA256) (err error) {

	var val []byte
	if val, _, err = c.Get(rootHash, 0); err != nil {
		return
	}

	var r *registry.Root
	if r, err = registry.DecodeRoot(val); err != nil {
		return
	}

	if val, _, err = c.Get(cipher.SHA256(r.Reg), 0); err != nil {
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

func (c *Container) checkSize() (err error) {

	if c.conf.CheckSizes == false {
		return
	}

	err = c.db.CXDS().Iterate(
		func(key cipher.SHA256, _ uint32) (err error) {
			var val []byte
			if val, _, err = c.db.CXDS().Get(key, 0); err != nil {
				return
			}
			if len(val) > c.conf.MaxObjectSize {
				return &ObjectIsTooLargeError{key}
			}
			return
		})

	return

}

func (c *Container) initDB() (err error) {

	if c.db.IdxDB().IsClosedSafely() == true && c.conf.ReinitDB == false {

		// get stat from the CXDS
		var amount, volume uint32
		if amount, volume, err = c.db.CXDS().Stat(); err != nil {
			return
		}

		// create cache
		c.Cache.initialize(c.db.CXDS(), &c.conf, int(amount), int(volume))

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
	for key, rc := range cr.hr {
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

// Walk walks throug given Root calling given
// walkFunc for every obejct of the Root including
// hash of the Root and Regsitry (depending on the
// deepper reply of the WalkFunc)
func (c *Container) Walk(
	r *registry.Root,
	walkFunc registry.WalkFunc,
) (
	err error,
) {

	var reg *registry.Registry
	if reg, err = c.Registry(r.Reg); err != nil {
		return
	}

	var pack = c.getPack(reg)

	return c.walkRoot(pack, r, walkFunc)
}

func (c *Container) walkRoot(
	pack registry.Pack,
	r *registry.Root,
	walkFunc registry.WalkFunc,
) (
	err error,
) {

	defer func() {
		if err == registry.ErrStopIteration {
			err = nil
		}
	}()

	// hash of the Root is first (ignore deepper)
	if _, err = walkFunc(r.Hash, 0); err != nil {
		return
	}

	// registry is second (ignore deepper)
	if _, err = walkFunc(cipher.SHA256(r.Reg), 0); err != nil {
		return
	}

	return r.Walk(pack, walkFunc)
}

// Config returns configs of the Container
func (c *Container) Config() (conf Config) {
	conf = c.conf // copy
	return
}

// RootByHash returns Root by its hash, that is
// obvious. Errors the method can returns are
// data.ErrNotfound or decoding error
func (c *Container) RootByHash(
	hash cipher.SHA256,
) (
	r *registry.Root,
	err error,
) {

	var val []byte
	if val, _, err = c.Get(hash, 0); err != nil {
		return
	}

	r, err = registry.DecodeRoot(val)
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

func fatal(args ...interface{}) {
	log.Fatalln(arg...)
}

func fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}
