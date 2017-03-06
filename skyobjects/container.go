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

// Href is an objects data wrapped with it's Schema.
type Href struct {
	SchemaKey cipher.SHA256
	Data      []byte
}

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
		_, data, e := c.Get(key)
		if e != nil {
			return e
		}
		encoder.DeserializeRaw(data, &rt)
		if rt.TimeStamp > c.rootTS {
			c.rootKey, c.rootSeq, c.rootTS = key, rt.Sequence, rt.TimeStamp
		}
	}
	// Update Schemas.
	c.IndexSchemas()
	return nil
}

// Save saves an object into container.
func (c *Container) Save(schemaKey cipher.SHA256, data []byte) (key cipher.SHA256) {
	h := Href{SchemaKey: schemaKey, Data: data}
	return c.SaveHref(h)
}

// SaveObject also saves an object into container, but Serialises for you.
func (c *Container) SaveObject(schemaKey cipher.SHA256, obj interface{}) (key cipher.SHA256) {
	data := encoder.Serialize(obj)
	h := Href{SchemaKey: schemaKey, Data: data}
	return c.SaveHref(h)
}

// SaveHref saves object in form of 'Href'.
func (c *Container) SaveHref(h Href) (key cipher.SHA256) {
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

// SaveRoot saves a root object (if latest).
func (c *Container) SaveRoot(newRoot RootObject) bool {
	if newRoot.TimeStamp < c.rootTS {
		return false
	}
	c.SaveObject(c._rootType, newRoot)
	// c.rootTS = newRoot.TimeStamp
	// c.rootSeq = newRoot.Sequence
	// c.rootKey = c.SaveObject(c._rootType, newRoot)
	return true
}

// Get retrieves a stored object.
func (c *Container) Get(key cipher.SHA256) (schemaKey cipher.SHA256, data []byte, e error) {
	hrefData, ok := c.ds.Get(key)
	if ok == false {
		e = fmt.Errorf("no object found with key '%s'", key.Hex())
		return
	}
	var h Href
	encoder.DeserializeRaw(hrefData, &h) // Shouldn't create an error, everything stored in db is of type href.
	schemaKey, data = h.SchemaKey, h.Data
	return
}

// GetHref retrieves a stored object in the form of 'Href'.
func (c *Container) GetHref(key cipher.SHA256) (h Href, e error) {
	data, ok := c.ds.Get(key)
	if ok == false {
		e = fmt.Errorf("no object found with key '%s'", key.Hex())
		return
	}
	encoder.DeserializeRaw(data, &h)
	return
}

// GetRoot gets the latest local root.
func (c *Container) GetRoot() (root RootObject, e error) {
	if c.rootKey == (cipher.SHA256{}) {
		e = fmt.Errorf("no root avaliable")
		return
	}
	_, data, e := c.Get(c.rootKey)
	if e != nil {
		return
	}
	encoder.DeserializeRaw(data, &root)
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
func (c *Container) GetChildren(schemaKey cipher.SHA256, data []byte) (cMap map[cipher.SHA256]bool) {
	cMap = make(map[cipher.SHA256]bool)
	// Get schema of object.
	schema, e := c.GetSchemaOfKey(schemaKey)
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
			var keyArray HashArray
			encoder.DeserializeField(data, schema.Fields, field.Name, &keyArray)
			for _, k := range keyArray {
				cMap[k] = c.ds.Has(k)
			}
		case _StrHashObject, "sha256":
			var k cipher.SHA256
			encoder.DeserializeField(data, schema.Fields, field.Name, &k)
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
				var keyArray HashArray
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
