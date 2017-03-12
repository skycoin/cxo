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
	hrefTypeName      = reflect.TypeOf(cipher.SHA256{}).Name()
	hrefArrayTypeName = reflect.TypeOf([]cipher.SHA256{}).Name()

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
func (c *Container) SaveArray(i ...interface{}) (a cipher.SHA256) {
	if len(i) == 0 {
		return // nil
	}
	var ch []cipher.SHA256 = make([]cipher.SHA256, 0, len(i))
	for _, x := range i {
		ch = append(ch, c.Save(x))
	}
	a = c.Save(ch)
	return
}

// Want returns slice of nessessary references that
// doesn't exist in db but required
func (c *Container) Want() (want map[cipher.SHA256]struct{}, err error) {
	if c.root == nil {
		return
	}
	want = make(map[cipher.SHA256]struct{})
	err = c.want(c.root.Schema, c.root.Root, want)
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
		c.addMissing(want, objk)
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

	for i := 0; i < len(s.Fields); i++ {
		var (
			sf  encoder.StructField = s.Fields[i]
			tag string              = skyobjectTag(sf)
		)
		if tag == "" {
			continue
		}
		if sf.Type == hrefTypeName {
			var ref cipher.SHA256
			if ref, err = getRefField(objd, s.Fields, sf.Name); err != nil {
				goto Error
			}
			if strings.Contains(tag, "schema=") {
				// the field contains cipher.SHA256 reference
				err = c.mergeRefWants(want, tag, ref)
				if err != nil {
					goto Error
				}
			} else if strings.Contains(tag, "dynamic_schema") {
				i++
				if i >= len(s.Fields) {
					err = ErrMissingObjKeyField
					goto Error
				}
				err = c.mergeDynamicWants(objd, s.Fields, s.Fields[i], want, ref)
				if err != nil {
					goto Error
				}
			} // else -> not a reference
		} else if sf.Type == hrefArrayTypeName {
			// the field contains reference to []cipher.SHA256 references
			err = c.mergeArrayWants(objd, s.Fields, sf.Name, want, tag)
			if err != nil {
				goto Error
			}
		}
	}
	return
Error:
	want = nil // set want to nil if we have got an error
	return
}

func (c *Container) mergeDynamicWants(objd []byte,
	fs []encoder.StructField,
	sf encoder.StructField,
	want map[cipher.SHA256]struct{},
	ref cipher.SHA256) (err error) {

	var tag string = skyobjectTag(sf)
	if !strings.Contains(tag, "dynamic_objkey") ||
		sf.Type != hrefTypeName {
		err = ErrMissingObjKeyField
		return
	}
	var objRef cipher.SHA256
	objRef, err = getRefField(objd, fs, sf.Name)
	if err != nil {
		return
	}
	if err = c.want(ref, objRef, want); err != nil {
		return
	}
	return
}

func (c *Container) mergeRefWants(want map[cipher.SHA256]struct{},
	tag string,
	ref cipher.SHA256) (err error) {

	var schk cipher.SHA256

	if schk, err = c.schemaByTag(tag, "schema="); err != nil {
		return
	}
	if err = c.want(schk, ref, want); err != nil {
		return
	}
	return
}

func (c *Container) mergeArrayWants(objd []byte,
	fs []encoder.StructField,
	name string,
	want map[cipher.SHA256]struct{},
	tag string) (err error) {

	var (
		refs []cipher.SHA256
		schk cipher.SHA256
	)
	if refs, err = getRefsField(objd, fs, name); err != nil {
		return
	}
	if schk, err = c.schemaByTag(tag, "array="); err != nil {
		return
	}
	for _, ref := range refs {
		if err = c.want(schk, ref, want); err != nil {
			return
		}
	}
	return
}

func getRefField(data []byte, fs []encoder.StructField,
	fname string) (ref cipher.SHA256, err error) {
	err = encoder.DeserializeField(data, fs, fname, &ref)
	return
}

func getRefsField(data []byte, fs []encoder.StructField,
	fname string) (refs []cipher.SHA256, err error) {
	err = encoder.DeserializeField(data, fs, fname, &refs)
	return
}

// append key to array if it is not exist in db
func (c *Container) addMissing(w map[cipher.SHA256]struct{},
	keys ...cipher.SHA256) {

	for _, key := range keys {
		if _, ok := c.db.Get(key); !ok {
			w[key] = struct{}{}
		}
	}

}

// get vlaue of `skyobjet:"xxx"` tag or empty string
func skyobjectTag(sf encoder.StructField) string {
	return reflect.StructTag(sf.Tag).Get("skyobject")
}

// tagSchemaName returns schema name or error if there is no
// schema=xxx in given tag, it returns an error if given tag
// is invalid
func tagSchemaName(tag, key string) (s string, err error) {
	for _, p := range strings.Split(tag, ",") {
		if strings.HasPrefix(p, key) {
			ss := strings.Split(p, "=")
			if len(ss) != 2 {
				err = fmt.Errorf("invalid schema tag: %s", p)
				return
			}
			s = ss[1]
			return
		}
	}
	if s == "" {
		err = ErrMissingSchemaName
	}
	return
}

// if given tag contains schema=xxx then it reutrns appropriate schema or
// error if the schema is not registered
func (c *Container) schemaByTag(tag string, key string) (s cipher.SHA256,
	err error) {

	var (
		schemaName string
		ok         bool
	)
	if schemaName, err = tagSchemaName(tag, key); err != nil {
		return
	}
	if c.root == nil {
		err = ErrMissingRoot
		return
	}
	if s, ok = c.root.registry[schemaName]; !ok {
		err = fmt.Errorf("unregistered schema: %s", schemaName)
	}
	return
}
