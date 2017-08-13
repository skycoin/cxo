package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

func TestPack_Save(t *testing.T) {

	pk, sk := cipher.GenerateKeyPair()

	t.Run("can't save", func(t *testing.T) {
		c := getCont()
		// no such feed
		pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pack.Save(); err == nil {
			t.Error("missing no-such-feed error")
		}
		if err := c.AddFeed(pk); err != nil {
			t.Fatal(err)
		}
		// empty sec key
		pack, err = c.Unpack(pack.Root(), 0, c.CoreRegistry().Types(),
			cipher.SecKey{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pack.Save(); err == nil {
			t.Error("save with secret key")
		}
		// view only
		pack, err = c.Unpack(pack.Root(), ViewOnly, c.CoreRegistry().Types(),
			sk)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pack.Save(); err == nil {
			t.Error("save with ViewOnly")
		}
	})

	// TODO (kostyarin): Commit error

	t.Run("save", func(t *testing.T) {
		c := getCont()
		if err := c.AddFeed(pk); err != nil {
			t.Fatal(err)
		}
		pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
		if err != nil {
			t.Fatal(err)
		}

		pack.Append(Group{
			Name: "VDPG",
			Leader: pack.Ref(User{
				Name: "Alice",
				Age:  23,
			}),
			Members: pack.Refs(
				User{"Eva", 19, nil},
				User{"Ammy", 17, nil},
				User{"Kate", 21, nil},
			),
			Curator: pack.Dynamic(Developer{
				Name:   "Konstantin",
				GitHub: "logrusorgru",
			}),
		})
		if _, err = pack.Save(); err != nil {
			t.Fatal(err)
		}
		// check out DB
		c.DB().View(func(tx data.Tv) (_ error) {
			// root object
			roots := tx.Feeds().Roots(pk)
			if roots == nil {
				t.Fatal("missing roots")
			}
			rp := roots.Last()
			if rp == nil {
				t.Fatal("misisng root")
			}
			if rp.Seq != pack.Root().Seq {
				t.Fatal("wrong root")
			}
			rt := pack.Root()
			if len(rt.Refs) != 1 {
				t.Fatal("wrong Root.Refs count")
			}
			groupDr := rt.Refs[0]
			// objects
			objs := tx.Objects()
			if groupDr.Object == (cipher.SHA256{}) {
				t.Fatal("wrong Dynaimc")
			}
			if objs.Get(groupDr.Object) == nil {
				t.Fatal("unsaved object")
			}
			// the group
			groupInterface, err := pack.RefByIndex(0)
			if err != nil {
				t.Fatal(err)
			}
			if groupInterface == nil {
				t.Fatal("group is nil")
			}
			group, ok := groupInterface.(*Group)
			if !ok {
				t.Fatal("wrong type")
			}
			if group.Name != "VDPG" {
				t.Fatal("wrong value after saving")
			}
			// leader
			if lead := group.Leader.Hash; lead == (cipher.SHA256{}) {
				t.Fatal("empty ref")
			} else if objs.Get(lead) == nil {
				t.Error("unsaved object")
			}
			leadInterface, err := group.Leader.Value()
			if err != nil {
				t.Fatal(err)
			}
			if leadInterface == nil {
				t.Fatal("nil interface")
			}
			lead, ok := leadInterface.(*User)
			if !ok {
				t.Fatal("wrong type")
			}
			if lead.Name != "Alice" || lead.Age != 23 {
				t.Fatal("wron value after saving", lead)
			}
			// members
			if mms := group.Members.Hash; mms == (cipher.SHA256{}) {
				t.Fatal("empty refs")
			} else if objs.Get(mms) == nil {
				t.Fatal("unsaved refs")
			}
			// explore the refs
			mmsVal := objs.Get(group.Members.Hash)
			var er encodedRefs
			if err := encoder.DeserializeRaw(mmsVal, &er); err != nil {
				t.Fatal(err)
			}
			if len(er.Nested) == 0 {
				t.Fatal("no netsed nodes")
			}
			if er.Depth != 0 {
				t.Error("wrong depth")
			}
			for _, hash := range er.Nested {
				if objs.Get(hash) == nil {
					t.Fatal("missing member")
				}
			}
			// curator
			cur := group.Curator
			if cur.Object == (cipher.SHA256{}) {
				t.Fatal("unsaved dynamic")
			}
			if cur.SchemaRef == (SchemaRef{}) {
				t.Fatal("empty schema ref")
			}
			if objs.Get(cur.Object) == nil {
				t.Fatal("unssaved object")
			}
			curInterface, err := cur.Value()
			if err != nil {
				t.Fatal(err)
			}
			if curInterface == nil {
				t.Fatal("nil interface")
			}
			curg, ok := curInterface.(*Developer)
			if !ok {
				t.Fatal("wrong type")
			}
			if curg.Name != "Konstantin" || curg.GitHub != "logrusorgru" {
				t.Fatal("wrong value")
			}
			return
		})
	})

}

/*

func (p *Pack) init() (err error) {
	if p.flags&ViewOnly == 0 {
		p.updateStack.init()
	}
	// Do we need to unpack entire tree?
	if p.flags&EntireTree != 0 {
		// unpack all possible
		_, err = p.RootRefs()
	}
	return
}

func (p *Pack) RootRefs() (objs []interface{}, err error) {
	if len(p.r.Refs) == 0 {
		return // (empty) nil, nil
	}
	objs = make([]interface{}, 0, len(p.r.Refs))
	for i := range p.r.Refs {
		var obj interface{}
		if obj, err = p.RefByIndex(i); err != nil {
			return
		}
		objs = append(objs, obj)
	}
	return
}

func validateIndex(i, ln int) (err error) {
	if i < 0 {
		err = fmt.Errorf("negative index %d", i)
	} else if i >= ln {
		err = fmt.Errorf("index out of range %d (len %d)", i, ln)
	}
	return
}

func (p *Pack) validateRootRefsIndex(i int) error {
	return validateIndex(i, len(p.r.Refs))
}

func (p *Pack) RefByIndex(i int) (obj interface{}, err error) {
	if err = p.validateRootRefsIndex(i); err != nil {
		return
	}

	if p.r.Refs[i].wn == nil {
		p.r.Refs[i].wn = &walkNode{pack: p}
	}
	obj, err = p.r.Refs[i].Value()
	return
}

func (p *Pack) SetRefByIndex(i int, obj interface{}) (err error) {
	if err = p.validateRootRefsIndex(i); err != nil {
		return
	}
	p.r.Refs[i] = p.Dynamic(obj)
	return
}

func (p *Pack) Append(objs ...interface{}) {
	for _, obj := range objs {
		p.r.Refs = append(p.r.Refs, p.Dynamic(obj))
	}
	return
}

func (p *Pack) Clear() {
	p.r.Refs = nil
}

// perform
//
// - get schema of the obj
// - if type of the object is not pointer, then the method makes it pointer
// - initialize (setupTo) the obj
//
// reply
//
// sch - schema of nil if obj is nil (or err is not)
// ptr - (1) pointer to value of the obj, (2) the obj if the obj is pointer
//       to non-nil, (3) nil if obj is nil (4) nil if obj represents
//       nil-pointer of some type (for example: var usr *User, the usr
//       is nil pointer to User), (5) nil if err is not
// err - first error
func (p *Pack) initialize(obj interface{}) (sch Schema, ptr interface{},
	err error) {

	if obj == nil {
		return // nil, nil, nil
	}

	// val -> non-pointer value for setupToGo method
	// typ -> non-pointer type for schemaOf method
	// ptr -> pointer

	val := reflect.ValueOf(obj)
	typ := val.Type()

	// we need the obj to be pointer
	if typ.Kind() != reflect.Ptr {
		valp := reflect.New(typ)
		valp.Elem().Set(val)
		ptr = valp.Interface()
	} else {
		typ = typ.Elem() // we need non-pointer type for schemaOf method
		if !val.IsNil() {
			val = val.Elem() // non-pointer value for setupToGo method
			ptr = obj        // it is already a pointer
		}
	}

	if sch, err = p.schemaOf(typ); err != nil {
		return
	}

	if ptr != nil {
		err = p.setupToGo(val)
	}
	return
}

// get Schema by given reflect.Type, the type must not be a pointer
func (p *Pack) schemaOf(typ reflect.Type) (sch Schema, err error) {
	if name, ok := p.types.Inverse[typ]; !ok {
		// detailed error
		err = fmt.Errorf("can't get Schema of %s:"+
			" given object not found in Types.Inverse map",
			typ.String())
	} else if sch, err = p.reg.SchemaByName(name); err != nil {
		// dtailed error
		err = fmt.Errorf(`wrong Types of Pack:
    schema name found in Types, but schema by the name not found in Registry
    error:                      %s
    registry reference:         %s
    schema name:                %s
    reflect.Type of the obejct: %s`,
			err,
			p.reg.Reference().Short(),
			name,
			typ.String())
	}
	return
}

func (p *Pack) Dynamic(obj interface{}) (dr Dynamic) {
	dr = p.dynamic()
	if err := dr.SetValue(obj); err != nil {
		panic(err)
	}
	return
}

func (p *Pack) Refs(objs ...interface{}) (r Refs) {
	r = p.refs()
	if err := r.Append(objs...); err != nil {
		panic(err)
	}
	return
}


*/
