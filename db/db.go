package db

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

type Stat struct {
	Total  int `json:"total"`
	Memory int `json:"memory"`
}

type DB interface {
	Has(cipher.SHA256) bool
	Set(cipher.SHA256, []byte)
	Get(cipher.SHA256) ([]byte, bool)

	Stat() Stat
}

type db struct {
	sync.RWMutex
	data map[cipher.SHA256][]byte
}

func NewDB() DB {
	return &db{
		data: make(map[cipher.SHA256][]byte),
	}
}

func (d *db) has(k cipher.SHA256) (ok bool) {
	_, ok = d.data[k]
	return
}

func (d *db) Has(k cipher.SHA256) bool {
	d.RLock()
	defer d.RUnlock()
	return d.has(k)
}

func (d *db) Set(k cipher.SHA256, v []byte) {
	d.Lock()
	defer d.Unlock()
	d.data[k] = v
}

func (d *db) Get(k cipher.SHA256) (v []byte, ok bool) {
	d.RLock()
	defer d.RUnlock()
	v, ok = d.data[k]
	return
}

func (d *db) Stat() (s Stat) {
	d.RLock()
	defer d.RUnlock()
	s.Total = len(d.data)
	for _, v := range d.data {
		s.Memory += len(v)
	}
	return
}
