package gnet

import (
	"errors"
	"unicode"
)

// prefix related contants
const (
	// PrefixLength is length of a message prefix
	PrefixLength int = 4 // message prefix length
)

// prefix related errors
var (
	// ErrMinusInPrefix occurs when you trying to use
	// prefixx contains '-' that reserved for internal
	// usage
	ErrMinusInPrefix = errors.New("minus '-' is not allowed for prefix")
	// ErrUnpritableRune occurs when you trying to use
	// unprintable prefix. The prefix must be printable
	// for logs
	ErrUnpritableRune = errors.New("unprintable rune in prefix")
)

// Prefix represents message prefix that used as
// alias for type of message
type Prefix [4]byte

// String implements fmt.Stringer interface
func (p Prefix) String() string {
	return string(p[:])
}

// Validate check the Prefix to be sure that the prefix
// doesn't contains service symbols ('-') and is printable
// for logging
func (p Prefix) Validate() (err error) {
	var r rune
	for _, r = range p.String() {
		if r == '-' {
			err = ErrMinusInPrefix
			return
		}
		if !unicode.IsPrint(r) {
			err = ErrUnpritableRune
			return
		}
	}
	return // ok
}

// NewPrefix creates Prefix from given string. The function
// panics if an error occurs
func NewPrefix(s string) (p Prefix) {
	if len(s) != PrefixLength {
		panic("wrong length of NewPrefix argument")
	}
	copy(p[:], s)
	var err error
	if err = p.Validate(); err != nil {
		panic(err)
	}
	return
}
