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
	Refs []Dynamic
	Reg  RegistryEntities // registery
}

// A Root represents wrapper around root object
type Root struct {
	Time int64
	Seq  uint64

	Refs []Dynamic // all references of the root

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
func (r *Root) Inject(i interface{}, sec cipher.SecKey) (inj Dynamic) {
	inj = r.cnt.Dynamic(i)
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
	var val *Value
	for _, dr := range r.Refs {
		val, err = r.cnt.ValueOf(dr)
		if err != nil {
			vs = nil // GC
			return
		}
		vs = append(vs, val)
	}
	return
}

// A GotFunc represents function that used in
// (*Root).GotFunc() and (*Container).GotOfFunc()
// methods. If the function returns an error
// then caller of the GotFunc terminates and returns
// the error. There is special case ErrStopRange
// that used to break the itteration without
// returning the error
type GotFunc func(hash Reference) (err error)

// GotFunc takes in function that will be recursively
// called for objects of the Root that the Root has got
func (r *Root) GotFunc(gf GotFunc) (err error) {
	for _, inj := range r.Refs {
		if err = r.cnt.GotOfFunc(inj, gf); err != nil {
			return
		}
	}
	return
}

// GotOfFunc ranges over given Dynamic object and its childrens
// recursively calling given GotFunc on every object that
// underlying database contains. The GotOfFunc never returns Missing*
// errors
func (c *Container) GotOfFunc(inj Dynamic, gf GotFunc) (err error) {
	val, err := c.ValueOf(inj)
	if err != nil {
		err = dropMissingError(err)
		return
	}
	if val.Kind() == reflect.Invalid { // nil-value
		return
	}
	if err = gf(inj.Schema); err != nil { // got schema of the inj
		if err == ErrStopRange {
			err = nil
		}
		return
	}
	if err = gf(inj.Object); err != nil { // got object of the inj
		if err == ErrStopRange {
			err = nil
		}
		return
	}
	if err = gotValueFunc(val, gf); err != nil {
		if err == ErrStopRange {
			err = nil
		} else {
			err = dropMissingError(err)
		}
	}
	return
}

func gotValueFunc(val *Value, gf GotFunc) (err error) {
	switch val.Schema().Kind() {
	case reflect.Slice, reflect.Array:
		var l int
		if l, err = val.Len(); err != nil {
			return
		}
		// we have schema of the object, but, anyway, we
		// need to check: got we the schema
		var es *Schema // element schema
		if es, err = val.Schema().Elem(); err != nil {
			return
		} else if es.isRegistered() {
			if sr, ok := val.c.reg.reg[es.Name()]; !ok {
				err = ErrInvalidSchemaOrData
				return
			} else if _, ok := val.c.get(sr); !ok {
				return // hasn't got (don't need extract objects without schema)
			} else if err = gf(sr); err != nil { // got
				return
			}
		}
		//
		for i := 0; i < l; i++ {
			var d *Value
			if d, err = val.Index(i); err != nil {
				return
			}
			if err = gotValueFunc(d, gf); err != nil {
				return
			}
		}
	case reflect.Struct:
		err = val.RangeFields(func(fname string, d *Value) error {
			// we have schema of the object, but, anyway, we
			// need to check: got we the schema
			if ss := d.Schema(); ss.isRegistered() {
				if sr, ok := val.c.reg.reg[ss.Name()]; !ok {
					return ErrInvalidSchemaOrData
				} else if _, ok := val.c.get(sr); !ok {
					// hasn't got (don't need extract objects without schema)
					return nil
				} else if gerr := gf(sr); gerr != nil { // got
					return gerr
				}
			}
			//
			return gotValueFunc(d, gf)
		})
	case reflect.Ptr:
		var v *Value
		switch val.s.Name() {
		case DYNAMIC:
			var dr Dynamic
			if dr, err = val.dynamic(); err != nil { // already validated
				return
			}
			if dr.IsBlank() { // nil value
				return
			}
			if _, ok := val.c.get(dr.Schema); ok {
				if err = gf(dr.Schema); err != nil { // got
					return
				}
			} else if _, ok := val.c.get(dr.Object); ok {
				// if no schema then no need to dereference,
				// but need to check does object exists
				err = gf(dr.Object) // got
				return
			} // else (got scema, but don't know about object)
			if v, err = val.dereferenceDynamic(dr); err != nil {
				return
			}
			if err = gf(dr.Object); err != nil { // got
				return
			}
			err = gotValueFunc(v, gf)
		case SINGLE:
			// take a look the schema of the reference
			var es *Schema
			if es, err = val.s.Elem(); err != nil {
				return
			} else if sr, ok := val.c.reg.reg[es.Name()]; !ok {
				err = ErrInvalidSchemaOrData
				return
			} else if _, ok := val.c.get(sr); !ok {
				return // hasn't got
			} else if err = gf(sr); err != nil { // got
				return
			}
			//
			var ref Reference
			if ref, err = val.static(); err != nil {
				return
			}
			if ref.IsBlank() { // nil-value
				return
			}
			if v, err = val.dereferenceStatic(ref); err != nil {
				return
			}
			if err = gf(ref); err != nil { // got
				return
			}
			err = gotValueFunc(v, gf)
		case ARRAY:
			// schema
			var es *Schema
			if es, err = val.Schema().Elem(); err != nil {
				return
			}
			if sr, ok := val.c.reg.reg[es.Name()]; !ok {
				err = ErrInvalidSchemaOrData
				return
			} else if err = gf(sr); err != nil { // got
				return
			}
			// range over the array
			var (
				ln    int
				shift int = 4                // length prefix
				s     int = len(Reference{}) // size of Reference, bytes
			)
			if ln, err = getLength(val.od); err != nil || ln == 0 {
				return
			}
			for i := 0; i < ln; i++ {
				if shift+s > len(val.od) {
					err = ErrInvalidSchemaOrData
					return
				}
				var ref Reference
				err = encoder.DeserializeRaw(val.od[shift:shift+s], &ref)
				if err != nil {
					return
				}
				shift += s         // shitf forward
				if ref.IsBlank() { // nil
					continue
				}
				if _, ok := val.c.get(ref); ok {
					if err = gf(ref); err != nil {
						return
					}
				}
			}
		default:
			err = ErrInvalidType
		}
	}
	return
}
