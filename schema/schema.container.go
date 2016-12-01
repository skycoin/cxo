package schema

import (
	"github.com/skycoin/cxo/data"
	"fmt"
	"errors"
	"github.com/skycoin/skycoin/src/cipher"
	"strings"
	"github.com/skycoin/cxo/encoder"
)

type Container struct {
	ds    data.IDataSource
	Root  Href
	types map[string]cipher.SHA256
}

func NewContainer(ds data.IDataSource) *Container {
	return &Container{ds:ds, types:map[string]cipher.SHA256{}}
}

func (c *Container) Register(value interface{}) {
	schema := ExtractSchema(value)
	key, _ := c.saveObj(schema)
	c.types[strings.ToLower(string(schema.StructName))] = key
}

func (c *Container) SaveRoot(value interface{}) Href {
	sm := ExtractSchema(value)
	smKey, _ := c.saveObj(sm)
	key, _ := c.saveObj(value)
	c.Root = Href{Hash:key, Type:smKey}

	return c.Root
}

func (c *Container) GetSchema(name string) (*Schema, error) {
	key, ok := c.types[name]
	if (ok) {
		schema := &Schema{}
		err := c.Load(key, schema)
		if (err == nil) {
			return schema, nil
		}

	}
	return nil, errors.New("Schema does not exist")
}

func (c *Container) GetSchemaKey(name string) (cipher.SHA256, error) {
	key, ok := c.types[name]
	if !ok {
		return cipher.SHA256{}, errors.New("Schema does not exist")
	}
	return key, nil
}

func (c *Container) LoadFields(key cipher.SHA256, schema *Schema) (map[string]string) {
	data, _ := c.ds.Get(key)
	return encoder.ParseFields(data, schema.StructFields)
}

func (c *Container) saveObj(value interface{}) (cipher.SHA256, error) {
	data := encoder.Serialize(value)
	key := CreateKey(data)

	if c.Has(key) {
		return key, fmt.Errorf("key already present: %v", key)
	}
	e := c.Add(key, data)
	if (e != nil) {
		return key, e
	}
	return key, nil
}

func (c *Container) Save(value interface{}) (Href, error) {
	key, e := c.saveObj(value)

	if (e != nil) {
		return Href{}, e
	}

	return c.CreateRef(key, value), nil
}

func (c *Container) Load(key cipher.SHA256, data interface{}) error {
	value, ok := c.Get(key)
	if !ok {
		return errors.New("Object does not exist")
	}
	encoder.DeserializeRaw(value, data)
	return nil
}

func (c *Container) Add(key cipher.SHA256, data []byte) error {
	return c.ds.Add(cipher.SHA256(key), data)
}

func (c *Container) Get(key cipher.SHA256) ([]byte, bool) {
	return c.ds.Get(cipher.SHA256(key))
}

func (c *Container) Has(key cipher.SHA256) bool {
	return c.ds.Has(cipher.SHA256(key))
}

func (c *Container) CreateRef(key cipher.SHA256, value interface{}) Href {
	schema := ExtractSchema(value)
	schemaKey, _ := c.saveObj(schema)
	result := Href{Hash:key, Type:schemaKey}
	return result
}

func (c *Container) CreateArray(objType interface{}, items ...cipher.SHA256) HArray {
	schema := ExtractSchema(objType)
	schemaKey, _ := c.saveObj(schema)
	result := HArray{Type: schemaKey, Items:items[:]}
	return result
}

func (c *Container) GetAllBySchema(schemaKey cipher.SHA256) []cipher.SHA256 {
	q := HrefQuery{}
	c.Root.ExpandBySchema(c, schemaKey, &q)
	return q.Items

	//return c.ds.Where(func(k cipher.SHA256, data []byte) bool {
	//	return false
	//})
}
