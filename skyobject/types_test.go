package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func getHash(i interface{}) cipher.SHA256 {
	return cipher.SumSHA256(encoder.Serialize(i))
}

type User struct {
	Name   string
	Age    int64
	Hidden string `enc:"-"` // ignore the field
}

type Man struct {
	Name   string
	Height int64
	Weight int64
}

type SamllGroup struct {
	Name     string
	Leader   cipher.SHA256   `skyobject:"schema=User"` // single User
	Outsider cipher.SHA256   // not a reference
	FallGuy  Dynamic         // dynamic href
	Members  []cipher.SHA256 `skyobject:"array=User"` // array of Users
}
