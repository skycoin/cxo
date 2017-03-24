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

	// All of the references points to Dynamic objects
	Refs []Reference // all references of the root

	Sig cipher.Sig    `enc:"-"` // signature
	Pub cipher.PubKey `enc:"-"` // public key

	reg *Registry  `enc:"-"` // back reference to registery
	cnt *Container `enc:"-"` // back reference to container
}

// Sing encodes the root and calculate signature of hash of encoded data
// using given secret key
func (r *Root) Sign(sec cipher.SecKey) {
	r.Sig = cipher.SignHash(cipher.SumSHA256(r.Encode()), sec)
}

// Touch set timestamp to now and increment seq
func (r *Root) Touch() {
	r.Time = time.Now().UnixNano()
	r.Seq++
}

// Add given object to root
func (r *Root) Inject(i interface{}) {
	r.Refs = append(r.Refs, r.Save(r.Dynamic(i)))
}

// Encode convertes a root to []byte
func (r *Root) Encode() (p []byte) {
	var x struct {
		Root Root
		Reg  []struct { // map[string]cipher.SHA256
			K string
			V cipher.SHA256
		}
	}
	x.Root = *r
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
	if sr.IsBlank() {
		err = ErrEmptySchemaKey
		return
	}
	s, err = r.reg.SchemaByReference(sr)
	return
}

// Save an object to db and get reference-key to it
func (r *Root) Save(i interface{}) Reference {
	return Reference(r.cnt.db.AddAutoKey(encoder.Serialize(i)))
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

// Dynamic saves object and its schema in db and returns dynamic reference,
// that points to the object and the schema
func (r *Root) Dynamic(i interface{}) (dn Dynamic) {
	dn.Object = r.Save(i)
	dn.Schema = r.SaveSchema(i)
	return
}

// Register schema of given object with given name
func (r *Root) Register(name string, i interface{}) {
	r.reg.Register(name, i)
}

//
// reference value
//

// Values returns set of values the root object refers to
func (r *Root) Values() (vs []*Value, err error) {
	if r == nil {
		return
	}
	if len(r.Refs) == 0 {
		return
	}
	vs = make([]*Value, 0, len(r.Refs))
	var (
		s *Schema

		dd     []byte
		sd, od []byte
		ok     bool
	)
	for _, rd := range r.Refs {
		// take a look at the reference
		if rd.IsBlank() {
			err = ErrInvalidReference // nil-references are not allowed for root
			return
		}
		// obtain dynamic reference, the reference points to
		if dd, ok = r.cnt.get(rd); !ok {
			err = &MissingObject{rd, ""}
			return
		}
		// decode the dynamic reference
		var dr Dynamic
		if err = encoder.DeserializeRaw(dd, &dr); err != nil {
			return
		}
		// is the dynamic reference valid
		if !dr.IsValid() {
			err = ErrInvalidReference
			return
		}
		// is it blank
		if dr.IsBlank() {
			vs = append(vs, nilValue(r, nil)) // no value, nor schema
			continue
		}
		// obtain schema of the dynamic reference
		if sd, ok = r.cnt.get(dr.Schema); !ok {
			err = &MissingSchema{dr.Schema}
			return
		}
		// decode the schema
		s = new(Schema)
		if err = s.Decode(r.reg, sd); err != nil {
			return
		}
		// obtain object of the dynamic reference
		if od, ok = r.cnt.get(dr.Object); !ok {
			err = &MissingObject{key: dr.Object, schemaName: s.Name()}
			return
		}
		// create value
		vs = append(vs, &Value{r, s, od})
	}
	return
}
