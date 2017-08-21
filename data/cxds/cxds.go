// Package cxds represents stub for CXO data server.
// The cxds is some kind of client of CX data server.
// In future it will be replaced with real client.
// API of this package close to goal and can be used
package cxds

import (
	"encoding/binary"

	"github.com/skycoin/skycoin/src/cipher"
)

var one []byte

func init() {
	one = make([]byte, 4)
	binary.LittleEndian.PutUint32(one, 1)
}

type CXDS interface {
	// Get value by key. Result is value and references count
	Get(key cipher.SHA256) (val []byte, rc uint32, err error)
	// Set key-value pair. If value already exists the Set
	// increments references count
	Set(key cipher.SHA256, val []byte) (rc uint32, err error)
	// Add value, calculating hash internally. If
	// value already exists the Add increments references
	// count
	Add(val []byte) (key cipher.SHA256, rc uint32, err error)
	// Inc increments references count. It never returns
	// "not found" error. The rc reply is zero if value
	// not found
	Inc(key cipher.SHA256) (rc uint32, err error)
	// Dec decrements references count. The rc is zero if
	// value was not found or was deleted by the Dec
	Dec(key cipher.SHA256) (rc uint32, err error)

	// batch operation

	// MultiGet returns values by keys
	MultiGet(keys []cipher.SHA256) (vals [][]byte, err error)
	// MultiAdd append given values
	MultiAdd(vals [][]byte) (err error)

	// MultiInc increments all existsing
	MultiInc(keys []cipher.SHA256) (err error)
	// MultiDec decrements all existsing
	MultiDec(keys []cipher.SHA256) (err error)

	// Clsoe the CXDS
	Close() (err error)
}

//
func getRefsCount(val []byte) (rc uint32) {
	rc = binary.LittleEndian.Uint32(val)
	return
}

// returns new uint32
func incRefsCount(val []byte) (rc uint32) {
	rc = binary.LittleEndian.Uint32(val) + 1
	binary.LittleEndian.PutUint32(val, rc)
	return
}

// returns new uint32
func decRefsCount(val []byte) (rc uint32) {
	rc = binary.LittleEndian.Uint32(val) - 1
	binary.LittleEndian.PutUint32(val, rc)
	return
}

func getHash(val []byte) (key cipher.SHA256) {
	return cipher.SumSHA256(val)
}
