package gnet

import (
	"strings"
)

// prefix related contants
const (
	// PrefixLength is length of a message prefix
	PrefixLength int = 4 // message prefix length
)

// Prefix represents message prefix that used as
// alias for type of message
type Prefix [4]byte

func NewPrefix(s string) (p Prefix) {
	if len(s) != PrefixLength {
		panic("wrong length of NewPrefix argument")
	}
	if strings.Contains(s, "-") {
		panic("minus '-' is not allowed for prefix")
	}
	copy(p[:], s)
	return
}
