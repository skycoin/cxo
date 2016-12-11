package skyobject

import "github.com/skycoin/skycoin/src/cipher"

type HashRoot struct {
	Href
	size int
	refs int
	sign cipher.Sig
}
