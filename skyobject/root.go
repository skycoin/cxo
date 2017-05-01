package skyobject

import (
	"reflect"
	"sort"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// RegistryEntities is used for sorting
type RegistryEntities []RegistryEntity

func (r RegistryEntities) Len() int {
	return len(r)
}

func (r RegistryEntities) Less(i, j int) bool {
	return r[i].K < r[j].K
}

func (r RegistryEntities) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// an entity of map[string]Reference
type RegistryEntity struct {
	K string
	V Reference
}

// rootEncoding is used to encode and decode the Root
type rootEncoding struct {
	Time int64
	Seq  uint64
	Refs []Reference
	Reg  RegistryEntities // registery
}

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

// Touch set timestamp to now and increment seq. The Inject and InjectHash
// methods call Touch implicit
func (r *Root) Touch() {
	r.Time = time.Now().UnixNano()
	r.Seq++
}

// Add given object to root. The Inject creates Dynamic object from given one
// and appends the Dynamic to the Root. The Inject signs the root and touch it
// too
func (r *Root) Inject(i interface{}, sec cipher.SecKey) (inj Reference) {
	inj = r.cnt.Save(r.cnt.Dynamic(i))
	r.Refs = append(r.Refs, inj)
	r.Touch()
	r.Sign(sec)
	return
}

// Encode convertes a root to []byte
func (r *Root) Encode() (p []byte) {
	var x rootEncoding
	// by unknown reasons Pub and Sig of original was changed after encoding
	x.Time = r.Time
	x.Seq = r.Seq
	x.Refs = r.Refs
	if len(r.reg.reg) > 0 {
		x.Reg = make(RegistryEntities, 0, len(r.reg.reg))
	}
	for k, v := range r.reg.reg {
		x.Reg = append(x.Reg, RegistryEntity{k, v})
	}
	sort.Sort(x.Reg)
	p = encoder.Serialize(&x)
	return
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
			vs = append(vs, nilValue(r.cnt, nil)) // no value, nor schema
			continue
		}
		// obtain schema of the dynamic reference
		if sd, ok = r.cnt.get(dr.Schema); !ok {
			err = &MissingSchema{dr.Schema}
			return
		}
		// decode the schema
		s = r.reg.newSchema()
		if err = s.Decode(sd); err != nil {
			return
		}
		// obtain object of the dynamic reference
		if od, ok = r.cnt.get(dr.Object); !ok {
			err = &MissingObject{key: dr.Object, schemaName: s.Name()}
			return
		}
		// create value
		vs = append(vs, &Value{r.cnt, s, od})
	}
	return
}

// Got is opposite to Want. It returns all objects the root object has got
func (r *Root) Got() (set Set, err error) {
	if len(r.Refs) == 0 {
		return
	}
	set = make(Set)
	var vs []*Value = make([]*Value, 0, len(r.Refs))
	var (
		s *Schema

		dd     []byte
		sd, od []byte
		ok     bool
	)
	for _, rd := range r.Refs {
		if rd.IsBlank() {
			err = ErrInvalidReference
			return
		}
		if dd, ok = r.cnt.get(rd); !ok {
			err = &MissingObject{rd, ""}
			return
		}
		set.Add(rd) // got
		var dr Dynamic
		if err = encoder.DeserializeRaw(dd, &dr); err != nil {
			return
		}
		if !dr.IsValid() {
			err = ErrInvalidReference
			return
		}
		if dr.IsBlank() { // skip blank
			continue
		}
		if sd, ok = r.cnt.get(dr.Schema); !ok {
			err = &MissingSchema{dr.Schema}
			return
		}
		set.Add(dr.Schema) // got
		s = r.reg.newSchema()
		if err = s.Decode(sd); err != nil {
			return
		}
		if od, ok = r.cnt.get(dr.Object); !ok {
			err = &MissingObject{key: dr.Object, schemaName: s.Name()}
			return
		}
		set.Add(dr.Object) // got
		vs = append(vs, &Value{r.cnt, s, od})
	}
	for _, val := range vs {
		if err = gotValue(val, set); err != nil {
			return
		}
	}
	return
}

// GotOf returns values of particular object from list
// of top objects of the Root
func (r *Root) GotOf(ref Reference) (set Set, err error) {
	var val *Value
	if val, err = r.ValueOf(ref); err != nil {
		return
	}
	set = make(Set)
	err = gotValue(val, set)
	return
}

func gotValue(val *Value, set Set) (err error) {
	switch val.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Uint8,
		reflect.Int16, reflect.Uint16,
		reflect.Int32, reflect.Uint32, reflect.Float32,
		reflect.Int64, reflect.Uint64, reflect.Float64,
		reflect.String:
	case reflect.Slice, reflect.Array:
		var l int
		if l, err = val.Len(); err != nil {
			return
		}
		for i := 0; i < l; i++ {
			var d *Value
			if d, err = val.Index(i); err != nil {
				return
			}
			gotValue(d, set)
		}
	case reflect.Struct:
		err = val.RangeFields(func(fname string, d *Value) error {
			gotValue(d, set)
			return nil
		})
		if err != nil {
			return
		}
	case reflect.Ptr:
		var v *Value
		switch val.s.Name() {
		case DYNAMIC:
			var dr Dynamic
			if dr, err = val.dynamic(); err != nil {
				return
			}
			if _, ok := val.c.get(dr.Schema); ok {
				set.Add(dr.Schema) // got
			} else if _, ok := val.c.get(dr.Object); ok {
				// if no schema then no need to dereference,
				// but need to check does object exists
				set.Add(dr.Object) // got
				return
			} // else (got scema, but don't know about object)
			if v, err = val.dereferenceDynamic(dr); err != nil {
				return
			}
			set.Add(dr.Object) // got
			err = gotValue(v, set)
		case SINGLE:
			var ref Reference
			if ref, err = val.static(); err != nil {
				return
			}
			if v, err = val.dereferenceStatic(ref); err != nil {
				return
			}
			set.Add(ref) // got
			err = gotValue(v, set)
		default:
			err = ErrInvalidType
		}
	}
	return
}
