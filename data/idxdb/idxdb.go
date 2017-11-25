package idxdb

import (
	"encoding/binary"
	"errors"
)

const Version = 1 // previous is 0

// common errors
var (
	ErrInvalidSize = errors.New("invalid size of encoded object")

	ErrMissingMetaInfo = errors.New("missing meta information")
	ErrMissingVersion  = errors.New("missing version in meta")
	ErrOldVersion      = errors.New("db file of old version")      // cxodbfix
	ErrNewVersion      = errors.New("db file newer then this CXO") // go get
)

// version

func versionBytes() (vb []byte) {
	vb = make([]byte, 4)
	binary.BigEndian.PutUint32(vb, uint32(Version))
	return
}
