package skyobject

import (
	"errors"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

var (
	ErrMissingRoot = errors.New("misisng root object")
)

type Container struct {
	root *Root
	db   *data.DB
}

func (c *Container) Root() *Root {
	return c.root
}

func (c *Container) SetRoot(root *Root) (ok bool) {
	if c.root == nil {
		c.root, ok = root, true
		return
	}
	if c.root.Time < root.Time {
		c.root, ok = root, true
		return
	}
	return // false
}

// keys set
type Set map[cipher.SHA256]struct{}

func (s Set) Add(k cipher.SHA256) {
	s[k] = struct{}{}
}

// missing objects
func (c *Container) Want() (set Set, err error) {
	if c.root == nil {
		return // don't want anything (has no root object)
	}
	set = make(Set)
	err = c.wantKeys(c.root.Schema, c.root.Object, set)
	return
}

// want by schema key and object key
func (c *Container) wantKeys(sk, ok cipher.SHA256, set Set) (err error) {
	var sd, od []byte // shcema data and object data
	var ex bool       // exist
	if sd, ex = c.db.Get(sk); !ex {
		set.Add(sk)
		if _, ex = c.db.Get(ok); ex {
			set.Add(ok)
		}
		return
	}
	var s Schema
	if err = encoder.DeserializeRaw(sd, &s); err != nil {
		return
	}
	err = c.wantSchema(&s, ok, set)
	return
}

func (c *Container) wantSchema(s *Schema,
	ok cipher.SHA256, set Set) (err error) {

	var od []byte // object data
	var ex bool   // exist
	if _, ex = c.db.Get(ok); ex {
		set.Add(ok)
		return
	}

	switch typ.Kind() {
	case reflect.Bool,
		reflect.Int8, reflect.Uint8,
		reflect.Int16, reflect.Uint16,
		reflect.Int32, reflect.Uint32,
		reflect.Int64, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		//
	case reflect.Array:
		//
	case reflect.Slice:
		//
	case reflect.Struct:
		//
	default:
		//
	}

}
