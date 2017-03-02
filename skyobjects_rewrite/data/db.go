package data

import (
	"fmt"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// DB represents a database.
type DB struct {
	sync.RWMutex
	data map[cipher.SHA256][]byte
}

// Statistic is used for showing database statistics.
type Statistic struct {
	Total  int `json:"total"`
	Memory int `json:"memory"`
}

// QueryFunc is used for querying the database.
type QueryFunc func(key cipher.SHA256, data []byte) bool

// IDataSource is an interface for databases.
type IDataSource interface {
	Save(value interface{}) cipher.SHA256
	Update(value []byte) cipher.SHA256
	Add(ds cipher.SHA256, value []byte) error
	AddAutoKey(data []byte) cipher.SHA256
	Remove(ds cipher.SHA256)
	Has(ds cipher.SHA256) bool
	Get(ds cipher.SHA256) ([]byte, bool)
	Where(QueryFunc) []cipher.SHA256
	Statistic() Statistic

	Data() map[cipher.SHA256][]byte
}

// NewDB creates a new database.
func NewDB() *DB {
	return &DB{
		data: make(map[cipher.SHA256][]byte),
	}
}

func createKey(data []byte) cipher.SHA256 {
	return cipher.SumSHA256(data)
}

// Save stores an object to database.
// Key will be hash of data.
func (d *DB) Save(value interface{}) cipher.SHA256 {
	return d.Update(encoder.Serialize(value))
}

// Update stores a serialized object to database.
// Key will be hash of data.
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

// Add strictly adds something new to database.
// Key will be separately specified.
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

// AddAutoKey strictly adds something new to database while auto generating key.
func (d *DB) AddAutoKey(data []byte) (key cipher.SHA256) {
	key = cipher.SumSHA256(data)
	d.Add(key, data)
	return
}

// Remove removes an item from db.
func (d *DB) Remove(ds cipher.SHA256) {
	d.Lock()
	defer d.Unlock()
	delete(d.data, ds)
}

func (d *DB) has(key cipher.SHA256) (ok bool) {
	_, ok = d.data[key]
	return
}

// Has determines whether we have stored something of specified key.
func (d *DB) Has(key cipher.SHA256) bool {
	d.RLock()
	defer d.RUnlock()
	return d.has(key)
}

// Get retrieves data of specified key.
func (d *DB) Get(key cipher.SHA256) (v []byte, ok bool) {
	d.RLock()
	defer d.RUnlock()
	v, ok = d.data[key]
	return
}

// Where retrieves list of keys for specified query.
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

// Statistic returns statistics for database.
func (d *DB) Statistic() (s Statistic) {
	d.RLock()
	defer d.RUnlock()
	s.Total = len(d.data)
	for _, v := range d.data {
		s.Memory += len(v) // + len(cipher.SHA256) ?
	}
	return
}

// Data retrieves all data. iIt's unsafe to use it asyncronously.
func (d *DB) Data() map[cipher.SHA256][]byte {
	return d.data
}
