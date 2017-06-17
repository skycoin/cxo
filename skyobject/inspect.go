package skyobject

import (
	"encoding/hex"
	"fmt"
	"reflect"

	"github.com/disiqueira/gotree" // alpha
)

// Inspect returns string that contains printed
// object tree of this root object. The tree doesn't
// containe information about seq, timestamp and feed
// of the root
func (r *Root) Inspect() (tree string) {
	var rt gotree.GTStructure

	hash := r.Hash()
	rt.Name = "(root) " + hash.String()
	rt.Items = rootItems(r)
	tree = gotree.StringTree(rt)

	return
}

func rootItems(root *Root) (items []gotree.GTStructure) {
	vals, err := root.Values()
	if err != nil {
		items = []gotree.GTStructure{
			{Name: "error: " + err.Error()},
		}
		return
	}
	for _, val := range vals {
		items = append(items, valueItem(val))
	}
	return
}

func valueItem(val *Value) (item gotree.GTStructure) {
	if val.IsNil() {
		item.Name = "nil"
		return
	}

	var err error
	switch val.Kind() {
	case reflect.Struct:
		item, err = valueItemStruct(val)
	case reflect.Ptr:

		if val.Schema().ReferenceType() == ReferenceTypeSingle {
			var ref Reference
			if ref, err = val.Static(); err != nil {
				break
			}
			item.Name = "*(static) " + ref.String()
		} else { // dynamic reference
			var dr Dynamic
			if dr, err = val.Dynamic(); err != nil {
				break
			}
			item.Name = "*(dynamic) " + dr.String()
		}

		if val, err = val.Dereference(); err == nil {
			item.Items = []gotree.GTStructure{
				valueItem(val),
			}
		} else {
			item.Items = []gotree.GTStructure{
				{
					Name: "error: " + err.Error(),
				},
			}

		}
		return
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var i int64
		if i, err = val.Int(); err == nil {
			item.Name = fmt.Sprint(i)
		}
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var u uint64
		if u, err = val.Uint(); err == nil {
			item.Name = fmt.Sprint(u)
		}
	case reflect.Float32, reflect.Float64:
		var f float64
		if f, err = val.Float(); err == nil {
			item.Name = fmt.Sprint(f)
		}
	case reflect.Array:
		item, err = valueItemArray(val)
	case reflect.Slice:
		item, err = valueItemSlice(val)
	case reflect.String:
		var s string
		if s, err = val.String(); err == nil {
			item.Name = s
		}

	}
	if err != nil {
		item.Name = "error: " + err.Error()
		item.Items = nil // clear
	}
	return
}

func valueItemStruct(val *Value) (item gotree.GTStructure, err error) {
	item.Name = "struct " + val.Schema().Name()

	err = val.RangeFields(func(name string, val *Value) (_ error) {

		var field gotree.GTStructure
		field.Name = "(field) " + name
		field.Items = []gotree.GTStructure{
			valueItem(val),
		}
		item.Items = append(item.Items, field)
		return
	})
	return
}

func valueItemSlice(val *Value) (item gotree.GTStructure, err error) {
	// special case for []byte
	sch := val.Schema()
	if sch.Kind() == reflect.Slice && sch.Elem().Kind() == reflect.Uint8 {
		var bt []byte
		if bt, err = val.Bytes(); err != nil {
			return
		}
		item.Name = "[]byte"
		item.Items = []gotree.GTStructure{
			{Name: hex.EncodeToString(bt)},
		}
		return
	}
	// including References
	if item.Name = sch.Name(); item.Name == "" {
		// unnamed type
		if sch.Elem() == nil {
			err = ErrInvalidSchema
			return
		}
		item.Name = "[]" + sch.Elem().String()
	}
	err = val.RangeIndex(func(_ int, val *Value) (_ error) {
		item.Items = append(item.Items, valueItem(val))
		return
	})
	return
}

func valueItemArray(val *Value) (item gotree.GTStructure, err error) {
	item.Name = val.Schema().Name()
	if item.Name == "" {
		// unnamed type
		var ln int
		if ln, err = val.Len(); err != nil {
			return
		}
		if val.Schema().Elem() == nil {
			err = ErrInvalidSchema
			return
		}
		item.Name = fmt.Sprintf("[%d]", ln) + val.Schema().Elem().String()
	}
	err = val.RangeIndex(func(_ int, val *Value) (_ error) {
		item.Items = append(item.Items, valueItem(val))
		return
	})
	return
}
