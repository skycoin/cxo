package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"reflect"
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/encoder"
	"bytes"
	"strings"
	"fmt"
)

//TODO: Split on implementation on something like
// container.Schema.All()
// container.Schema.Where()
// container.Objects.All()
// container.Objects.Where()
type ISkyObjects interface {
	HashObject(schemaKey cipher.SHA256, data []byte) (IHashObject, bool)
	ValidateHashType(typeName string) bool

	//CreateBySchema(schemaKey cipher.SHA256) (interface{}, bool)
	//CreateByType(typeName string) (interface{}, bool)
	GetSchemas() []Schema
	//GetSchemaKey(typeName string) cipher.SHA256
	GetSchema(typeName string) *Schema
	GetSchemaKey(typeName string) (cipher.SHA256, bool)

	Publish(ref Href, sign *cipher.Sig) (cipher.SHA256)

	//SignRoot(hr IHashObject) HashRoot
	Save(hr IHashObject) Href
	SaveObject(schemaKey cipher.SHA256, obj interface{}) (cipher.SHA256)
	SaveData(schemaKey cipher.SHA256, data []byte) (cipher.SHA256)
	Get(key cipher.SHA256) ([]byte, bool)
	Has(key cipher.SHA256) bool
	Statistic() *data.Statistic
	GetAllBySchema(schemaKey cipher.SHA256) []cipher.SHA256
	RegisterSchema(tp ...interface{})
	Inspect()

	LoadFields(key cipher.SHA256) (map[string]string)
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
	result := &skyObjects{ds:ds, types:[]skyTypes{}}
	result.RegisterSchema(HashRoot{}, HashObject{}, HashArray{})
	return result
}

func (s *skyObjects) Get(key cipher.SHA256) ([]byte, bool) {
	return s.ds.Get(key)
}

func (s *skyObjects) Has(key cipher.SHA256) bool {
	return s.ds.Has(key)
}
//
//func (s *skyObjects) SignRoot(hr IHashObject) HashRoot {
//	root := HashRoot{}
//	//hr.Save(s)
//	return root
//}

func (s *skyObjects) SaveObject(schemaKey cipher.SHA256, obj interface{}) (cipher.SHA256) {
	h := href{Type:schemaKey, Data:encoder.Serialize(obj)}
	data := encoder.Serialize(h)
	key := cipher.SumSHA256(data)
	s.ds.Add(key, data)
	return key
}

func (s *skyObjects) SaveData(schemaKey cipher.SHA256, data []byte) (cipher.SHA256) {
	h := href{Type:schemaKey, Data:data}
	refData := encoder.Serialize(h)
	key := cipher.SumSHA256(refData)
	s.ds.Add(key, data)
	return key
}

func (s *skyObjects) Publish(ref Href, sign *cipher.Sig) (cipher.SHA256) {
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
		key := s.SaveObject(_schemaType, schemaData)
		s.types = append(s.types, skyTypes{Name:schema.Name, Type:reflect.TypeOf(tp), Schema:key})
	}
}

func (s *skyObjects) typeByName(name string) (reflect.Type, bool) {
	for _, tp := range s.types {
		if (tp.Name == name) {
			return tp.Type, true
		}
	}
	return nil, false
}

func (s *skyObjects) typeBySchema(key cipher.SHA256) (reflect.Type, bool) {
	for _, tp := range s.types {
		if (tp.Schema == key) {
			return tp.Type, true
		}
	}
	return nil, false
}

func (s *skyObjects) HashObject(schemaKey cipher.SHA256, data []byte) (IHashObject, bool) {
	r, ok := s.typeBySchema(schemaKey)
	if (ok) {
		res := reflect.New(r)
		if (!res.IsNil() && res.IsValid()) {
			resValue, ok := res.Interface().(IHashObject)
			if (ok) {
				resValue.SetData(data)
				return resValue, true
			}
		}
	}
	return nil, false
}

func (s *skyObjects) ValidateHashType(typeName string) bool {
	for _, tp := range s.types {
		if (strings.ToLower(tp.Name) == strings.ToLower(typeName)) {
			return true
		}
	}
	return false
}

func (s *skyObjects) Statistic() *data.Statistic {
	return s.ds.Statistic()
}

func (s *skyObjects) GetSchema(typeName string) *Schema {
	for _, tp := range s.types {
		if (strings.ToLower(tp.Name) == strings.ToLower(typeName)) {
			var sm Schema
			data, _ := s.Get(tp.Schema)
			encoder.DeserializeRaw(data, &sm)
			return &sm
		}
	}
	return nil
}

func (s *skyObjects) GetSchemaKey(typeName string) (cipher.SHA256, bool) {
	for _, tp := range s.types {
		if (strings.ToLower(tp.Name) == strings.ToLower(typeName)) {
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

func (c *skyObjects) LoadFields(key cipher.SHA256) (map[string]string) {
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

		smKey := cipher.SHA256{}
		smKey.Set(data[:32])
		schemaData, ok := c.ds.Get(smKey)
		if (ok) {
			var sm Schema
			encoder.DeserializeRaw(schemaData, &sm)
			fmt.Println("Schema", sm, " for data: ")
		}

		return false
	}
	c.ds.Where(query)
}
