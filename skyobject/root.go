package skyobject

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// A Root represents wrapper around root object
type Root struct {
	Time int64
	Seq  uint64

	Refs []Dynamic // all references of the root

	// TODO
	Sign cipher.Sig    // signature
	Pub  cipher.PubKey // public key

	reg *Registry  `enc:"-"` // TODO: encoding and decoding
	cnt *Container `enc:"-"` // back reference
}

// Touch set timestamp to now and increment seq
func (r *Root) Touch() {
	r.Time = time.Now().UnixNano()
	r.Seq++
}

// Add given object to root
func (r *Root) Inject(i interface{}) {
	r.Refs = append(r.Refs, r.Dynamic(i))
}

// Encode convertes a root to []byte
func (r *Root) Encode() (p []byte) {
	var x struct {
		Root Root
		Nmr  []struct{ K, V string } // map[string]string
		Reg  []struct {              // map[string]cipher.SHA256
			K string
			V cipher.SHA256
		}
	}
	x.Root = *r
	for k, v := range r.reg.nmr {
		x.Nmr = append(x.Nmr, struct{ K, V string }{k, v})
	}
	for k, v := range r.reg.reg {
		x.Reg = append(x.Reg, struct {
			K string
			V cipher.SHA256
		}{k, v})
	}
	p = encoder.Serialize(&x)
	return
}

// SetEncodedRoot set given data as root object of the container.
// It returns an error if the data can't be encoded. It returns
// true if the root is set
func (c *Container) SetEncodedRoot(p []byte) (ok bool, err error) {
	var x struct {
		Root Root
		Nmr  []struct{ K, V string } // map[string]string
		Reg  []struct {              // map[string]cipher.SHA256
			K string
			V cipher.SHA256
		}
	}
	if err = encoder.DeserializeRaw(p, &x); err != nil {
		return
	}
	var root *Root = &x.Root
	root.cnt = c
	root.reg = NewRegistery(c.db)
	for _, v := range x.Nmr {
		root.reg.nmr[v.K] = v.V
	}
	for _, v := range x.Reg {
		root.reg.reg[v.K] = v.V
	}
	ok = c.AddRoot(root)
	return
}

func (r *Root) SchemaByReference(sr Reference) (s *Schema, err error) {
	if sr == (Reference{}) {
		err = ErrEmptySchemaKey
		return
	}
	s, err = r.reg.SchemaByReference(sr)
	return
}
