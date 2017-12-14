package idxdb

import (
	"io/ioutil"
	"os"

	"github.com/skycoin/cxo/data"
)

type memoryDB struct {
	*driveDB
	name string
}

// NewMemoryDB returns stub for memory DB.
// The memory-db is not implemened yet
// and the function returns on-drive-db that
// uses temporary file that deleted on Close
func NewMemoryDB() (idx data.IdxDB) {
	fl, err := ioutil.TempFile("", "cxds")
	if err != nil {
		panic(err)
	}
	fl.Close()
	os.Remove(fl.Name())
	// the NewDriveIdxDB uses os.Stat for internals
	// the removing is not as safe, but any problem
	// can occurs in < 1% of cases
	if idx, err = NewDriveIdxDB(fl.Name()); err != nil {
		panic(err)
	}
	idx = &memoryDB{idx.(*driveDB), fl.Name()} // wrap
	return
}

func (m *memoryDB) Close() (err error) {
	err = m.driveDB.Close()
	os.Remove(m.name)
	return
}
