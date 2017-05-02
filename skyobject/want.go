package skyobject

import (
	"reflect"
)

// ValueOf returns value of given Dynamic object
func (c *Container) ValueOf(dr Dynamic) (val *Value, err error) {
	var sd, od []byte
	var ok bool
	var s *Schema
	// is the dynamic reference valid
	if !dr.IsValid() {
		err = ErrInvalidReference
		return
	}
	// is it blank
	if dr.IsBlank() {
		val = nilValue(c, nil) // no value, nor schema
		return
	}
	// obtain schema of the dynamic reference
	if sd, ok = c.get(dr.Schema); !ok {
		err = &MissingSchema{dr.Schema}
		return
	}
	// decode the schema
	s = c.reg.newSchema()
	if err = s.Decode(sd); err != nil {
		return
	}
	// obtain object of the dynamic reference
	if od, ok = c.get(dr.Object); !ok {
		err = &MissingObject{key: dr.Object, schemaName: s.Name()}
		return
	}
	// create value
	val = &Value{c, s, od}
	return
}

// A WantFunc represents function that used in
// (*Root).WantFunc() and (*Container).WantOfFunc()
// methods. If the function returns an error
// then caller terminates and returns
// the error. There is special case ErrStopRange
// that used to break the itteration without
// returning the error
type WantFunc func(hash Reference) (err error)

func errWantFunc(err error, wf WantFunc) error {
	switch x := err.(type) {
	case *MissingSchema:
		return wf(x.Key())
	case *MissingObject:
		return wf(x.Key())
	}
	return err
}

// WantFunc recursively calls given WantFunc on every
// object that the Root object hasn't got, but knows about
func (r *Root) WantFunc(wf WantFunc) (err error) {
	for _, inj := range r.Refs {
		if err = r.cnt.WantOfFunc(inj, wf); err != nil {
			return
		}
	}
	return
}

// WantOfFunc ranges over given Dynamic object and its childrens
// recursively calling given WantFunc on every object that
// the container hasn't got, but knows about
func (c *Container) WantOfFunc(inj Dynamic, wf WantFunc) (err error) {
	val, err := c.ValueOf(inj)
	if err != nil {
		if err = errWantFunc(err, wf); err == ErrStopRange {
			err = nil
		}
		return
	}
	if val.Kind() == reflect.Invalid { // nil-value
		return
	}
	if err = errWantFunc(wantValueFunc(val, wf), wf); err == ErrStopRange {
		err = nil
	}
	return
}

func wantValueFunc(val *Value, wf WantFunc) (err error) {
	switch val.Kind() {
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
			if err = errWantFunc(wantValueFunc(d, wf), wf); err != nil {
				return
			}
		}
	case reflect.Struct:
		err = val.RangeFields(func(fname string, d *Value) error {
			return errWantFunc(wantValueFunc(d, wf), wf)
		})
	case reflect.Ptr:
		var d *Value
		if d, err = val.Dereference(); err != nil {
			return
		}
		err = wantValueFunc(d, wf)
	}
	return
}
