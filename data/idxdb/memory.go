package idxdb

import (
	"io/ioutil"
	"os"
)

type memoryDB struct {
	*driveDB
	name string
}

// NewMemeoryDB returns stub for memory DB.
// The memeory-db is not implemened yet
// and the function retusn on-drive-db that
// uses temporary file that deleted on Close
func NewMemeoryDB() (idx IdxDB) {
	fl, err := ioutil.TempFile("", "cxds")
	if err != nil {
		panic(err)
	}
	fl.Close()
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
