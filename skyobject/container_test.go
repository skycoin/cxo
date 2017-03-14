package skyobject

import (
	"bytes"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

func TestNewContainer(t *testing.T) {
	t.Run("nil-db", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("missing panic")
			}
		}()
		NewContainer(nil)
	})
	t.Run("pass", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		if c == nil {
			t.Error("misisng container")
			return
		}
		if c.db == nil {
			t.Error("missing db")
		}
		if c.root != nil {
			t.Error("unexpecte root")
		}
	})
}

// including SetRoot and SetEncodedRoot
func TestContainer_Root(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		if c.Root() != nil {
			t.Error("unexpected root in new container")
		}
	})
	t.Run("full", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		c.root = c.NewRoot()
		if c.Root() == nil {
			t.Error("misisng root")
		}
	})
}

func TestContainer_SetRoot(t *testing.T) {
	t.Run("set n get", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		root := c.NewRoot()
		if !c.SetRoot(root) {
			t.Error("can't set root to empty container")
		}
		if c.Root() != root {
			t.Error("wrong root object after SetRoot")
		}
	})
	t.Run("older", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		r1, r2 := c.NewRoot(), c.NewRoot()
		c.SetRoot(r2)
		if c.SetRoot(r1) {
			t.Error("set older")
		}
	})
	t.Run("newer", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		r1, r2 := c.NewRoot(), c.NewRoot()
		c.SetRoot(r1)
		if !c.SetRoot(r2) {
			t.Error("can't set newer")
		}
	})
}

func TestContainer_SetEncodedRoot(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		if ok, err := c.SetEncodedRoot(nil); err == nil {
			t.Error("missing error")
		} else if ok {
			t.Error("set nil root")
		}
	})
	t.Run("valid", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		root := c.NewRoot()
		if ok, err := c.SetEncodedRoot(root.Encode()); err != nil {
			t.Error("unexpected error")
		} else if !ok {
			t.Error("can't set encoded root")
		}
	})
	t.Run("malformed", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		any := []byte("it's not an encoded root")
		if ok, err := c.SetEncodedRoot(any); err == nil {
			t.Error("missing error")
		} else if ok {
			t.Error("set malformed root")
		}
	})
}

func TestContainer_Save(t *testing.T) {
	c := NewContainer(data.NewDB())
	usr := User{"Alex", 23, "usr"}
	hash := c.Save(usr)
	if data, ok := c.db.Get(hash); !ok {
		t.Error("misisng value in db")
	} else if bytes.Compare(data, encoder.Serialize(usr)) != 0 {
		t.Error("wrong value in db")
	}
}

func TestContainer_SaveArray(t *testing.T) {
	c := NewContainer(data.NewDB())
	users := []interface{}{
		User{"Alice", 21, ""},
		User{"Eva", 22, ""},
		User{"John", 23, ""},
		User{"Michel", 24, ""},
	}
	for i, ref := range c.SaveArray(users...) {
		d, ok := c.db.Get(ref)
		if !ok {
			t.Error("missisng member of array in db")
			continue
		}
		if bytes.Compare(d, encoder.Serialize(users[i])) != 0 {
			t.Error("wrong value in db")
		}
	}
}

func TestContainer_Want(t *testing.T) {
	c := NewContainer(data.NewDB())
	list, err := c.Want()
	if err != nil {
		t.Error("unexpected error")
	}
	if len(list) != 0 {
		t.Error("non-empty list for empty container")
	}

	// type User struct {
	// 	Name   string
	// 	Age    int64
	// 	Hidden string `enc:"-"` // ignore the field
	// }

	// type Man struct {
	// 	Name   string
	// 	Height int64
	// 	Weight int64
	// }

	// type SamllGroup struct {
	// 	Name     string
	// 	Leader   cipher.SHA256   `skyobject:"href"` // single User
	// 	Outsider cipher.SHA256   // not a reference
	// 	FallGuy  Dynamic         `skyobject:"href"` // dynamic href
	// 	Members  []cipher.SHA256 `skyobject:"href"` // array of Users
	// }

	leader := User{"Billy Kid", 16, ""}
	man := Dynamic{
		Schema: getHash(getSchema(Man{})),
		ObjKey: getHash(Man{"Bob Simple", 182, 82}),
	}
	users := []interface{}{
		User{"Alice", 21, ""},
		User{"Eva", 22, ""},
		User{"John", 23, ""},
		User{"Michel", 24, ""},
	}

	group := SamllGroup{
		Name:     "the group",
		Leader:   getHash(leader),
		Outsider: cipher.SHA256{9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
		FallGuy:  man,
		Members: []cipher.SHA256{
			getHash(users[0]),
			getHash(users[1]),
			getHash(users[2]),
			getHash(users[3]),
		},
	}

	root := c.NewRoot()
	root.Set(group)
	c.SetRoot(root)

	data, ok := c.db.Get(getHash(group))
	if !ok {
		t.Fatal("missing")
	}
	var gr SamllGroup
	if err = encoder.DeserializeRaw(data, &gr); err != nil {
		t.Fatal(err)
	}

	// --------------------
	// scheme of user    +1
	// leader            -1 w
	// schema of man     -1 w
	// man               -1 w
	// members           -4 w
	// --------------------
	//                   -7
	list, err = c.Want()
	if err != nil {
		t.Error(err)
		return
	}
	if len(list) != 7 {
		t.Error("wrong len: want 7, got ", len(list))
	}

}
