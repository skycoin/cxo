package data

import (
	"fmt"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

type DB struct {
	sync.RWMutex
	data map[cipher.SHA256][]byte
}

type Statistic struct {
	Total  int `json:"total"`
	Memory int `json:"memory"`
}

func (s Statistic) String() string {
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

type QueryFunc func(key cipher.SHA256, data []byte) bool

type IDataSource interface {
	Save(value interface{}) cipher.SHA256
	Update(value []byte) cipher.SHA256
	Add(ds cipher.SHA256, value []byte) error
	Has(ds cipher.SHA256) bool
	Get(ds cipher.SHA256) ([]byte, bool)
	Where(QueryFunc) []cipher.SHA256
	Statistic() Statistic

	Data() map[cipher.SHA256][]byte
}

func NewDB() *DB {
	return &DB{
		data: make(map[cipher.SHA256][]byte),
	}
}

func createKey(data []byte) cipher.SHA256 {
	return cipher.SumSHA256(data)
}

func (d *DB) Save(value interface{}) cipher.SHA256 {
	return d.Update(encoder.Serialize(value))
}

func (d *DB) Update(data []byte) cipher.SHA256 {
	key := createKey(data)
	if key == (cipher.SHA256{}) || data == nil {
		panic("Invalid key")
	}
	d.Lock()
	d.data[key] = data
	d.Unlock()
	return key
}

func (d *DB) Add(key cipher.SHA256, value []byte) (err error) {
	if key == (cipher.SHA256{}) || value == nil {
		panic("Invalid key")
	}

	d.Lock()
	defer d.Unlock()

	if d.has(key) {
		return fmt.Errorf("key already present: %v", key)
	}
	d.data[key] = value
	return
}

func (d *DB) has(key cipher.SHA256) (ok bool) {
	_, ok = d.data[key]
	return
}

func (d *DB) Has(key cipher.SHA256) bool {
	d.RLock()
	defer d.RUnlock()
	return d.has(key)
}

func (d *DB) Get(key cipher.SHA256) (v []byte, ok bool) {
	d.RLock()
	defer d.RUnlock()
	v, ok = d.data[key]
	return
}

func (d *DB) Where(q QueryFunc) []cipher.SHA256 {
	result := []cipher.SHA256{}
	d.RLock()
	defer d.RUnlock()
	for key, value := range d.data {
		if q(key, value) {
			result = append(result, key)
		}
	}
	return result
}

func (d *DB) Statistic() (s Statistic) {
	d.RLock()
	d.RUnlock()
	s.Total = len(d.data)
	for _, v := range d.data {
		s.Memory += len(v) // + len(cipher.SHA256) ?
	}
	return
}

// it's unsafe to use it asyncronously
func (d *DB) Data() map[cipher.SHA256][]byte {
	return d.data
}
