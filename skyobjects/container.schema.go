package skyobjects

import (
	"strings"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// ISchemaManager represents the schema manager for container.
type ISchemaManager interface {
	GetAll() []Schema
	GetOfType(typeName string) (cipher.SHA256, *Schema)
	GetOfKey(schemaKey cipher.SHA256) *Schema
	Register(types ...interface{})
}

type schemaManager struct {
	c       *Container
	ds      *data.DB
	schemas []schemaRef
}

func (m *schemaManager) GetAll() (schemas []Schema) {
	for _, schemaRef := range m.schemas {
		schemas = append(schemas, *m.GetOfKey(schemaRef.Key))
	}
	return
}

func (m *schemaManager) GetOfType(typeName string) (schemaKey cipher.SHA256, schema *Schema) {
	schema = &Schema{}
	query := func(key cipher.SHA256, data []byte) bool {
		if ref := m.c.dbGet(m.ds, key); ref.Type == _schemaType {
			tempSchema := Schema{}
			encoder.DeserializeRaw(ref.Data, &tempSchema)
			if strings.ToLower(tempSchema.Name) == strings.ToLower(typeName) {
				schema = &tempSchema
				schemaKey = key
				return true
			}
		}
		return false
	}
	if len(m.ds.Where(query)) != 1 {
		schema = &Schema{}
		schemaKey = cipher.SHA256{}
	}
	return
}

func (m *schemaManager) GetOfKey(schemaKey cipher.SHA256) (schema *Schema) {
	schema = &Schema{}
	ref := m.c.dbGet(m.ds, schemaKey)
	if ref.Type == _schemaType {
		encoder.DeserializeRaw(ref.Data, schema)
	}
	return
}

func (m *schemaManager) Register(objects ...interface{}) {
	for _, obj := range objects {
		schema := ReadSchema(obj)
		schemaData := encoder.Serialize(schema)
		schemaKey := m.c.dbSaveSchema(m.ds, schemaData)
		m.schemas = append(m.schemas, schemaRef{
			Name: schema.Name,
			Key:  schemaKey,
		})
	}
}

type schemaRef struct {
	Name string
	Key  cipher.SHA256
}
