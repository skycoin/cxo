package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

type Age uint32

type Group struct {
	Name    string
	Leader  cipher.SHA256   `skyobject:"schema=User"`
	Members []cipher.SHA256 `skyobject:"schema=User"`
	Curator Dynamic
}

type User struct {
	Name   string ``
	Age    Age    `json:"age"`
	Hidden int    `enc:"-"`
}

type List struct {
	Name    string
	Members []cipher.SHA256 `skyobject:"schema=User"`
}

type Man struct {
	Name    string
	Age     Age
	Seecret []byte
	Friends List
}
