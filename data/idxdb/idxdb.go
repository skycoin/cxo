package idxdb

import (
	"errors"
)

// common errors
var (
	ErrInvalidSize = errors.New("invalid size of encoded obejct")
)

/*

// A Volume represents size
// of an object in bytes
type Volume uint32

var units = [...]string{
	"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB",
}

// String implements fmt.String interface
// and returns human-readable string
// represents the Volume
func (v Volume) String() (s string) {

	fv := float64(v)

	var i int
	for ; fv >= 1024.0; i++ {
		fv /= 1024.0
	}

	s = fmt.Sprintf("%.2f", fv)
	s = strings.TrimRight(s, "0") // remove trailing zeroes (x.10, x.00)
	s = strings.TrimRight(s, ".") // remove trailing dot (x.)
	s += units[i]

	return
}

*/
