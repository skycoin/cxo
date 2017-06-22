package skyobject

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// TAG is name of struct tag the skyobject package use to determine
// schema of a struct field if the field is reference
const TAG = "skyobject"

// ErrInvalidEncodedSchema occurs during decoding an invalid registry
var ErrInvalidEncodedSchema = errors.New("invalid encoded schema")

// A Registry represents registry of schemas.
// The Registry is thread safe only after call of
// Done method. The main approach is
//
//     r := NewRegistry()
//     r.Register("cxo.User", User{})
//     r.Register("cxo.List", List{})
//     // register all types need
//     r.Done()
//     // use all other methods of the Registry
//     // except Register
//
// NewContainer calls Done implicitly.
// I.e. there are four steps (1) create, (2) register all
// types need (3) call Done (or pass to the NewContainer),
// (4) use all other methods of Registry
// And (4) step is thread safe because the other methods
// are read only. A server-side container can remove Registry
// if no Root objects refers to the Registry
type Registry struct {
	done bool                       // stop registration and use
	ref  RegistryReference          // reference to the registry
	reg  map[string]Schema          // by name
	srf  map[SchemaReference]Schema // by reference (for Dynamic references)

	// local
	tn map[reflect.Type]string // type -> registered name
}

// create registry without tn map
func newRegistry() (r *Registry) {
	r = new(Registry)
	r.reg = make(map[string]Schema)
	r.srf = make(map[SchemaReference]Schema)
	return
}

// NewRegistry creates fresh empty registry
func NewRegistry() (r *Registry) {
	r = newRegistry()

	r.tn = make(map[reflect.Type]string) // local
	return
}

// DecodeRegistry decodes registry. It's impossible to
// use SchemaByInterface of an decoded Registry. A decoded
// Registry already Done
func DecodeRegistry(b []byte) (r *Registry, err error) {
	var (
		res = registryEntities{}
		s   Schema
	)
	if err = encoder.DeserializeRaw(b, &res); err != nil {
		return
	}
	r = newRegistry()
	for _, re := range res {
		s, err = decodeSchema(re.Schema)
		r.reg[re.Name] = s
		r.srf[s.Reference()] = s
	}
	r.Done() // done it
	return
}

// Register a struct type using provided name. It panics if something
// wrong. It can register any invalid type and panic later during Done
func (r *Registry) Register(name string, i interface{}) {
	if r.done {
		panic("registration has been closed")
	}
	if name == "" {
		panic("empty name")
	}
	typ := typeOf(i)
	switch typ {
	case singleRef, sliceRef, dynamicRef:
		panic("can't register reference type")
	default:
	}

	for _, n := range r.tn {
		if n == name {
			panic("this name already registered")
		}
	}

	r.tn[typ] = name
}

// range over registered types, and create schemas
func (r *Registry) register() {

	for typ, name := range r.tn {

		s := r.getSchema(typ)

		// only named structures
		if !s.IsRegistered() {
			panic("can't register type: " + typ.Name())
		}

		// set registered name instead of type name (not nessessary)
		s.(*structSchema).name = []byte(name)

		if rr, ok := r.reg[name]; ok {
			if rr.Reference() != s.Reference() {
				panic("another type already registered with the name")
			}
		} else {
			r.reg[name] = s // store: name -> Scehma
		}

	}

}

// Done closes registration
func (r *Registry) Done() {
	if r.done == true {
		return // already done
	}

	r.done = true

	r.register()

	filled := make(map[Schema]struct{})
	for _, sch := range r.reg {
		r.fillSchema(sch, filled)
	}

	// fill up map by SchemaReference
	for _, sch := range r.reg {
		r.srf[sch.Reference()] = sch
	}

	r.ref = RegistryReference(cipher.SumSHA256(r.Encode()))
}

// SchemaByName returns schema by name or "missing schema" error
// (use it after Done)
func (r *Registry) SchemaByName(name string) (Schema, error) {
	r.mustBeDone()
	return r.schemaByName(name)
}

// SchemaByReference returns Schema by SchemaReference that is obvious
func (r *Registry) SchemaByReference(sr SchemaReference) (s Schema, err error) {
	r.mustBeDone()
	var ok bool
	if s, ok = r.srf[sr]; !ok {
		err = fmt.Errorf("miisng schema %q", sr.String())
	}
	return
}

// SchemaReferenceByName returns SchemaReference by registered name of a struct
// type
func (r *Registry) SchemaReferenceByName(name string) (sr SchemaReference,
	err error) {

	r.mustBeDone()
	var s Schema
	if s, err = r.schemaByName(name); err != nil {
		return
	}
	sr = s.Reference()
	return
}

// Encode registry to send (use it after Done)
func (r *Registry) Encode() []byte {
	r.mustBeDone()
	if len(r.reg) == 0 {
		return encoder.Serialize(registryEntities{}) // empty
	}
	ent := make(registryEntities, 0, len(r.reg))
	for name, sch := range r.reg {
		ent = append(ent, registryEntity{name, sch.Encode()})
	}
	sort.Sort(ent)
	return encoder.Serialize(ent)
}

// Reference to the register (use it after Done)
func (r *Registry) Reference() RegistryReference {
	r.mustBeDone()
	return r.ref
}

// set proper references for schemas that has references to
// another schemas, such as arrays, slices and structs
func (r *Registry) fillSchema(s Schema, filled map[Schema]struct{}) {
	if _, ok := filled[s]; ok {
		return // already
	}
	filled[s] = struct{}{} // filling
	var err error
	if s.IsReference() {
		switch s.ReferenceType() {
		case ReferenceTypeSingle, ReferenceTypeSlice:
			x := s.(*referenceSchema)
			x.elem, err = r.schemaByName(x.elem.Name())
			if err != nil {
				panic(err)
			}
			r.fillSchema(x.elem, filled)
		case ReferenceTypeDynamic:
			// do nothing
		default:
			panic("invalid reference: " + s.String())
		}
		return
	}
	switch s.Kind() {
	case reflect.Array:
		x := s.(*arraySchema)
		if s.Elem().IsRegistered() {
			x.elem, err = r.schemaByName(s.Elem().Name())
			if err != nil {
				panic(err)
			}
		}
		r.fillSchema(x.elem, filled)
	case reflect.Slice:
		x := s.(*sliceSchema)
		if s.Elem().IsRegistered() {
			x.elem, err = r.schemaByName(s.Elem().Name())
			if err != nil {
				panic(err)
			}
		}
		r.fillSchema(x.elem, filled)
	case reflect.Struct:
		for i, f := range s.Fields() {
			x := f.(*field)
			if fs := f.Schema(); fs.IsRegistered() {
				x.schema, err = r.schemaByName(fs.Name())
				if err != nil {
					panic(err)
				}
			}
			r.fillSchema(x.schema, filled)
			s.(*structSchema).fields[i] = x
		}
	}
}

func (r *Registry) mustBeDone() {
	if r.done != true {
		panic("call Done first")
	}
}

func (r *Registry) schemaByName(name string) (s Schema, err error) {
	var ok bool
	if s, ok = r.reg[name]; !ok {
		err = fmt.Errorf("missing schema %q", name)
	}
	return
}

// use (reflect.Type).Name() or name provided to Register;
// if there aren't, then return nil
func (r *Registry) typeName(typ reflect.Type) []byte {
	if name, ok := r.tn[typ]; ok {
		return []byte(name)
	}
	if name := typ.Name(); name != "" {
		return []byte(name)
	}
	return nil
}

func (r *Registry) getSchema(typ reflect.Type) Schema {

	s := new(schema)

	s.kind = typ.Kind()
	s.name = r.typeName(typ)

	switch typ.Kind() {

	case reflect.Bool, reflect.Int8, reflect.Uint8,
		reflect.Int16, reflect.Uint16,
		reflect.Int32, reflect.Uint32, reflect.Float32,
		reflect.Int64, reflect.Uint64, reflect.Float64,
		reflect.String:

		// do nothing for flat types

		return s

	case reflect.Slice:

		// get schema of element

		ss := new(sliceSchema)
		ss.schema = *s

		el := r.getSchema(typ.Elem())

		if el.IsRegistered() {
			ss.elem = &schema{SchemaReference{}, el.Kind(), el.RawName()}
			return ss
		}

		ss.elem = el
		return ss

	case reflect.Array:

		// get schema of element and length

		as := new(arraySchema)
		as.schema = *s
		as.length = typ.Len()

		el := r.getSchema(typ.Elem())

		if el.IsRegistered() {
			as.elem = &schema{SchemaReference{}, el.Kind(), el.RawName()}
			return as
		}

		as.elem = el
		return as

	case reflect.Struct:

		// get schemas of fields

		ss := new(structSchema)
		ss.schema = *s

		for i, nf := 0, typ.NumField(); i < nf; i++ {

			sf := typ.Field(i)
			if sf.Tag.Get("enc") == "-" || sf.PkgPath != "" || sf.Name == "_" {
				continue
			}
			ss.fields = append(ss.fields, r.getField(sf))

		}

		return ss

	default:
	}

	panic("invlaid type: " + typ.String())

}

func (r *Registry) getField(sf reflect.StructField) Field {

	f := new(field)

	f.name = []byte(sf.Name)
	f.tag = []byte(sf.Tag)

	t := sf.Type // reflect.Type

	switch t {

	case singleRef:

		// reference

		tagRef := tagReference(sf.Tag)

		f.schema = &referenceSchema{
			schema: schema{
				ref:  SchemaReference{},
				kind: t.Kind(),
			},
			typ:  ReferenceTypeSingle,
			elem: &schema{kind: reflect.Struct, name: []byte(tagRef)},
		}

		return f

	case sliceRef:

		// references

		tagRef := tagReference(sf.Tag)
		f.schema = &referenceSchema{
			schema: schema{
				ref:  SchemaReference{},
				kind: t.Kind(),
			},
			typ:  ReferenceTypeSlice,
			elem: &schema{kind: reflect.Struct, name: []byte(tagRef)},
		}

		return f

	case dynamicRef:

		// dynamic reference

		f.schema = &referenceSchema{
			schema: schema{
				ref:  SchemaReference{},
				kind: t.Kind(),
			},
			typ: ReferenceTypeDynamic,
		}

		return f

	default:
	}

	if s := r.getSchema(sf.Type); s.IsRegistered() {
		f.schema = &schema{SchemaReference{}, s.Kind(), s.RawName()}
	} else {
		f.schema = s
	}

	return f

}

func tagReference(tag reflect.StructTag) string {
	skytag := tag.Get(TAG)
	if skytag == "" {
		panic(`empty skyobject tag, expected "schema=XXX`)
	}
	for _, part := range strings.Split(skytag, ",") {
		if !strings.HasPrefix(part, "schema=") {
			continue
		}
		ss := strings.Split(part, "=")
		if len(ss) != 2 {
			panic("invalid schema tag: " + part)
		}
		if ss[1] == "" {
			panic("empty tag schema name: " + part)
		}
		return ss[1]
	}
	panic("invalid skyobject tag: " + skytag)
}

func typeOf(i interface{}) reflect.Type {
	return reflect.Indirect(reflect.ValueOf(i)).Type()
}

// decode schema

func decodeSchema(b []byte) (s Schema, err error) {
	// type encodedSchema struct {
	// 	RefTyp uint32
	// 	Kind   uint32
	// 	Name   []byte
	// 	Len    uint32
	// 	Fields [][]byte
	// 	Elem   []byte // encoded schema
	// }
	//
	// type encodedField struct {
	// 	Name   []byte
	// 	Tag    []byte
	// 	Schema []byte
	// }

	var x encodedSchema
	if err = encoder.DeserializeRaw(b, &x); err != nil {
		return
	}
	// is reference
	switch ReferenceType(x.RefTyp) {
	case ReferenceTypeSingle, ReferenceTypeSlice, ReferenceTypeDynamic:
		// kind, typ, elem
		rs := referenceSchema{}
		rs.kind = reflect.Kind(x.Kind)
		rs.typ = ReferenceType(x.RefTyp)
		if rs.typ != ReferenceTypeDynamic {
			if rs.elem, err = decodeSchema(x.Elem); err != nil {
				return
			}
		}
		s = &rs
		return
	case ReferenceTypeNone: // not a reference
	default:
		err = ErrInvalidEncodedSchema
		return
	}

	sc := schema{
		kind: reflect.Kind(x.Kind),
		name: x.Name,
	}

	switch k := reflect.Kind(x.Kind); k {
	case reflect.Slice:
		ss := sliceSchema{}
		ss.schema = sc
		if ss.elem, err = decodeSchema(x.Elem); err != nil {
			return
		}
		s = &ss
	case reflect.Array:
		as := arraySchema{}
		as.schema = sc
		as.length = int(x.Len)
		if as.elem, err = decodeSchema(x.Elem); err != nil {
			return
		}
		s = &as
	case reflect.Struct:
		ss := structSchema{}
		ss.schema = sc
		var f Field
		for _, ef := range x.Fields {
			if f, err = decodeField(ef); err != nil {
				return
			}
			ss.fields = append(ss.fields, f)
		}
		s = &ss
	default:
		s = &sc
	}

	return
}

func decodeField(b []byte) (f Field, err error) {
	var ef encodedField
	if err = encoder.DeserializeRaw(b, &ef); err != nil {
		return
	}
	ff := field{}
	ff.name = ef.Name
	ff.tag = ef.Tag
	if ff.schema, err = decodeSchema(ef.Schema); err != nil {
		return
	}
	f = &ff
	return
}

// encode

type registryEntity struct {
	Name   string
	Schema []byte
}

type registryEntities []registryEntity

// for sort.Sort

func (r registryEntities) Len() int {
	return len(r)
}

func (r registryEntities) Less(i, j int) bool {
	return r[i].Name < r[j].Name
}

func (r registryEntities) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
