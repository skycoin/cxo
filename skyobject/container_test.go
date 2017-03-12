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
	ary := c.SaveArray(users...)
	if data, ok := c.db.Get(ary); !ok {
		t.Error("missing value in db")
	} else {
		var refs []cipher.SHA256
		if err := encoder.DeserializeRaw(data, &refs); err != nil {
			t.Error("unexpected error: ", err)
			return
		}
		if len(refs) != len(users) {
			t.Error("missmatch length of references and length of array")
			return
		}
		for i, ref := range refs {
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
	root := c.NewRoot()
	root.Register("User", User{})
	// fields
	leader := User{"Billy Kid", 16, ""}
	leaderHash := getHash(leader)
	fg := Man{"Bob", 182, 82}
	fgHash := getHash(fg)
	fgSchemaHash := getHash(getSchema(fg))
	users := []interface{}{
		User{"Alice", 21, ""},  // 0
		User{"Eva", 22, ""},    // 1
		User{"John", 23, ""},   // 2
		User{"Michel", 24, ""}, // 3
	}
	usersHashes := []cipher.SHA256{}
	for _, u := range users {
		usersHashes = append(usersHashes, getHash(u))
	}
	group := SamllGroup{
		Name:     "Average small group",
		Leader:   leaderHash,
		Outsider: cipher.SHA256{0, 1, 2, 3},
		FallGuy: Dynamic{
			Schema: fgSchemaHash,
			ObjKey: fgHash,
		},
		Members: usersHashes,
	}
	// log
	t.Log("Leader:         ", leaderHash.Hex())
	t.Log("FallGuy:        ", fgHash.Hex())
	t.Log("FallGuy Schema: ", fgSchemaHash.Hex())
	for i, uh := range usersHashes {
		t.Logf("User #%d:         %s", i, uh.Hex())
	}
	// - - missing
	// + - exists
	// w - want
	// d - don't know about yet
	// ------------------------
	// members:              -4 w
	// leader:               -1 w
	// small group:          +1
	// man:                  -1 w
	// schema of Man         -1 w
	// schema of User:       +1
	// schema of SmallGroup: +1
	// ------------------------
	// want: 7
	root.Set(group)
	c.SetRoot(root)
	if list, err = c.Want(); err != nil {
		t.Error("unexpected error: ", err)
		return
	}
	if len(list) != 7 {
		t.Error("wrong length: want 7, got ", len(list))
		return
	}
	for _, k := range append(usersHashes,
		leaderHash,
		fgHash,
		fgSchemaHash,
	) {
		if _, ok := list[k]; !ok {
			t.Error("missing wanted object: ", k.Hex())
		}
	}
	// + leader
	c.Save(leader)
	// ------------------------
	// members:              -4 w
	// leader:               +1
	// small group:          +1
	// man:                  -1 w
	// schema of Man         -1 w
	// schema of User:       +1
	// schema of SmallGroup: +1
	// ------------------------
	// want: 6
	if list, err = c.Want(); err != nil {
		t.Error("unexpected error: ", err)
		return
	}
	if len(list) != 6 {
		t.Error("wrong length: want 6, got ", len(list))
		return
	}
	for _, k := range append(usersHashes,
		fgHash,
		fgSchemaHash,
	) {
		if _, ok := list[k]; !ok {
			t.Error("missing wanted object: ", k.Hex())
		}
	}
	// + man
	c.Save(fg)
	// ------------------------
	// members:              -4 w
	// leader:               +1
	// small group:          +1
	// man:                  +1
	// schema of Man         -1 w
	// schema of User:       +1
	// schema of SmallGroup: +1
	// ------------------------
	// want: 5
	if list, err = c.Want(); err != nil {
		t.Error("unexpected error: ", err)
		return
	}
	if len(list) != 5 {
		t.Error("wrong length: want 5, got ", len(list))
		return
	}
	for _, k := range append(usersHashes,
		fgSchemaHash,
	) {
		if _, ok := list[k]; !ok {
			t.Error("missing wanted object: ", k.Hex())
		}
	}
	// + schema of Man
	c.Save(getSchema(fg))
	// ------------------------
	// members:              -4 w
	// leader:               +1
	// small group:          +1
	// man:                  +1
	// schema of Man         +1
	// schema of User:       +1
	// schema of SmallGroup: +1
	// ------------------------
	// want: 4
	if list, err = c.Want(); err != nil {
		t.Error("unexpected error: ", err)
		return
	}
	if len(list) != 4 {
		t.Error("wrong length: want 4, got ", len(list))
		return
	}
	for _, k := range usersHashes {
		if _, ok := list[k]; !ok {
			t.Error("missing wanted object: ", k.Hex())
		}
	}
	// + members (1, 3)
	for i, u := range users {
		if i == 1 || i == 3 {
			c.Save(u)
		}
	}
	// ------------------------
	// members:              -4 (2) w
	// leader:               +1
	// small group:          +1
	// man:                  +1
	// schema of Man         +1
	// schema of User:       +1
	// schema of SmallGroup: +1
	// ------------------------
	// want: 2
	if list, err = c.Want(); err != nil {
		t.Error("unexpected error: ", err)
		return
	}
	if len(list) != 2 {
		t.Error("wrong length: want 2, got ", len(list))
		return
	}
	for i, k := range usersHashes {
		if i == 0 || i == 2 {
			if _, ok := list[k]; !ok {
				t.Error("missing wanted object: ", k.Hex())
			}
		}
	}
	// + members (0, 2)
	for i, u := range users {
		if i == 0 || i == 2 {
			c.Save(u)
		}
	}
	// ------------------------
	// members:              +4
	// leader:               +1
	// small group:          +1
	// man:                  +1
	// schema of Man         +1
	// schema of User:       +1
	// schema of SmallGroup: +1
	// ------------------------
	// want: 0
	if list, err = c.Want(); err != nil {
		t.Error("unexpected error: ", err)
		return
	}
	if len(list) != 0 {
		t.Error("wrong length: want 0, got ", len(list))
		return
	}
}

func TestContainer_want(t *testing.T) {
	// TODO
}

func TestContainer_addMissing(t *testing.T) {
	// TODO
}

func Test_skyobjectTag(t *testing.T) {
	// TODO
}

func Test_tagSchemaName(t *testing.T) {
	// TODO
}

func TestContainer_schemaByTag(t *testing.T) {
	// TODO
}
