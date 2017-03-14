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
	refTypeName     = reflect.TypeOf(cipher.SHA256{}).Name()
	refsTypeName    = reflect.TypeOf([]cipher.SHA256{}).Name()
	dynamicTypeName = reflect.TypeOf(Dynamic{}).Name()

	// ErrMissingRoot occurs when a Container doesn't have
	// a root object, but the action requires it
	ErrMissingRoot = errors.New("missing root object")
	// ErrMissingSchemaName occurs when a tag constains "schema=" or "array="
	// key but does't contain a value
	ErrMissingSchemaName = errors.New("missing schema name")
	// ErrMissingObject occurs where requested object is not received yet
	ErrMissingObject = errors.New("missing object")
	// ErrInvalidArgument occurs when given argument is not valid
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrMalformedRoot can occur during SetRoot call if the given
	// root is malformed
	ErrMalformedRoot = errors.New("malformed root")
	// ErrMissingObjKeyField occurs when a malformed schema is used with
	// Dynamic.Schema field but without Dynamic.ObjKey
	ErrMissingObjKeyField = errors.New("missing ObjKey field")
	//
	ErrInvalidSchema = errors.New("invalid schema")
	//
	ErrMalformedData = errors.New("malformed data")

	// ErrStopInspection is used to stop Inspect
	ErrStopInspection = errors.New("stop inspection")
)

// A Container is a helper type to manage skyobjects. The container is not
// thread safe
type Container struct {
	db   *data.DB
	root *Root
}

// NewContainer creates new Container that will use provided database.
// The database must not be nil
func NewContainer(db *data.DB) (c *Container) {
	if db == nil {
		panic("NewContainer tooks in nil-db")
	}
	c = &Container{
		db: db,
	}
	return
}

// Root returns root object or nil
func (c *Container) Root() *Root {
	return c.root
}

// SetRoot replaces existing root from given one if timespamp of the given root
// is greater. The given root must not be nil
func (c *Container) SetRoot(root *Root) (ok bool) {
	if root == nil {
		panic(ErrMissingRoot)
	}
	if c.root == nil {
		c.root, ok = root, true
		return
	}
	if c.root.Time < root.Time {
		c.root, ok = root, true
	}
	return
}

// SetEncodedRoot decodes given data to Root and set it as root of the
// Container. It returns (ok, nil) if root of the container replaced,
// (false, nil) if not and (false, err) if there is a decoding error
func (c *Container) SetEncodedRoot(data []byte) (ok bool, err error) {
	var root *Root
	if root, err = decodeRoot(data); err != nil {
		return
	}
	ok = c.SetRoot(root)
	return
}

// Save serializes given object and sotres it in DB returning
// key of the object
func (c *Container) Save(i interface{}) (k cipher.SHA256) {
	return c.db.AddAutoKey(encoder.Serialize(i))
}

// SaveArray saves array of objects and retursn references to them
func (c *Container) SaveArray(i ...interface{}) (ch []cipher.SHA256) {
	if len(i) == 0 {
		return // nil
	}
	for _, x := range i {
		ch = append(ch, c.Save(x))
	}
	return
}

// Want returns slice of nessessary references that
// doesn't exist in db but required
func (c *Container) Want() (want map[cipher.SHA256]struct{}, err error) {
	if c.root == nil {
		return
	}
	want = make(map[cipher.SHA256]struct{})
	if err = c.want(c.root.Schema, c.root.Root, want); err != nil {
		want = nil
	}
	return
}

func (c *Container) want(schk, objk cipher.SHA256,
	want map[cipher.SHA256]struct{}) (err error) {

	var (
		schd, objd []byte
		ok         bool

		s Schema
	)

	if schd, ok = c.db.Get(schk); !ok { // don't have the schema
		want[schk] = struct{}{}
		if _, ok := c.db.Get(objk); !ok { // add objk if it's missing
			want[objk] = struct{}{}
		}
		return
	}

	if objd, ok = c.db.Get(objk); !ok {
		want[objk] = struct{}{}
		return
	}

	// we have both schema and object
	if err = encoder.DeserializeRaw(schd, &s); err != nil {
		return
	}

	err = c.getReferences(objd, s.Fields, want)
	return
}

func (c *Container) wantSchema(s *Schema, objk cipher.SHA256,
	want map[cipher.SHA256]struct{}) (err error) {

	var (
		schd, objd []byte
		ok         bool
	)

	if objd, ok = c.db.Get(objk); !ok {
		want[objk] = struct{}{}
		return
	}

	err = c.getReferences(objd, s.Fields, want)
	return
}

func (c *Container) getReferences(objd []byte, fs []Field,
	want map[cipher.SHA256]struct{}) (err error) {

	for i := 0; i < len(fs); i++ {
		var (
			sf    Field = fs[i]
			shift int
		)
		if !strings.Contains(sf.Tag.Get("skyobject"), "href") {
			continue
		}
		switch kind := reflect.Kind(sf.Schema.Kind); kind {
		case reflect.Array:
			if sf.Schema.Name != refTypeName { // todo
				continue
			}
			var (
				ref cipher.SHA256
				ln  int
			)
			if ln = sf.Schema.Size(objd[shift:]); ln < 0 {
				err = ErrMalformedData
				return
			}
			err = encoder.DeserializeRaw(objd[shift:shift+ln], &ref)
			if err != nil {
				return
			}
			if err = c.wantSchema(&sf.Schema, ref, want); err != nil {
				return
			}
			shift += ln
			continue
		case reflect.Slice:
			if sf.Schema.Name != refsTypeName { // todo
				continue
			}
			if len(sf.Schema.Elem) != 1 {
				err = ErrInvalidSchema
				return
			}
			var (
				refs []cipher.SHA256
				ln   int
			)
			if ln = sf.Schema.Size(objd[shift:]); ln < 0 {
				err = ErrMalformedData
				return
			}
			err = encoder.DeserializeRaw(objd[shift:shift+ln], &ref)
			if err != nil {
				return
			}
			for _, ref := range refs {
				err = c.wantSchema(&sf.Schema.Elem[0], ref, want)
				if err != nil {
					return
				}
			}
			shift += ln
			continue
		case reflect.Struct:
			var ln int
			if ln = sf.Schema.Size(objd[shift:]); ln < 0 {
				err = ErrMalformedData
				return
			}
			if sf.Schema.Name == dynamicTypeName { // todo
				var dn Dynamic
				err = encoder.DeserializeRaw(objd[shift:shift+ln], &dn)
				if err != nil {
					return
				}
				if err = c.want(dn.Schema, dn.ObjKey, want); err != nil {
					return
				}
			} else {
				err = c.getReferences(objd[shift:shift+ln],
					sf.Schema.Fields, want)
				if err != nil {
					return
				}
			}
			shift += ln
			continue
		}
		shift += sf.Schema.Size(objd[shift:])
	}
}
