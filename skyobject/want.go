package skyobject

import (
	"reflect"
)

//
// set of keys
//

// A Set represents set of Reference(s)
type Set map[Reference]struct{}

// Add appends given key to the Set
func (s Set) Add(k Reference) {
	s[k] = struct{}{}
}

func (s *Set) AddMissing(k Reference, c *Container) {
	if _, ok := c.get(k); !ok {
		s.Add(k)
	}
}

// Err is works with MissingSchema and MissingObject errors.
// If given error is Missing* error then the Err extract key from the error
// and append the Key to the Set returning nil. If given error is not Missing*
// error then it returns the error
func (s *Set) Err(err error) error {
	switch x := err.(type) {
	case *MissingSchema:
		s.Add(x.Key())
	case *MissingObject:
		s.Add(x.Key())
	default:
		return err
	}
	return nil
}

// Want returns set of keys of missing objects. The set is empty if root is
// nil or full. The set can be incomplite.
func (r *Root) Want() (set Set, err error) {
	if r == nil {
		return // don't want anything (has no root object)
	}
	set = make(Set)
	var vs []*Value
	if vs, err = r.Values(); err != nil {
		err = set.Err(err)
		return
	}
	for _, val := range vs {
		if err = set.Err(wantValue(val, set)); err != nil {
			return
		}
	}
	return
}

func wantValue(val *Value, set Set) (err error) {
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
			if err = wantValue(d, set); err != nil {
				if err = set.Err(err); err != nil {
					return
				} // else -> continue
			}
		}
	case reflect.Struct:
		err = val.RangeFields(func(fname string, d *Value) error {
			return set.Err(wantValue(d, set))
		})
		if err != nil {
			return
		}
	case reflect.Ptr:
		var d *Value
		if d, err = val.Dereference(); err != nil {
			return
		}
		if err = wantValue(d, set); err != nil {
			return
		}
	}
	return
}
