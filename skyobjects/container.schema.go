package skyobjects

import (
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

type schemaRef struct {
	Name string
	Key  cipher.SHA256
}

// GetAllSchemas returns a list of all schemas in container.
func (c *Container) GetAllSchemas(schemas []Schema) {
	for _, ref := range c.schemas {
		schema := c.GetSchemaOfKey(ref.Key)
		schemas = append(schemas, *schema)
	}
	return
}

// GetSchemaOfKey returns the schema of specified key.
func (c *Container) GetSchemaOfKey(schemaKey cipher.SHA256) (schema *Schema) {
	schema = &Schema{}
	ref := c.dbGet(c.ds, schemaKey)
	if ref.Type == _schemaType {
		encoder.DeserializeRaw(ref.Data, schema)
	}
	return
}

// GetSchemaOfTypeName gets schema of specified type name.
func (c *Container) GetSchemaOfTypeName(typeName string) (schemaKey cipher.SHA256, schema *Schema) {
	schema = &Schema{}
	query := func(key cipher.SHA256, data []byte) bool {
		if ref := c.dbGet(c.ds, key); ref.Type == _schemaType {
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
	if len(c.ds.Where(query)) != 1 {
		schema = &Schema{}
		schemaKey = cipher.SHA256{}
	}
	return
}

// RegisterSchema registers a schema.
func (c *Container) RegisterSchema(objects ...interface{}) {
	for _, obj := range objects {
		schema := ReadSchema(obj)
		schemaData := encoder.Serialize(schema)
		schemaKey := c.dbSaveSchema(c.ds, schemaData)
		c.schemas = append(c.schemas, schemaRef{
			Name: schema.Name,
			Key:  schemaKey,
		})
	}
}
