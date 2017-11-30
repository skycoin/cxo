package skyobject

import (
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/data/cxds"
	"github.com/skycoin/cxo/data/idxdb"
	"github.com/skycoin/cxo/skyobject/registry"
)

// A Container represents
type Container struct {
	Cache // cache of the Container

	db *data.DB // database

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
	//
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

func (c *Container) initDB() (err error) {

	if c.db.IdxDB().IsClosedSafely() == true {
		return // everything is ok
	}

	// so, we have to walk through IdxDB and all
	// Root obejcts setting correct rcs for CXDS

	//

}

// Close the Container and related DB. The DB
// will be closed, even if the Container created
// with user-provided DB.
func (c *Container) Close() (err error) {

	//

	return

}
