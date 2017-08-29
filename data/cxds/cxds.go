// Package cxds represents stub for CXO data server.
// The cxds is some kind of client of CX data server.
// In future it will be replaced with real client.
// API of this package close to goal and can be used
package cxds

import (
	"encoding/binary"
	"errors"

	"github.com/skycoin/skycoin/src/cipher"
)

// comon errors
var (
	ErrNotFound   = errors.New("not found in CXDS")
	ErrEmptyValue = errors.New("empty value")
)

var one []byte

func init() {
	one = make([]byte, 4)
	binary.LittleEndian.PutUint32(one, 1)
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
