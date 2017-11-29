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

func mincr(
	m *memoryCXDS,
	key cipher.SHA256,
	mo memoryObject,
	rc uint32,
	inc int,
) (
	nrc uint32,
) {

	switch {
	case inc == 0:
		nrc = rc
	case inc < 0:
		inc = -inc // change the sign

		if uinc := uint32(inc); uinc >= rc {
			nrc = 0
			delete(m.kvs, key) // remove
		} else {
			nrc = rc - uinc
			mo.rc = nrc
			m.kvs[key] = mo
		}
	case inc > 0:
		nrc = rc + uint32(inc)
		mo.rc = nrc
		m.kvs[key] = mo
	}

	return

}

// Get value and change rc
func (m *memoryCXDS) Get(
	key cipher.SHA256,
	inc int,
) (
	val []byte,
	rc uint32,
	err error,
) {

	if inc == 0 { // read only
		m.mx.RLock()
		defer m.mx.RUnlock()
	} else { // read-write
		m.mx.Lock()
		defer m.mx.Unlock()
	}

	if mo, ok := m.kvs[key]; ok {
		val, rc = mo.val, mo.rc
		rc = mincr(m, key, mo, rc, inc)
		return
	}
	err = data.ErrNotFound
	return
}

// Set value and change rc
func (m *memoryCXDS) Set(
	key cipher.SHA256,
	val []byte,
	inc int,
) (
	rc uint32,
	err error,
) {

	if inc <= 0 {
		panicf("invalid inc argument is Set: %d", inc)
	}

	m.mx.Lock()
	defer m.mx.Unlock()

	if len(val) == 0 {
		err = ErrEmptyValue
		return
	}

	if mo, ok := m.kvs[key]; ok {
		rc = mincr(m, key, mo, mo.rc, inc)
		return
	}

	rc = uint32(inc)
	m.kvs[key] = memoryObject{rc, val}

	return
}

// Inc changes rc
func (m *memoryCXDS) Inc(
	key cipher.SHA256,
	inc int,
) (
	rc uint32,
	err error,
) {

	if inc == 0 { // presence check
		m.mx.RLock()
		defer m.mx.RUnlock()
	} else { // changes
		m.mx.Lock()
		defer m.mx.Unlock()
	}

	if mo, ok := m.kvs[key]; ok {
		rc = mincr(m, key, mo, mo.rc, inc)
		return
	}

	err = data.ErrNotFound
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

// Stat is just a stub
func (m *memoryCXDS) Stat() (_, _ uint32, _ error) {
	return
}

// SetStat is just a stub
func (m *memoryCXDS) SetStat(_, _ uint32) (_ error) {
	return
}

// Close DB
func (m *memoryCXDS) Close() (_ error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.kvs = nil // clear
	return
}
