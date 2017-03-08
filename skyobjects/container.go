package skyobjects

import (
	"bytes"
	"fmt"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

const _StrHashArray = "hasharray"
const _StrHashObject = "hashobject"

// Container contains skyobjects.
type Container struct {
	ds      *data.DB                 // Data source.
	rootKey cipher.SHA256            // Key of latest root object.
	rootSeq uint64                   // Sequence number of latest root object.
	rootTS  int64                    // Timestamp of latest root object.
	schemas map[string]cipher.SHA256 // List of avaliable schemas.

	// Keys used for storing/retrieving specific object types.
	// NO NOT MODIFY.
	_rootType   cipher.SHA256
	_schemaType cipher.SHA256
}

// NewContainer creates a new skyobjects container.
func NewContainer(ds *data.DB) (c *Container) {
	c = &Container{
		ds:      ds,
		schemas: make(map[string]cipher.SHA256),
	}
	c._rootType = cipher.SumSHA256(encoder.Serialize(RootObject{}))
	c._schemaType = cipher.SumSHA256(encoder.Serialize(Schema{}))
	return
}

// SetDB sets a new DB for container.
// Member values of the container will be changed appropriately.
func (c *Container) SetDB(ds *data.DB) error {
	// TODO: Implement and complete!!!!!!!!!!
	c.ds = ds
	// Clear members.
	c.rootKey, c.rootSeq, c.rootTS = cipher.SHA256{}, 0, 0
	// Find latest root.
	for _, key := range c.GetAllOfSchema(c._rootType) {
		var rt RootObject
		h, e := c.Get(key)
		if e != nil {
			return e
		}
		encoder.DeserializeRaw(h.Data, &rt)
		if rt.TimeStamp > c.rootTS {
			c.rootKey, c.rootSeq, c.rootTS = key, rt.Sequence, rt.TimeStamp
		}
	}
	// Update Schemas.
	c.IndexSchemas()
	return nil
}

// Save saves object (form of 'Href') into container.
func (c *Container) Save(h Href) (key cipher.SHA256) {
	key = c.ds.AddAutoKey(encoder.Serialize(h))
	// SPECIAL CASES:
	switch h.SchemaKey {
	case c._schemaType: // Special case for schemas.
		c.IndexSchemas()
	case c._rootType: // Special case for root.
		var root RootObject
		encoder.DeserializeRaw(h.Data, &root)
		// Only add to main root if latest root.
		if root.TimeStamp > c.rootTS {
			c.rootKey = key
			c.rootSeq = root.Sequence
			c.rootTS = root.TimeStamp
		}
	}
	return
}

// SaveData saves an object into container.
func (c *Container) SaveData(schemaKey cipher.SHA256, data []byte) (key cipher.SHA256) {
	h := Href{SchemaKey: schemaKey, Data: data}
	return c.Save(h)
}

// SaveObject also saves an object into container, but Serialises for you.
func (c *Container) SaveObject(schemaKey cipher.SHA256, obj interface{}) (key cipher.SHA256) {
	data := encoder.Serialize(obj)
	h := Href{SchemaKey: schemaKey, Data: data}
	return c.Save(h)
}

// Get retrieves a stored object in the form of 'Href'.
func (c *Container) Get(key cipher.SHA256) (h Href, e error) {
	data, ok := c.ds.Get(key)
	if ok == false {
		e = ErrorSkyObjectNotFound{Key: key}
		return
	}
	encoder.DeserializeRaw(data, &h)
	return
}

// GetAllOfSchema gets all keys of objects with specified schemaKey.
func (c *Container) GetAllOfSchema(schemaKey cipher.SHA256) []cipher.SHA256 {
	query := func(key cipher.SHA256, data []byte) bool {
		return bytes.Compare(schemaKey[:32], data[:32]) == 0
	}
	return c.ds.Where(query)
}

// GetChildren gets all keys of the direct children of an object.
// Boolean value in map:
// * TRUE: We have a copy of this object in container.
// * FALSE: We don't have a copy of this object in container.
func (c *Container) GetChildren(h Href) (cMap map[cipher.SHA256]bool) {
	cMap = make(map[cipher.SHA256]bool)
	// Get schema of object.
	schema, e := c.GetSchemaOfKey(h.SchemaKey)
	if e != nil {
		fmt.Println(e)
		return
	}
	// Iterate through fields of object.
	for _, field := range schema.Fields {
		// Continue if not referenced.
		if st, _ := getSkyTag(c, field.Tag); st.href == false {
			continue
		}
		switch field.Type {
		case _StrHashArray, "":
			var keyArray []cipher.SHA256
			encoder.DeserializeField(h.Data, schema.Fields, field.Name, &keyArray)
			for _, k := range keyArray {
				cMap[k] = c.ds.Has(k)
			}
		case _StrHashObject, "sha256":
			var k cipher.SHA256
			encoder.DeserializeField(h.Data, schema.Fields, field.Name, &k)
			cMap[k] = c.ds.Has(k)
		}
	}
	return
}

// GetReferencesFor gets a list of objects that reference the specified object.
func (c *Container) GetReferencesFor(objKey cipher.SHA256) []cipher.SHA256 {
	query := func(key cipher.SHA256, data []byte) bool {
		var h Href
		encoder.DeserializeRaw(data, &h)
		schema, e := c.GetSchemaOfKey(h.SchemaKey)
		if e != nil {
			fmt.Println(e)
			return false
		}
		for _, field := range schema.Fields {
			// Continue if not referenced.
			if st, _ := getSkyTag(c, field.Tag); st.href == false {
				continue
			}
			// fmt.Println("[FIELD TYPE]", field.Type)
			switch field.Type {
			case _StrHashArray, "":
				var keyArray []cipher.SHA256
				encoder.DeserializeField(h.Data, schema.Fields, field.Name, &keyArray)
				for _, k := range keyArray {
					if k == objKey {
						return true
					}
				}
			case _StrHashObject, "sha256":
				var k cipher.SHA256
				encoder.DeserializeField(h.Data, schema.Fields, field.Name, &k)
				if k == objKey {
					return true
				}
			}
		}
		return false
	}
	return c.ds.Where(query)
}
