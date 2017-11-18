package cxds

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

type memoryCXDS struct {
	mx  sync.RWMutex
	kvs map[cipher.SHA256]memoryObject
}

// object stored in memory
type memoryObject struct {
	rc  uint32
	val []byte
}

// NewMemoryCXDS creates CXDS-databse in
// memory. The database based on golang map
func NewMemoryCXDS() data.CXDS {
	return &memoryCXDS{kvs: make(map[cipher.SHA256]memoryObject)}
}

// Get value by key
func (m *memoryCXDS) Get(key cipher.SHA256) (val []byte, rc uint32, err error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	if mo, ok := m.kvs[key]; ok {
		val, rc = mo.val, mo.rc
		return
	}
	err = data.ErrNotFound
	return
}

// GetInc is Get + Inc
func (m *memoryCXDS) GetInc(key cipher.SHA256) (val []byte, rc uint32,
	err error) {

	m.mx.Lock()
	defer m.mx.Unlock()

	if mo, ok := m.kvs[key]; ok {
		mo.rc++
		val = mo.val
		rc = mo.rc
		m.kvs[key] = mo // save
		return
	}
	err = data.ErrNotFound
	return
}

// Set of Inc if exists
func (m *memoryCXDS) Set(key cipher.SHA256, val []byte) (rc uint32, err error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	if len(val) == 0 {
		err = ErrEmptyValue
		return
	}
	if mo, ok := m.kvs[key]; ok {
		mo.rc++
		m.kvs[key] = mo
		rc = mo.rc
		return
	}
	m.kvs[key] = memoryObject{1, val}
	rc = 1
	return
}

// Add is like Set, but it calculates key inside
func (m *memoryCXDS) Add(val []byte) (key cipher.SHA256, rc uint32, err error) {
	if len(val) == 0 {
		err = ErrEmptyValue
		return
	}
	key = getHash(val)
	rc, err = m.Set(key, val)
	return
}

// Inc increments references counter
func (m *memoryCXDS) Inc(key cipher.SHA256) (rc uint32, err error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	if mo, ok := m.kvs[key]; ok {
		mo.rc++
		m.kvs[key] = mo
		rc = mo.rc
		return
	}
	err = data.ErrNotFound
	return
}

// Dec decrements referecnes counter and removes value if it turns 0
func (m *memoryCXDS) Dec(key cipher.SHA256) (rc uint32, err error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	if mo, ok := m.kvs[key]; ok {
		mo.rc--
		rc = mo.rc
		if mo.rc == 0 {
			delete(m.kvs, key)
			return
		}
		m.kvs[key] = mo
		return
	}
	err = data.ErrNotFound
	return
}

// DecGet is Get + Dec
func (m *memoryCXDS) DecGet(key cipher.SHA256) (val []byte, rc uint32,
	err error) {

	m.mx.Lock()
	defer m.mx.Unlock()

	if mo, ok := m.kvs[key]; ok {
		mo.rc--
		rc = mo.rc
		val = mo.val
		if mo.rc == 0 {
			delete(m.kvs, key)
			return
		}
		m.kvs[key] = mo
		return
	}
	err = data.ErrNotFound
	return
}

// MultiGet returns many values by list of keys
func (m *memoryCXDS) MultiGet(keys []cipher.SHA256) (vals [][]byte, err error) {
	if len(keys) == 0 {
		return
	}
	vals = make([][]byte, 0, len(keys))

	m.mx.RLock()
	defer m.mx.RUnlock()

	for _, key := range keys {
		if mo, ok := m.kvs[key]; ok {
			vals = append(vals, mo.val)
			continue
		}
		err = data.ErrNotFound
		return
	}
	return
}

// MultiAdd appends all given values like the Add
func (m *memoryCXDS) MultiAdd(vals [][]byte) (err error) {
	for _, val := range vals {
		if _, _, err = m.Add(val); err != nil {
			return
		}
	}
	return
}

// MultiInc increments references counter for all values by given keys
func (m *memoryCXDS) MultiInc(keys []cipher.SHA256) (err error) {
	for _, key := range keys {
		if _, err = m.Inc(key); err != nil {
			return
		}
	}
	return
}

// MultiDec decrements all values by given keys
func (m *memoryCXDS) MultiDec(keys []cipher.SHA256) (err error) {
	for _, key := range keys {
		if _, err = m.Dec(key); err != nil {
			return
		}
	}
	return
}

// Iterate all keys
func (m *memoryCXDS) Iterate(iterateFunc func(cipher.SHA256,
	uint32) error) (err error) {

	m.mx.RLock()
	defer m.mx.RUnlock()

	for k, mo := range m.kvs {
		if err = iterateFunc(k, mo.rc); err != nil {
			if err == data.ErrStopIteration {
				err = nil
			}
			return
		}
	}

	return
}

// Close DB
func (m *memoryCXDS) Close() (_ error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.kvs = nil // clear
	return
}
