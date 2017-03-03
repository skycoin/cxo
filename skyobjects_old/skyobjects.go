package skyobject

import (
	"bytes"
	"fmt"

	"reflect"
	"strings"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// ISkyObjects is the SkyObjects container interface.
//TODO: Split on implementation on something like
// container.Schema.All()
// container.Schema.Where()
// container.Objects.All()
// container.Objects.Where()
type ISkyObjects interface {
	HashObject(ref href) (IHashObject, bool)
	ValidateHashType(typeName string) bool

	GetSchemas() []Schema
	GetSchema(typeName string) *Schema
	GetSchemaFromKey(schemaKey cipher.SHA256) *Schema
	GetSchemaKey(typeName string) (cipher.SHA256, bool)
	Publish(ref Href, sign *cipher.Sig) cipher.SHA256

	Save(hr IHashObject) Href
	SaveToRoot(hr IHashObject) Href
	SaveObject(schemaKey cipher.SHA256, obj interface{}) cipher.SHA256
	SaveData(schemaKey cipher.SHA256, data []byte) cipher.SHA256
	Get(key cipher.SHA256) ([]byte, bool)
	GetObjRef(key cipher.SHA256) *ObjRef
	Set(key cipher.SHA256, data []byte) error
	Has(key cipher.SHA256) bool
	Statistic() data.Statistic
	GetAllBySchema(schemaKey cipher.SHA256) []cipher.SHA256
	RegisterSchema(tp ...interface{})
	Inspect()
	MissingDependencies(key cipher.SHA256) []cipher.SHA256

	LoadFields(key cipher.SHA256) map[string]string

	getDataSource() data.IDataSource
	setRoot(key cipher.SHA256)
}

type skyTypes struct {
	Name   string
	Schema cipher.SHA256
	Type   reflect.Type
}

type skyObjects struct {
	types   []skyTypes
	ds      data.IDataSource
	root    cipher.SHA256
	rootSeq uint64
}

// SkyObjects creates a container.
func SkyObjects(ds data.IDataSource) ISkyObjects {
	result := &skyObjects{ds: ds, types: []skyTypes{}}
	result.RegisterSchema(HashRoot{}, HashObject{}, HashArray{})
	result.Inspect()
	return result
}

func (s *skyObjects) Get(key cipher.SHA256) ([]byte, bool) {
	return s.ds.Get(key)
}

func (s *skyObjects) GetObjRef(key cipher.SHA256) (objref *ObjRef) {
	var ref href
	data, ok := s.Get(key)
	if ok == false {
		return
	}
	encoder.DeserializeRaw(data, &ref)

	return &ObjRef{
		key:       key,
		schemaKey: ref.Type,
		data:      ref.Data,
		c:         s,
	}
}

func (s *skyObjects) Set(key cipher.SHA256, data []byte) error {
	return s.ds.Add(key, data)
}

func (s *skyObjects) Has(key cipher.SHA256) bool {
	return s.ds.Has(key)
}

func (s *skyObjects) SaveObject(schemaKey cipher.SHA256, obj interface{}) cipher.SHA256 {
	h := href{Type: schemaKey, Data: encoder.Serialize(obj)}
	data := encoder.Serialize(h)
	return s.ds.AddAutoKey(data)
}

//
func (s *skyObjects) SaveData(schemaKey cipher.SHA256, data []byte) cipher.SHA256 {
	h := href{Type: schemaKey, Data: data}
	refData := encoder.Serialize(h)
	return s.ds.AddAutoKey(refData)
}

func (s *skyObjects) Publish(ref Href, sign *cipher.Sig) cipher.SHA256 {
	// root := newRoot(ref, *sign)
	// return root.save(s).Ref
	return cipher.SHA256{}
}

func (s *skyObjects) Save(hr IHashObject) Href {
	return hr.save(s)
}

func (s *skyObjects) SaveToRoot(hr IHashObject) Href {
	ref := hr.save(s)
	if s.rootSeq == 0 {
		// New Root object.
		linkedArray := HashArray{}
		linkedArray.rtype = encoder.Serialize(ReadSchema(HashArray{}))
		linkedArray.rdata = encoder.Serialize([]cipher.SHA256{ref.Ref})
		typeKey := s.SaveData(_schemaType, linkedArray.rtype)
		linkedArray.Ref = s.SaveData(typeKey, linkedArray.rdata)
		// linked := Href(*linkedArray)

		deletedArray := HashArray{}
		deletedArray.rtype = encoder.Serialize(ReadSchema(HashArray{}))
		deletedArray.rdata = encoder.Serialize([]cipher.SHA256{})
		deletedArray.Ref = s.SaveData(typeKey, deletedArray.rdata)
		// deleted := Href(*linkedArray)

		root := newRoot(linkedArray, deletedArray, s.rootSeq+1)
		rootRef := root.save(s)
		s.root = rootRef.Ref
	} else {
		// TODO
	}
	return ref
}

func (s *skyObjects) RegisterSchema(types ...interface{}) {
	s.SaveObject(_schemaType, Schema{})
	for _, tp := range types {
		schema := ReadSchema(tp)
		schemaData := encoder.Serialize(schema)
		key := s.SaveData(_schemaType, schemaData)
		s.types = append(s.types, skyTypes{
			Name:   schema.Name,
			Type:   reflect.TypeOf(tp),
			Schema: key,
		})
	}
}

func (s *skyObjects) typeByName(name string) (reflect.Type, bool) {
	for _, tp := range s.types {
		if tp.Name == name {
			return tp.Type, true
		}
	}
	return nil, false
}

func (s *skyObjects) typeBySchema(key cipher.SHA256) (reflect.Type, bool) {
	for _, tp := range s.types {
		if tp.Schema == key {
			return tp.Type, true
		}
	}
	return nil, false
}

func (s *skyObjects) HashObject(ref href) (IHashObject, bool) {
	r, ok := s.typeBySchema(ref.Type)
	if ok {
		res := reflect.New(r)
		if !res.IsNil() && res.IsValid() {
			resValue, ok := res.Interface().(IHashObject)
			if ok {
				typeData, _ := s.Get(ref.Type)
				resValue.SetData(typeData, ref.Data)
				return resValue, true
			}
		}
	}
	return nil, false
}

func (s *skyObjects) ValidateHashType(typeName string) bool {
	for _, tp := range s.types {
		if strings.ToLower(tp.Name) == strings.ToLower(typeName) {
			return true
		}
	}
	return false
}

func (s *skyObjects) Statistic() data.Statistic {
	return s.ds.Statistic()
}

func (s *skyObjects) GetSchema(typeName string) *Schema {
	res := Schema{}
	//TODO: Optimize query
	query := func(key cipher.SHA256, data []byte) bool {
		hr := href{}
		encoder.DeserializeRaw(data, &hr)
		smKey := hr.Type
		smKey.Set(data[:32])

		sm := Schema{}
		if smKey == _schemaType {
			encoder.DeserializeRaw(hr.Data, &sm)
			if strings.ToLower(sm.Name) == strings.ToLower(typeName) {
				res = sm
				return true
			}

		}
		return false
	}
	s.ds.Where(query)
	return &res

}

func (s *skyObjects) GetSchemaFromKey(schemaKey cipher.SHA256) *Schema {
	var res Schema
	var ref href

	dataBytes, ok := s.ds.Get(schemaKey)
	if ok == false {
		return &res
	}

	encoder.DeserializeRaw(dataBytes, &ref)
	smKey := ref.Type
	smKey.Set(dataBytes[:32])

	if smKey == _schemaType {
		encoder.DeserializeRaw(ref.Data, &res)
	}

	return &res
}

func (s *skyObjects) GetSchemaKey(typeName string) (cipher.SHA256, bool) {
	for _, tp := range s.types {
		if strings.ToLower(tp.Name) == strings.ToLower(typeName) {
			return tp.Schema, true
		}
	}
	return cipher.SHA256{}, false
}

func (s *skyObjects) GetSchemas() []Schema {
	result := []Schema{}
	for _, k := range s.types {
		result = append(result, *s.GetSchemaFromKey(k.Schema))
	}
	return result
}

func (s *skyObjects) GetAllBySchema(schemaKey cipher.SHA256) []cipher.SHA256 {
	query := func(key cipher.SHA256, data []byte) bool {
		return bytes.Compare(schemaKey[:32], data[0:32]) == 0
	}
	return s.ds.Where(query)
}

func (s *skyObjects) LoadFields(key cipher.SHA256) map[string]string {
	data, _ := s.ds.Get(key)
	ref := href{}
	encoder.DeserializeRaw(data, &ref)
	schemaData, _ := s.ds.Get(ref.Type)
	var sm Schema
	encoder.DeserializeRaw(schemaData, &sm)
	return encoder.ParseFields(ref.Data, sm.Fields)
}

func (s *skyObjects) Inspect() {
	query := func(key cipher.SHA256, data []byte) bool {
		hr := href{}
		encoder.DeserializeRaw(data, &hr)
		smKey := hr.Type
		smKey.Set(data[:32])

		var sm = Schema{}
		if smKey == _schemaType {
			encoder.DeserializeRaw(hr.Data, &sm)
			fmt.Println("Schema object: ", sm)
		} else {
			schemaData, _ := s.ds.Get(smKey)
			shr := href{}
			encoder.DeserializeRaw(schemaData, &shr)
			if shr.Type != _schemaType {
				panic("Reference mast be an schema type")
			}
			encoder.DeserializeRaw(shr.Data, &sm)
			fmt.Println("Object type", sm)
		}
		return false
	}
	s.ds.Where(query)
}

func (s *skyObjects) getDataSource() data.IDataSource {
	return s.ds
}

func (s *skyObjects) setRoot(key cipher.SHA256) {
	s.root = key
}

func (s *skyObjects) MissingDependencies(key cipher.SHA256) []cipher.SHA256 {
	result := []cipher.SHA256{}
	data, ok := s.Get(key)
	if !ok {
		return []cipher.SHA256{key}
	}

	typeKey := cipher.SHA256{}
	typeKey.Set(data[:32])
	if typeKey != _schemaType {
		result = append(result, s.MissingDependencies(typeKey)...)
		r := Href{Ref: key}
		if len(result) > 0 {
			result = append(result, key)
		}

		for k := range r.References(s) {
			if k != key {
				result = append(result, s.MissingDependencies(k)...)
			}
		}
	}
	return result
}
