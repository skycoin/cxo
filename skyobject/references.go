package skyobject

import (
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

//
// use special named types for references
//

// A Reference type represents reference to another object
type Reference cipher.SHA256

// Save an object to db and get reference-key to it
func (r *Root) Save(i interface{}) Reference {
	return Reference(r.cnt.db.AddAutoKey(encoder.Serialize(i)))
}

// A References type represents references to array of another objects
type References []Reference

// SaveArray of objects and get array of references-keys to them
func (r *Root) SaveArray(ary ...interface{}) (rs References) {
	if len(ary) == 0 {
		return
	}
	rs = make(References, 0, len(ary))
	for _, a := range ary {
		rs = append(rs, r.Save(a))
	}
	return
}

// A Dynamic represents dynamic reference to any object and reference to its
// schema
type Dynamic struct {
	Schema Reference
	Object Reference
}

// SaveSchema and get reference-key to it
func (r *Root) SaveSchema(i interface{}) (ref Reference) {
	typ := reflect.Indirect(reflect.ValueOf(i)).Type()
FromMap:
	if sk, ok := r.reg.reg[typeName(typ)]; ok {
		ref = Reference(sk)
		return
	}
	sv := r.reg.getSchema(i)
	if sv.IsNamed() { // getSchema registers named type automatically
		goto FromMap
	}
	ref = Reference(r.reg.db.AddAutoKey(sv.Encode())) // save manually
	return
}

func (r *Root) Dynamic(i interface{}) (dn Dynamic) {
	dn.Object = r.Save(i)
	dn.Schema = r.SaveSchema(i)
	return
}
