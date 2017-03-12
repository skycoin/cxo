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

// Missing error is used for InspectFunc
type MissingError struct {
	Key cipher.SHA256
}

func (m *MissingError) Error() string {
	return fmt.Sprint("misisng object: ", m.Key.Hex())
}

// An InspectFuunc is used to inspect objects tree. It receives schema and
// fields of an object, and distance from root called deep.
// Or error. The error can be ErrMissingRoot,
// or MissingError (with key) when required object doesn't exists
// in database. If an InspectFunc returns an error then call of Inspect
// terminates and returns the error. There is special error called
// ErrStopInspection that is used to stop the call of Inspect without
// errors
type InspectFunc func(s *Schema, fields map[string]interface{},
	deep int, err error) error

// Inspect prints the tree. Inspect doesn't returns "missing" errors
//
//     Note: unfortunately Inspect doesn't show fields of data
//     because of encoder bug
//
func (c *Container) Inspect(insp InspectFunc) (err error) {
	if insp == nil {
		err = ErrInvalidArgument
		return
	}
	if c.root == nil {
		err = ErrMissingRoot
		return
	}

	err = c.inspect(c.root.Schema, c.root.Root, 0, insp)
	return
}

func (c *Container) inspect(schk, objk cipher.SHA256, deep int,
	insp InspectFunc) (err error) {

	var (
		schd, objd []byte
		ok         bool

		s Schema
	)

	defer func() {
		if err == ErrStopInspection { // clean up service error
			err = nil
		}
	}()

	if schd, ok = c.db.Get(schk); !ok { // don't have the schema
		if err = insp(nil, nil, deep, &MissingError{schk}); err != nil {
			return
		}
		return
	}

	if objd, ok = c.db.Get(objk); !ok {
		if err = insp(nil, nil, deep, &MissingError{objk}); err != nil {
			return
		}
		return
	}

	// we have both schema and object
	if err = encoder.DeserializeRaw(schd, &s); err != nil {
		return
	}

	// --- TODO: encoder.ParseFields(objd, s.Fields) instead of hidden:hidden
	if err = insp(&s, map[string]interface{}{"hidden": "hidden"}, deep, nil); err != nil {
		return
	}
	// ---

	// follow references
	for _, sf := range s.Fields {
		var tag string
		if tag = skyobjectTag(sf); !strings.Contains(tag, "href") {
			continue
		}
		switch sf.Type {
		case hrefTypeName:
			var ref cipher.SHA256
			err = encoder.DeserializeField(objd, s.Fields, sf.Name, &ref)
			if err != nil {
				return
			}
			if strings.Contains(tag, "schema=") {
				// the field contains cipher.SHA256 reference
				if schk, err = c.schemaByTag(tag); err != nil {
					return
				}
				if err = c.inspect(schk, ref, deep+1, insp); err != nil {
					return
				}
			} else {
				// the field contains cipher.SHA256 reference to dynamic href
				var data []byte
				var ok bool
				if data, ok = c.db.Get(ref); !ok { // we don't have dh-object
					err = insp(nil, nil, deep+1, &MissingError{ref})
					if err != nil {
						return
					}
					continue
				}
				var dh DynamicHref
				if err = encoder.DeserializeRaw(data, &dh); err != nil {
					return
				}
				err = c.inspect(dh.Schema, dh.ObjKey, deep+1, insp)
				if err != nil {
					return
				}
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
				if err = c.inspect(schk, ref, deep+1, insp); err != nil {
					return
				}
			}
		default:
			err = ErrUnexpectedHrefTag
			return
		}
	}
	return
}
