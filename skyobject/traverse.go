package skyobject

import (
	"fmt"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// SchemaByKey returns Schema by its reference
func (c *Container) SchemaByKey(sk cipher.SHA256) (s Schema, err error) {
	data, ok := c.db.Get(sk)
	if !ok {
		err = ErrMissingObject
		return
	}
	err = encoder.DeserializeRaw(data, &s)
	return
}

// Inspect prints the tree. Inspect doesn't returns "missing" errors
func (c *Container) Inspect() (err error) {
	if c.root == nil {
		err = ErrMissingRoot
		return
	}

	err = c.inspect(c.root.Schema, c.root.Root)
	return
}

func (c *Container) inspect(schk, objk cipher.SHA256) (err error) {
	var (
		schd, objd []byte
		ok         bool

		s Schema
	)

	if schd, ok = c.db.Get(schk); !ok { // don't have the schema
		fmt.Println("missing schema: ", schk.Hex())
		return
	}

	if objd, ok = c.db.Get(objk); !ok {
		fmt.Println("mising object: ", objk.Hex())
		return
	}

	// we have both schema and object
	if err = encoder.DeserializeRaw(schd, &s); err != nil {
		return
	}

	fmt.Println("---")
	fmt.Println("schema: ", s.String())
	fmt.Println("object: ", encoder.ParseFields(objd, s.Fields))
	fmt.Println("---")

	for _, sf := range s.Fields {
		var tag string
		if tag = skyobjectTag(sf); !strings.Contains(tag, "href") {
			continue
		}
		//
		// TODO: DRY
		//
		switch sf.Type {
		case hrefTypeName:
			// the field contains cipher.SHA256 reference
			var ref cipher.SHA256
			err = encoder.DeserializeField(objd, s.Fields, sf.Name, &ref)
			if err != nil {
				return
			}
			if schk, err = c.schemaByTag(tag); err != nil {
				return
			}
			if err = c.inspect(schk, ref); err != nil {
				return
			}
		case hrefArrayTypeName:
			// the field contains []cipher.SHA256 references
			var refs []cipher.SHA256
			err = encoder.DeserializeField(objd, s.Fields, sf.Name, &refs)
			if err != nil {
				return
			}
			if schk, err = c.schemaByTag(tag); err != nil {
				return
			}
			for _, ref := range refs {
				if err = c.inspect(schk, ref); err != nil {
					return
				}
			}
		case dynamicHrefTypeName:
			// the field is dynamic schema
			var dh DynamicHref
			err = encoder.DeserializeField(objd, s.Fields, sf.Name, &dh)
			if err != nil {
				return
			}
			if err = c.inspect(dh.Schema, dh.ObjKey); err != nil {
				return
			}
		default:
			err = ErrUnexpectedHrefTag
			return
		}
	}
	return
}

//
// TODO: fix encoder.ParseFields
//

func (c *Container) parseFields(data []byte,
	sf []encoder.StructField) (mss map[string]string) {

	return
}
