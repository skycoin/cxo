// Package comm represents common types for RPC messaging
package comm

import (
	"github.com/skycoin/skycoin/src/cipher"
)

type Info struct {
	Address string                     // listening address
	Feeds   map[cipher.PubKey][]string // feed -> connections
}
