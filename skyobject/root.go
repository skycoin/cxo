package skyobject

import (
	"fmt"
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

func (r *Root) SchemaByReference(sr Reference) (s *Schema, err error) {
	if sr == (Reference{}) {
		err = ErrEmptySchemaKey
		return
	}
	s, err = r.reg.SchemaByReference(sr)
	return
}

// SaveArray of objects and get array of references-keys to them
func (r *Root) SaveArray(ary ...interface{}) (rs References) {
	if len(ary) == 0 {
		return
	}
	rs = make(References, 0, len(ary))
	for _, a := range ary {
		rs = append(rs, r.Save(a))
	}
	return
}

// SaveSchema and get reference-key to it
func (r *Root) SaveSchema(i interface{}) (ref Reference) {
	return r.reg.SaveSchema(i)
}

func (r *Root) Dynamic(i interface{}) (dn Dynamic) {
	dn.Object = r.Save(i)
	dn.Schema = r.SaveSchema(i)
	return
}

// RegisterSchema with given name
func (r *Root) RegisterSchema(name string, i interface{}) {
	r.reg.Register(name, i)
}

//
// reference value
//

func (r *Root) Values() (vs []Value, err error) {
	if r == nil {
		return
	}
	if len(r.Refs) == 0 {
		return
	}
	vs = make([]Value, 0, len(r.Refs))
	var (
		s *Schema

		sd, od []byte
		ok     bool
	)
	for _, dr := range r.Refs {
		if sd, ok = r.cnt.get(dr.Schema); !ok {
			err = &MissingSchema{dr.Schema}
			return
		}
		s = new(Schema)
		if err = s.Decode(r.reg, sd); err != nil {
			return
		}
		if od, ok = r.cnt.get(dr.Object); !ok {
			err = &MissingObject{key: dr.Object, schemaName: s.Name()}
			return
		}
		vs = append(vs, &value{r, s, od})
	}
	return
}

// Tree returns printed tree for debugging
func (r *Root) Tree() string {
	return fmt.Sprintf("ROOT %d", 3)
}
