package skyobject

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

type Root struct {
	container *Container `enc:"-"` // back reference
	registry  map[string]cipher.SHA256
	Schema    cipher.SHA256
	Root      []byte
	Time      int64
	Seq       int64
}

// NewRoot creates new root object from given interface. The method don't
// set the new root as root of the Container. Use SetRoot as follow
//
//     c.SetRoot(c.NewRoot(Object{}))
//
func (c *Container) NewRoot(i interface{}) (root *Root) {
	return &Root{
		Schema:   c.Save(getSchema(i)),
		Root:     encoder.Serialize(i),
		Time:     time.Now().UnixNano(),
		Seq:      0,
		registry: make(map[string]cipher.SHA256),
	}
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

func (r *Root) Touch() {
	r.Seq++
	r.Time = time.Now().UnixNano()
}
