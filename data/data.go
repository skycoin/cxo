package data

import (
	"fmt"
	"strings"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

type DB struct {
	sync.RWMutex
	data map[cipher.SHA256][]byte
}

type Stat struct {
	Total  int `json:"total"`
	Memory int `json:"memory"`
}

// String implemets fmt.Stringer interface and returns
// human readable string
func (s Stat) String() string {
	return fmt.Sprintf("{total: %d, memory: %s}",
		s.Total,
		HumanMemory(s.Memory))
}

// HumanMemory returns human readable memory string
func HumanMemory(bytes int) string {
	var fb float64 = float64(bytes)
	var ms string = "B"
	for _, m := range []string{"KiB", "MiB", "GiB"} {
		if fb > 1024.0 {
			fb = fb / 1024.0
			ms = m
			continue
		}
		break
	}
	if ms == "B" {
		return fmt.Sprintf("%.0fB", fb)
	}
	// 2.00 => 2
	// 2.10 => 2.1
	// 2.53 => 2.53
	return strings.TrimRight(
		strings.TrimRight(fmt.Sprintf("%.2f", fb), "0"),
		".") + ms
}

func NewDB() *DB {
	return &DB{
		data: make(map[cipher.SHA256][]byte),
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
