package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// ObejctIsTooLargeError represents error that
// occurs when an obejct exceed max obejct size
// limit. The error contians hash of the obejct
type ObjectIsTooLargeError struct {
	hash cipher.SHA256
}

// Hash of the large object
func (o *ObjectIsTooLargeError) Hash() cipher.SHA256 {
	return o.hash
}

// Error implements error interface
func (o *ObjectIsTooLargeError) Error() string {
	return "object is too large: " + o.Hash().Hex()[:7]
}
