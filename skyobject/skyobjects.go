package skyobject

import (
	"bytes"
	"fmt"
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/encoder"
	"github.com/skycoin/skycoin/src/cipher"
	"reflect"
	"strings"
)

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
	GetSchemaKey(typeName string) (cipher.SHA256, bool)
	Publish(ref Href, sign *cipher.Sig) cipher.SHA256

	Save(hr IHashObject) Href
	SaveObject(schemaKey cipher.SHA256, obj interface{}) cipher.SHA256
	SaveData(schemaKey cipher.SHA256, data []byte) cipher.SHA256
	Get(key cipher.SHA256) ([]byte, bool)
	GetRef(key cipher.SHA256) (cipher.SHA256, []byte)
	Set(key cipher.SHA256, data []byte) error
	Has(key cipher.SHA256) bool
	Statistic() data.Statistic
	GetAllBySchema(schemaKey cipher.SHA256) []cipher.SHA256
	RegisterSchema(tp ...interface{})
	Inspect()
	MissingDependencies(key cipher.SHA256) []cipher.SHA256

	LoadFields(key cipher.SHA256) map[string]string
}
type skyTypes struct {
	Name   string
	Schema cipher.SHA256
	Type   reflect.Type
}

type skyObjects struct {
	types []skyTypes
	ds    data.IDataSource
}

func SkyObjects(ds data.IDataSource) *skyObjects {
	result := &skyObjects{ds: ds, types: []skyTypes{}}
	result.RegisterSchema(HashRoot{}, HashObject{}, HashArray{})
	return result
}

func (s *skyObjects) Get(key cipher.SHA256) ([]byte, bool) {
	return s.ds.Get(key)
}

func (s *skyObjects) GetRef(key cipher.SHA256) (cipher.SHA256, []byte) {
	byteArray, ok := s.ds.Get(key)
	if ok == false {
		return cipher.SHA256{}, nil
	}
	var ref href
	encoder.DeserializeRaw(byteArray, &ref)
	return ref.Type, ref.Data
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
	key := cipher.SumSHA256(data)
	s.ds.Add(key, data)
	return key
}

//
func (s *skyObjects) SaveData(schemaKey cipher.SHA256, data []byte) cipher.SHA256 {
	h := href{Type: schemaKey, Data: data}
	refData := encoder.Serialize(h)
	key := cipher.SumSHA256(refData)
	s.ds.Add(key, refData)
	return key
}

func (s *skyObjects) Publish(ref Href, sign *cipher.Sig) cipher.SHA256 {
	root := newRoot(ref, *sign)
	return root.save(s).Ref
}

func (s *skyObjects) Save(hr IHashObject) Href {
	return hr.save(s)
}

func (s *skyObjects) RegisterSchema(types ...interface{}) {
	s.SaveObject(_schemaType, Schema{})
	for _, tp := range types {
		schema := ReadSchema(tp)
		schemaData := encoder.Serialize(schema)
		key := s.SaveData(_schemaType, schemaData)
		s.types = append(s.types, skyTypes{Name: schema.Name, Type: reflect.TypeOf(tp), Schema: key})
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
		data, _ := s.Get(k.Schema)
		var sm Schema
		encoder.DeserializeRaw(data, &sm)
		result = append(result, sm)
	}
	return result
}

func (c *skyObjects) GetAllBySchema(schemaKey cipher.SHA256) []cipher.SHA256 {
	query := func(key cipher.SHA256, data []byte) bool {
		return bytes.Compare(schemaKey[:32], data[0:32]) == 0
	}
	return c.ds.Where(query)
}

func (c *skyObjects) LoadFields(key cipher.SHA256) map[string]string {
	data, _ := c.ds.Get(key)
	ref := href{}
	encoder.DeserializeRaw(data, &ref)
	schemaData, _ := c.ds.Get(ref.Type)
	var sm Schema
	encoder.DeserializeRaw(schemaData, &sm)
	return encoder.ParseFields(ref.Data, sm.Fields)
}

func (c *skyObjects) Inspect() {
	query := func(key cipher.SHA256, data []byte) bool {
		hr := href{}
		encoder.DeserializeRaw(data, &hr)
		smKey := hr.Type
		smKey.Set(data[:32])

		var sm Schema = Schema{}
		if smKey == _schemaType {
			encoder.DeserializeRaw(hr.Data, &sm)
			fmt.Println("Schema object: ", sm)
		} else {
			schemaData, _ := c.ds.Get(smKey)
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
	c.ds.Where(query)
}

func (c *skyObjects) MissingDependencies(key cipher.SHA256) []cipher.SHA256 {
	result := []cipher.SHA256{}
	data, ok := c.Get(key)
	if !ok {
		return []cipher.SHA256{key}
	}

	typeKey := cipher.SHA256{}
	typeKey.Set(data[:32])
	if typeKey != _schemaType {
		result = append(result, c.MissingDependencies(typeKey)...)
		r := Href{Ref: key}
		if len(result) > 0 {
			result = append(result, key)
		}

		for k := range r.References(c) {
			if k != key {
				result = append(result, c.MissingDependencies(k)...)
			}
		}
	}
	return result
}
