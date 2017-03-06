package skyobjects

import (
	"fmt"

	"github.com/evanlinjin/cxo/encoder"
	"github.com/skycoin/skycoin/src/cipher"
)

// SaveSchema saves a schema to container.
func (c *Container) SaveSchema(object interface{}) (schemaKey cipher.SHA256) {
	schema := ReadSchema(object)
	schemaData := encoder.Serialize(schema)
	h := Href{SchemaKey: c._schemaType, Data: schemaData}
	schemaKey = c.ds.AddAutoKey(encoder.Serialize(h))

	// Append data to c.schemas
	c.schemas[schemaKey] = schema.Name
	return
}

// GetAllSchemas returns a list of all schemas in container.
func (c *Container) GetAllSchemas() (schemas []*Schema) {
	for k := range c.schemas {
		schema, _ := c.GetSchemaOfKey(k)
		schemas = append(schemas, schema)
	}
	return
}

// GetSchemaOfKey gets the schema from schemaKey.
func (c *Container) GetSchemaOfKey(schemaKey cipher.SHA256) (schema *Schema, e error) {
	dbSchemaKey, data, e := c.Get(schemaKey)
	if e != nil {
		return
	}
	if dbSchemaKey != c._schemaType {
		e = fmt.Errorf("is not Schema type")
		return
	}
	schema = &Schema{}
	e = encoder.DeserializeRaw(data, schema)
	return
}

// IndexSchemas indexes all schemas direcly to Container for easy access.
func (c *Container) IndexSchemas() {
	// Clear pre-existing schema map.
	c.schemas = make(map[cipher.SHA256]string)
	query := func(key cipher.SHA256, data []byte) bool {
		var h Href
		encoder.DeserializeRaw(data, &h)
		if h.SchemaKey != c._schemaType {
			return false
		}
		// Continue only if key is of a schema.
		var schema Schema
		encoder.DeserializeRaw(h.Data, &schema)
		c.schemas[key] = schema.Name
		return false
	}
	c.ds.Where(query)
}
