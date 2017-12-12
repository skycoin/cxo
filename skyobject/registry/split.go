package registry

import (
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// A Splitter used by the node package
// to fill a Root object uinsg multiply
// connections. The Splitter provides
// Get method that lookups DB and if obejct
// doesn't exists, then it request the obejct
// using connectiosn from free list.
//
// The Splitter used to walking, but unlike the
// Walk the Splitter used to walk using many
// goroutines (the Walk is single-gourutine)
//
// The Get method returns not a real rc from DB.
// The rc is rc of obejcts that belongs to full
// Root obejcts. Thus, the rc can be used to
// skip subtrees of the Root tree that
// guaranteed in DB.
//
// The Splitter uses Acquire and Release methods
// from to limit number of parallel subtrees and
// wait for goroutines.
//
// The Splitter calls Splitter since it splits
// walking to all possible subtrees (unlike the
// Walk, that single-gourutine).
//
// So, the Splitter is CXO internal and unlikely
// useful outside
type Splitter interface {
	// Registry related to Root of the Splitter
	Registry() (reg *Registry)

	// Get value from DB or from related remote peers
	Get(key cipher.SHA256) (val []byte, rc uint32, err error)

	// Fail the splitting
	Fail(err error)

	//
	// goroutines limit and waiting
	//

	// Acquire is like (sync.WaitGroup).Add(1), but
	// the Acquire blocks if goroutins limit reached and
	// the Acquire returns false if the Splitter closed
	Acquire() (ok bool)
	// Release is like (sync.WaitGroup).Done()
	Release()
}

func splitSchemaHashAsync(
	s Splitter, //         : splitter
	sch Schema, //         : schema of the object
	hash cipher.SHA256, // : hash of the object
) {
	defer s.Release()
	splitSchemaHash(s, sch, hash)
}

func splitSchemaHash(
	s Splitter, //         : splitter
	sch Schema, //         : schema of the object
	hash cipher.SHA256, // : hash of the object
) {

	if hash == (cipher.SHA256{}) {
		return // nothing to split
	}

	var (
		rc  uint32
		val []byte
		err error
	)

	if val, rc, err = s.Get(hash); err != nil {
		s.Fail(err)
		return
	}

	if rc > 0 {
		return
	}

	// go deepper

	splitSchemaData(s, sch, val)

}

func splitSchemaDataAsync(
	s Splitter, // :
	sch Schema, // :
	val []byte, // :
) {
	defer s.Release()
	splitSchemaData(s, sch, val)
}

func splitSchemaData(
	s Splitter, // :
	sch Schema, // : schema of the object
	val []byte, // : encoded object
) {

	if sch.HasReferences() == false {
		return // no references, no walking
	}

	// the object represents Ref, Refs or Dynamic
	if sch.IsReference() == true {
		splitSchemaReference(s, sch, val)
		return
	}

	switch sch.Kind() {
	case reflect.Array:
		splitArray(s, sch, val)
	case reflect.Slice:
		splitSlice(s, sch, val)
	case reflect.Struct:
		splitStruct(s, sch, val)
	default:
		s.Fail(fmt.Errorf("invalid Schema to walk through: %s", sch))
	}

}

func splitSchemaReference(
	s Splitter, // :
	sch Schema, // : schema of the object
	val []byte, // : encoded reference
) {

	var err error

	switch rt := sch.ReferenceType(); rt {

	case ReferenceTypeSingle: // Ref

		var el Schema
		if el = sch.Elem(); el == nil {
			s.Fail(fmt.Errorf("sSchema of Ref with nil element: %s", sch))
			return
		}

		var ref Ref
		if err = encoder.DeserializeRaw(val, &ref); err != nil {
			s.Fail(err)
			return
		}

		ref.Split(s, el) // sync

	case ReferenceTypeSlice: // Refs

		var el Schema
		if el = sch.Elem(); el == nil {
			s.Fail(fmt.Errorf("Schema of Ref with nil element: %s", sch))
			return
		}

		var refs Refs
		if err = encoder.DeserializeRaw(val, &refs); err != nil {
			s.Fail(err)
			return
		}

		refs.Split(s, el) // sync

	case ReferenceTypeDynamic: // Dynamic

		var dr Dynamic
		if err = encoder.DeserializeRaw(val, &dr); err != nil {
			s.Fail(err)
			return
		}

		dr.Split(s) // sync

	default:

		s.Fail(fmt.Errorf("invalid ReferenceType %d to walk through", rt))

	}

}

func splitArray(
	s Splitter, // : pack to get
	sch Schema, // : schema of the array
	val []byte, // : encoded array
) {

	var el Schema // Schema of the element
	if el = sch.Elem(); el != nil {
		s.Fail(fmt.Errorf("Schema of element of array %q is nil", sch))
		return
	}

	splitArraySlice(s, el, sch.Len(), val)

}

func splitSlice(
	s Splitter, // : pack to get
	sch Schema, // : schema of the slice
	val []byte, // : encoded slice
) {

	var (
		ln  int // length of the slice
		err error
	)

	if ln, err = getLength(val); err != nil {
		s.Fail(err)
		return
	}

	var el Schema // Schema of the element
	if el = sch.Elem(); el != nil {
		s.Fail(fmt.Errorf("Schema of element of slice %q is nil", sch))
		return
	}

	splitArraySlice(s, el, ln, val[4:])
}

func splitArraySlice(
	s Splitter, // : pack to get
	el Schema, //  : shcema of an element
	ln int, //     : length of the array or slice (> 0)
	val []byte, // : encoded array or slice starting from first element
) {

	// doesn't need to walk through the zero-length
	// array or slice even if it contains references

	if ln == 0 {
		return
	}

	var (
		shift, m int
		err      error
	)

	for i := 0; i < ln; i++ {

		if shift > len(val) {
			err = fmt.Errorf("unexpected end of encoded  array or slice "+
				"of <%s>, length: %d, index: %d", el, ln, i)
			s.Fail(err)
			return
		}

		if m, err = el.Size(val[shift:]); err != nil {
			s.Fail(err)
			return
		}

		// split

		if s.Acquire() == false {
			return
		}

		go splitSchemaDataAsync(s, el, val[shift:shift+m])

		shift += m

	}

	return

}

func splitStruct(
	s Splitter, // : pack to get
	sch Schema, // : schema of the struct
	val []byte, // : encoded struct
) {

	var (
		shift, z int
		err      error
	)

	for i, fl := range sch.Fields() {

		if shift > len(val) {
			err = fmt.Errorf("unexpected end of encoded struct <%s>, "+
				"field number: %d, field name: %q, schema of field: %s",
				sch.String(),
				i,
				fl.Name(),
				fl.Schema().String())
			s.Fail(err)
			return
		}

		if z, err = fl.Schema().Size(val[shift:]); err != nil {
			s.Fail(err)
			return
		}

		if s.Acquire() == false {
			return
		}

		go splitSchemaDataAsync(s, fl.Schema(), val[shift:shift+z])

		shift += z

	}

}
