package skyobject

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

var (
	ErrMissingRoot        = errors.New("misisng root object")
	ErrShortBuffer        = errors.New("short buffer")
	ErrInvalidSchema      = errors.New("invalid schema")
	ErrMissingInDB        = errors.New("missing in db")
	ErrUnregisteredSchema = errors.New("unregistered schema")
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

// by schema and object key
func (c *Container) wantSchemaObjKey(s *Schema,
	ok cipher.SHA256, set Set) (n int, err error) {

	var od []byte // object data
	var ex bool   // exist
	if _, ex = c.db.Get(ok); !ex {
		set.Add(ok)
		return
	}

	n, err = c.wantSchemaObj(s, od, set)
	return
}

// by schema and object data
func (c *Container) wantSchemaObj(s *Schema,
	od []byte, set Set) (n int, err error) {

	switch s.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		n += 1
	case reflect.Int16, reflect.Uint16:
		n += 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		n += 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		n += 8
	case reflect.String:
		var l int
		if l, err = getLength(od); err != nil {
			return
		}
		n += 4 + l
	case reflect.Array:
		var elem *Schema = s.Elem()
		if elem == nil {
			err = ErrInvalidSchema
			return
		}
		if kind := elem.Kind(); isBasic(kind) {
			n = s.Len() * basicSize(kind)
			return
		}
		var l int = s.Len()
		var m int
		for i := 0; i < l; i++ {
			if m, err = c.wantSchemaObj(elem, od[n:], set); err != nil {
				return
			}
			n += m
		}
	case reflect.Slice:
		var elem *Schema = s.Elem()
		if elem == nil {
			err = ErrInvalidSchema
			return
		}
		var l int
		if l, err = getLength(od); err != nil {
			return
		}
		n += 4 // length
		if kind := elem.Kind(); isBasic(kind) {
			n = l * basicSize(kind)
			return
		}
		var m int
		for i := 0; i < l; i++ {
			if m, err = c.wantSchemaObj(elem, od[n:], set); err != nil {
				return
			}
			n += m
		}
	case reflect.Struct:
		var m int
		for _, sf := range s.Fields() {
			if m, err = c.wantField(&sf, od[n:], set); err != nil {
				return
			}
			n += m
		}
	default:
		err = ErrInvalidSchema
	}

	return
}

func (c *Container) wantField(f *Field, od []byte, set Set) (n int, err error) {

	var s *Schema = f.Schema()

	if s == nil {
		err = ErrInvalidSchema
		return
	}

	switch s.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		n += 1
	case reflect.Int16, reflect.Uint16:
		n += 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		n += 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		n += 8
	case reflect.String:
		var l int
		if l, err = getLength(od); err != nil {
			return
		}
		n += 4 + l
	case reflect.Array:
		var elem *Schema = s.Elem()
		if elem == nil {
			err = ErrInvalidSchema
			return
		}
		// short curcit for cipher.SHA256
		if s.Name() == htSingle {
			if strings.Contains(f.Tag().Get(TAG), "href") { // a reference
				// unfortunately the encoder can't deserialize arrays, but can
				// deserialize a struc; stupid but true
				var keeper struct {
					Ref cipher.SHA256
				}
				if len(od) < 32 {
					err = ErrShortBuffer
					return
				}
				if err = encoder.DeserializeRaw(od[:32], &keeper); err != nil {
					return
				}
			} else {
				n += 32 // not a reference (skip)
			}
			return
		}
		//
		if kind := elem.Kind(); isBasic(kind) {
			n = s.Len() * basicSize(kind)
			return
		}
		var l int = s.Len()
		var m int
		for i := 0; i < l; i++ {
			if m, err = c.wantSchemaObj(elem, od[n:], set); err != nil {
				return
			}
			n += m
		}
	case reflect.Slice:
		var elem *Schema = s.Elem()
		if elem == nil {
			err = ErrInvalidSchema
			return
		}
		var l int
		if l, err = getLength(od); err != nil {
			return
		}
		n += 4 // length
		if kind := elem.Kind(); isBasic(kind) {
			n = l * basicSize(kind)
			return
		}
		var m int
		for i := 0; i < l; i++ {
			if m, err = c.wantSchemaObj(elem, od[n:], set); err != nil {
				return
			}
			n += m
		}
	case reflect.Struct:
		var m int
		for _, sf := range s.Fields() {
			if m, err = c.wantField(&sf, od[n:], set); err != nil {
				return
			}
			n += m
		}
	default:
		err = ErrInvalidSchema
	}

	return
}

func getLength(p []byte) (l int, err error) {
	if len(p) < 4 {
		err = ErrShortBuffer
		return
	}
	var u uint32
	encoder.DeserializeAtomic(p, &u)
	l = int(u)
	return
}

func recoveredError(x interface{}) error {
	switch z := x.(type) {
	case error:
		return z
	case string:
		return errors.New(z)
	}
	return errors.New(fmt.Print(z))
}
