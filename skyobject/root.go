package skyobject

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

// A Root represetns wrapper of root object. The real root object is
// serialized to []byte
type Root struct {
	container *Container `enc:"-"` // back reference
	registry  map[string]cipher.SHA256
	Schema    cipher.SHA256
	Root      cipher.SHA256
	Time      int64
	Seq       int64
}

// NewRoot creates new root object from given interface. The method don't
// set the new root as root of the Container
func (c *Container) NewRoot() (root *Root) {
	return &Root{
		Time:     time.Now().UnixNano(),
		Seq:      0,
		registry: make(map[string]cipher.SHA256),
	}
}

// Set sets given object to the root (i.e. the object will be root)
func (r *Root) Set(i interface{}) {
	r.Schema = r.container.Save(getSchema(i))
	r.Root = r.container.Save(i)
}

// Register schema of given interface with provided name
func (r *Root) Register(name string, i interface{}) {
	r.registry[name] = r.container.Save(getSchema(i))
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
