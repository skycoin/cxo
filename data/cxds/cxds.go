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

const Version int = 1 // previous is 0

// comon errors
var (
	ErrEmptyValue      = errors.New("empty value")
	ErrMissingMetaInfo = errors.New("missing meta information")
	ErrMissingVersion  = errors.New("missing version in meta")
	ErrOldVersion      = errors.New("db file of old version")      // cxodbfix
	ErrNewVersion      = errors.New("db file newer then this CXO") // go get
)

var one []byte

func init() {
	one = make([]byte, 4)
	binary.BigEndian.PutUint32(one, 1)
}

//
func getRefsCount(val []byte) (rc uint32) {
	rc = binary.BigEndian.Uint32(val)
	return
}

// returns new uint32
func incRefsCount(val []byte) (rc uint32) {
	rc = binary.BigEndian.Uint32(val) + 1
	binary.BigEndian.PutUint32(val, rc)
	return
}

// returns new uint32
func decRefsCount(val []byte) (rc uint32) {
	rc = binary.BigEndian.Uint32(val) - 1
	binary.BigEndian.PutUint32(val, rc)
	return
}

func getHash(val []byte) (key cipher.SHA256) {
	return cipher.SumSHA256(val)
}

// version

func versionBytes() (vb []byte) {
	vb = make([]byte, 4)
	binary.BigEndian.PutUint32(vb, uint32(Version))
	return
}
