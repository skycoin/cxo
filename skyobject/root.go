package skyobject

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

// A Root represetns wrapper of root object. The real root object is
// serialized to []byte
type Root struct {
	container *Container
	registry  map[string]cipher.SHA256
	Schema    cipher.SHA256
	Root      cipher.SHA256
	Time      int64
	Seq       int64
}

// NewRoot creates new root object from given interface. The method doesn't
// set the new root as root of the Container
func (c *Container) NewRoot() (root *Root) {
	return &Root{
		container: c,
		Time:      time.Now().UnixNano(),
		Seq:       0,
		registry:  make(map[string]cipher.SHA256),
	}
}

// Set sets given object to the root (i.e. the object will be root)
func (r *Root) Set(i interface{}) (err error) {
	if i == nil {
		err = ErrInvalidArgument
		return
	}
	// todo: filter types
	r.Schema = r.container.Save(getSchema(i))
	r.Root = r.container.Save(i)
	return
}

// Register schema of given interface with provided name
func (r *Root) Register(name string, i interface{}) (err error) {
	if i == nil {
		err = ErrInvalidArgument
		return
	}
	r.registry[name] = r.container.Save(getSchema(i))
	return
}

// SchemaKey returns key of registered Schema by given name
func (r *Root) SchemaKey(name string) (sk cipher.SHA256, ok bool) {
	sk, ok = r.registry[name]
	return
}

// Touch increnets Seq and set Time of the Root to now
func (r *Root) Touch() {
	r.Seq++
	r.Time = time.Now().UnixNano()
}

// initialize is called when a root object received by node
// and it should be set as new root on a container
func (r *Root) initialize(c *Container) {
	r.container = c
}

// encoding and decoding root object

// Encode is used to transfer root object. Because of registry and encoder
// that can't encode map, we need to use intermediate value to encode and
// decode the Root
func (r *Root) Encode() []byte {
	var intermediate = struct {
		Registry map[string]cipher.SHA256
		*Root
	}{
		Registry: r.registry,
		Root:     r,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(intermediate); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// Decode root is used when a node received decoded root objet. Unfortunately,
// the cipher/ecnoded can't encode maps and unexported fields. The Decode
// returns an error if given data is malformed. You can to use methods of
// decoded root only, and only, after (*Container).SetRoot if the SetRoot
// returns true
func DecodeRoot(data []byte) (root *Root, err error) {
	var intermediate struct {
		Registry map[string]cipher.SHA256
		*Root
	}
	err = gob.NewDecoder(bytes.NewReader(data)).Decode(&intermediate)
	if err != nil {
		return
	}
	root = intermediate.Root
	if root == nil {
		err = ErrMissingRoot
		return
	}
	root.registry = intermediate.Registry
	return
}
