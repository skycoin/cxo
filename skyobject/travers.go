package skyobject

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

type MissingError struct {
	hash cipher.SHA256
}

func (m *MissingError) Error() string {
	return "missing object: " + m.hash.Hex()
}

type InspectFunc func(s *Schema, objd []byte, deep int, err error) error

func (c *Container) Inspect(insp InspectFunc) (err error) {
	if c.root == nil {
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

	if schd, ok = c.db.Get(schk); !ok { // don't have the schema
		want[schk] = struct{}{}
		c.addMissing(want, objk)
		return
	}

	if objd, ok = c.db.Get(objk); !ok {
		want[objk] = struct{}{}
		return
	}

	// we have both schema and object
	if err = encoder.DeserializeRaw(schd, &s); err != nil {
		return
	}

	for i := 0; i < len(s.Fields); i++ {
		var (
			sf  encoder.StructField = s.Fields[i]
			tag string              = skyobjectTag(sf)
		)
		if tag == "" {
			continue
		}
		if sf.Type == hrefTypeName {
			var ref cipher.SHA256
			if ref, err = getRefField(objd, s.Fields, sf.Name); err != nil {
				goto Error
			}
			if strings.Contains(tag, "schema=") {
				// the field contains cipher.SHA256 reference
				err = c.mergeRefWants(want, tag, ref)
				if err != nil {
					goto Error
				}
			} else if strings.Contains(tag, "dynamic_schema") {
				i++
				if i >= len(s.Fields) {
					err = ErrMissingObjKeyField
					goto Error
				}
				err = c.mergeDynamicWants(objd, s.Fields, s.Fields[i], want, ref)
				if err != nil {
					goto Error
				}
			} // else -> not a reference
		} else if sf.Type == hrefArrayTypeName {
			// the field contains reference to []cipher.SHA256 references
			err = c.mergeArrayWants(objd, s.Fields, sf.Name, want, tag)
			if err != nil {
				goto Error
			}
		}
	}
	return
Error:
	want = nil // set want to nil if we have got an error
	return
}

func (c *Container) mergeDynamicWants(objd []byte,
	fs []encoder.StructField,
	sf encoder.StructField,
	want map[cipher.SHA256]struct{},
	ref cipher.SHA256) (err error) {

	var tag string = skyobjectTag(sf)
	if !strings.Contains(tag, "dynamic_objkey") ||
		sf.Type != hrefTypeName {
		err = ErrMissingObjKeyField
		return
	}
	var objRef cipher.SHA256
	objRef, err = getRefField(objd, fs, sf.Name)
	if err != nil {
		return
	}
	if err = c.want(ref, objRef, want); err != nil {
		return
	}
	return
}

func (c *Container) mergeRefWants(want map[cipher.SHA256]struct{},
	tag string,
	ref cipher.SHA256) (err error) {

	var schk cipher.SHA256

	if schk, err = c.schemaByTag(tag, "schema="); err != nil {
		return
	}
	if err = c.want(schk, ref, want); err != nil {
		return
	}
	return
}

func (c *Container) mergeArrayWants(objd []byte,
	fs []encoder.StructField,
	name string,
	want map[cipher.SHA256]struct{},
	tag string) (err error) {

	var (
		refs []cipher.SHA256
		schk cipher.SHA256
	)
	if refs, err = getRefsField(objd, fs, name); err != nil {
		return
	}
	if schk, err = c.schemaByTag(tag, "array="); err != nil {
		return
	}
	for _, ref := range refs {
		if err = c.want(schk, ref, want); err != nil {
			return
		}
	}
	return
}

func getRefField(data []byte, fs []encoder.StructField,
	fname string) (ref cipher.SHA256, err error) {
	err = encoder.DeserializeField(data, fs, fname, &ref)
	return
}

func getRefsField(data []byte, fs []encoder.StructField,
	fname string) (refs []cipher.SHA256, err error) {
	err = encoder.DeserializeField(data, fs, fname, &refs)
	return
}
