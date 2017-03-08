package skyobjects

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// SaveSchema saves a schema to container.
func (c *Container) SaveSchema(object interface{}) (schemaKey cipher.SHA256) {
	schema := ReadSchema(object)
	schemaData := encoder.Serialize(schema)
	h := Href{SchemaKey: c._schemaType, Data: schemaData}
	schemaKey = c.ds.AddAutoKey(encoder.Serialize(h))

	// Append data to c.schemas
	c.schemas[schema.Name] = schemaKey
	return
}

// GetAllSchemas returns a list of all schemas in container.
func (c *Container) GetAllSchemas() (schemas []*Schema) {
	for _, k := range c.schemas {
		schema, _ := c.GetSchemaOfKey(k)
		schemas = append(schemas, schema)
	}
	return
}

// GetSchemaOfKey gets the schema from schemaKey.
func (c *Container) GetSchemaOfKey(schemaKey cipher.SHA256) (schema *Schema, e error) {
	h, e := c.Get(schemaKey)
	if e != nil {
		return
	}
	if h.SchemaKey != c._schemaType {
		e = ErrorKeyIsNotSchema{SchemaKey: schemaKey}
		return
	}
	schema = &Schema{}
	e = encoder.DeserializeRaw(h.Data, schema)
	return
}

// GetSchemaOfName finds schema of specified name.
func (c *Container) GetSchemaOfName(schemaName string) (schema *Schema, e error) {
	schemaKey, exists := c.schemas[schemaName]
	if exists == false {
		e = ErrorSchemaNotFound{SchemaName: schemaName}
		return
	}
	// Obtain schema from db.
	h, err := c.Get(schemaKey)
	if e != nil {
		return nil, err
	}
	schema = &Schema{}
	encoder.DeserializeRaw(h.Data, schema)
	return
}

// IndexSchemas indexes all schemas direcly to Container for easy access.
func (c *Container) IndexSchemas() {
	// Clear pre-existing schema map.
	c.schemas = make(map[string]cipher.SHA256)
	// Prepare query.
	query := func(key cipher.SHA256, data []byte) bool {
		var h Href
		encoder.DeserializeRaw(data, &h)
		if h.SchemaKey != c._schemaType {
			return false
		}
		// Continue only if key is of a schema.
		var schema Schema
		encoder.DeserializeRaw(h.Data, &schema)
		c.schemas[schema.Name] = key
		return false
	}
	c.ds.Where(query)
}
