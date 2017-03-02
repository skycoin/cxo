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
		humanMemory(s.Memory))
}

// humanMemory returns human readable memory string
func humanMemory(bytes int) string {
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

func (d *DB) Has(key cipher.SHA256) (ok bool) {
	d.RLock()
	defer d.RUnlock()
	_, ok = d.data[key]
	return
}

func (d *DB) Get(key cipher.SHA256) (v []byte, ok bool) {
	d.RLock()
	defer d.RUnlock()
	v, ok = d.data[key]
	return
}

func (d *DB) Set(key cipher.SHA256, data []byte) {
	d.Lock()
	defer d.Unlock()
	d.data[key] = data
}

func (d *DB) Stat() (s Stat) {
	d.RLock()
	d.RUnlock()
	s.Total = len(d.data)
	for _, v := range d.data {
		s.Memory += len(v)
	}
	return
}
