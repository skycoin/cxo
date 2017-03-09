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
// stringify fields of an object. Or error. The error can be ErrMissingRoot,
// or MissingError (with key) when required object doesn't exists
// in database. If an InspectFunc returns an error then call of Inspect
// terminates and returns the error. There is special error called
// ErrStopInspection that is used to stop the call of Inspect without
// errors
type InspectFunc func(s *Schema, fields map[string]string, err error) error

// Inspect prints the tree. Inspect doesn't returns "missing" errors
func (c *Container) Inspect(insp InspectFunc) (err error) {
	if insp == nil {
		err = ErrInvalidArgument
		return
	}
	if c.root == nil {
		err = ErrMissingRoot
		return
	}

	err = c.inspect(c.root.Schema, c.root.Root, insp)
	return
}

func (c *Container) inspect(schk, objk cipher.SHA256,
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
		if err = insp(nil, nil, &MissingError{schk}); err != nil {
			return
		}
		return
	}

	if objd, ok = c.db.Get(objk); !ok {
		if err = insp(nil, nil, &MissingError{objk}); err != nil {
			return
		}
		return
	}

	// we have both schema and object
	if err = encoder.DeserializeRaw(schd, &s); err != nil {
		return
	}

	// ---
	if err = insp(&s, encoder.ParseFields(objd, s.Fields), nil); err != nil {
		return
	}
	// ---

	// follow references
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
			if strings.Contains(tag, "schema=") {
				if schk, err = c.schemaByTag(tag); err != nil {
					return
				}
				if err = c.inspect(schk, ref, insp); err != nil {
					return
				}
			} else {
				var dhd []byte
				if dhd, ok = c.db.Get(ref); !ok {
					fmt.Println("missing dynamic reference value: ", ref.Hex())
					continue
				}
				var dh DynamicHref
				if err = encoder.DeserializeRaw(dhd, &dh); err != nil {
					return
				}
				if err = c.inspect(dh.Schema, dh.ObjKey, insp); err != nil {
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
				if err = c.inspect(schk, ref, insp); err != nil {
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
