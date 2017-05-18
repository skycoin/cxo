package data

import (
	"fmt"
	"strings"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

// in-memory database
type mdb struct {
	dmx  sync.RWMutex
	data map[cipher.SHA256][]byte // data objects

	rmx  sync.RWMutex
	root map[cipher.PubKey][][]byte // root objects
}

func newMdb() DB {
	return &mdb{
		data: make(map[cipher.SHA256][]byte),
		root: make(map[cipher.PubKey][][]byte),
	}
}

//
func (d *DB) Has(key cipher.SHA256) (ok bool) {
	d.RLock()
	defer d.RUnlock()
	_, ok = d.data[key]
	return
}

// Get value by key
func (d *DB) Get(key cipher.SHA256) (v []byte, ok bool) {
	d.RLock()
	defer d.RUnlock()
	v, ok = d.data[key]
	return
}

// Set or overwrite key-value pair
func (d *DB) Set(key cipher.SHA256, data []byte) {
	d.Lock()
	defer d.Unlock()
	d.data[key] = data
}

// Range over keys of DB, fn must not be nil (read only)
func (d *DB) Range(fn func(key cipher.SHA256)) {
	d.RLock()
	defer d.RUnlock()
	for k := range d.data {
		fn(k)
	}
}

func (d *DB) AddAutoKey(data []byte) (key cipher.SHA256) {
	key = cipher.SumSHA256(data)
	d.Lock()
	defer d.Unlock()
	d.data[key] = data
	return
}

func (d *DB) Del(key cipher.SHA256) {
	d.Lock()
	defer d.Unlock()
	delete(d.data, key)
}

// Stat return statistic of the DB
func (d *DB) Stat() (s Stat) {
	d.RLock()
	defer d.RUnlock()
	s.Total = len(d.data)
	for _, v := range d.data {
		s.Memory += len(v)
	}
	return
}
