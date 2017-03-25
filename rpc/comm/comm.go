// Package comm represents common types for RPC messaging
package comm

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// Reply of Info mehthod of node
type Info struct {
	Address string                     // listening address
	Feeds   map[cipher.PubKey][]string // feed -> connections
}

// Arguments for Inject method of node
type Inject struct {
	Hash cipher.SHA256
	Pub  cipher.PubKey
	Sec  cipher.SecKey
}
