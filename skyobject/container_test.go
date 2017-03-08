package skyobject

import (
	"reflect"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

func TestNewContainer(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("missing panic")
			}
		}()
		NewContainer(nil)
	})
	t.Run("intenals", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		if c.db == nil {
			t.Error("missing db")
		}
		if c.registry == nil {
			t.Error("misisng registry")
		}
		if c.root != nil {
			t.Error("non-nil root")
		}
	})
}

func TestContainer_Root(t *testing.T) {
	c := NewContainer(data.NewDB())
	if c.Root() != nil {
		t.Error("non-nil root from fresh container")
	}
	r := NewRoot(User{})
	c.root = r
	if c.Root() != r {
		t.Error("wrong root")
	}
}

func TestContainer_SetRoot(t *testing.T) {
	c := NewContainer(data.NewDB())
	ro, rn, rr := NewRoot(User{}), NewRoot(User{}), NewRoot(User{})
	c.SetRoot(rn)
	if c.root != rn {
		t.Error("wrong internal root")
	}
	c.SetRoot(ro)
	if c.root != rn {
		t.Error("root replaced with old one")
	}
	c.SetRoot(rr)
	if c.root != rr {
		t.Error("root was not replaced with new one")
	}
}

func TestContainer_Register(t *testing.T) {
	c := NewContainer(data.NewDB())
	c.Register("user", User{})
	if len(c.registry) != 1 {
		t.Error("don't register")
		return
	}
	for k, v := range c.registry {
		if k != "user" {
			t.Error("registered with wrong name")
		}
		if v.Name != reflect.TypeOf(User{}).Name() {
			t.Error("wrong schema registered")
		}
	}
}

func TestContainer_Schema(t *testing.T) {
	c := NewContainer(data.NewDB())
	if s, ok := c.Schema("user"); ok {
		t.Error("returns schema that should not have")
	} else if s != nil {
		t.Error("it's crazy")
	}
	c.Register("user", User{})
	if s, ok := c.Schema("user"); !ok {
		t.Error("can't return schema that should have")
	} else if s == nil {
		t.Error("misisng schema")
	} else if s.Name != reflect.TypeOf(User{}).Name() {
		t.Error("wrong schema")
	}
}

func TestContainer_Save(t *testing.T) {
	c := NewContainer(data.NewDB())
	key := c.Save(User{})
	if key == (cipher.SHA256{}) {
		t.Error("save returns empty key")
	}
	if c.db.Stat().Total != 1 {
		t.Error("wrong objects count in db")
	}
	if _, ok := c.db.Get(key); !ok {
		t.Error("saved object is not saved in db or returned key is wrong")
	}
}

func TestContainer_SaveArray(t *testing.T) {
	c := NewContainer(data.NewDB())
	keys := c.SaveArray(User{Age: 12}, User{Age: 13}, User{Age: 14})
	if len(keys) != 3 {
		t.Error("wrong keys count: want 3, got ", len(keys))
	}
	for _, k := range keys {
		if k == (cipher.SHA256{}) {
			t.Error("SaveArray returns empty key")
		}
	}
	if c.db.Stat().Total != 3 {
		t.Error("wrong objects count in db")
	}
	for _, k := range keys {
		if _, ok := c.db.Get(k); !ok {
			t.Error("saved object is not saved in db or returned key is wrong")
		}
	}
}

func Test_skyobjectTag(t *testing.T) {
	if skyobjectTag(encoder.StructField{}) != "" {
		t.Error("got skyobject tag for empty tag")
	}
	if skyobjectTag(encoder.StructField{Tag: `enc:"-",skybject:""`}) != "" {
		t.Error("got skyobject tag for empty tag")
	}
	tag := `skyobject:"blah,blah,blah"`
	if skyobjectTag(encoder.StructField{Tag: tag}) != "blah,blah,blah" {
		t.Error("wrong or missign skyobject tag")
	}
}

func Test_tagSchemaName(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s, err := tagSchemaName("")
		if err != nil {
			t.Error("unexpected error")
		}
		if s != "" {
			t.Error("unexpected schema name")
		}
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := tagSchemaName("schema=schema=name")
		if err == nil {
			t.Error("misisng error")
		}
	})
	t.Run("valid", func(t *testing.T) {
		s, err := tagSchemaName("href,schema=Thread")
		if err != nil {
			t.Error("unexpected error")
		}
		if s != "Thread" {
			t.Error("wrong schema name")
		}
	})
}

func TestContainer_schemaByTag(t *testing.T) {
	c := NewContainer(data.NewDB())
	c.Register("User", User{})
	t.Run("dynamic", func(t *testing.T) {
		s, err := c.schemaByTag("")
		if err != nil {
			t.Error("unexpected error")
		}
		if s != dynamicHrefSchema {
			t.Error("unexpected schema")
		}
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := c.schemaByTag("schema=schema=name")
		if err == nil {
			t.Error("misisng error")
		}
	})
	t.Run("registered", func(t *testing.T) {
		s, err := c.schemaByTag("href,schema=User")
		if err != nil {
			t.Error("unexpected error")
		}
		if s.Name != reflect.TypeOf(User{}).Name() {
			t.Error("wrong schema name")
		}
	})
}

type SmallGroup struct {
	Name     string
	Leader   cipher.SHA256   `skyobject:"href,schema=User"`
	Outsider cipher.SHA256   // not a reference
	FallGuy  cipher.SHA256   `skyobject:"href"` // dynamic
	Users    []cipher.SHA256 `skyobject:"href,schema=User"`
}

func preparedContainer() *Container {
	c := NewContainer(data.NewDB())
	c.Register("User", User{})
	leader := c.Save(User{Name: "leader", Age: 30})
	fallguy := c.Save(c.NewDynamicHref(User{Name: "fallguy", Age: 31}))
	users := c.SaveArray(
		User{"Alice", 21, ""},
		User{"Bob", 25, ""},
		User{"Eva", 27, ""},
		User{"Peter", 18, ""},
	)
	c.SetRoot(NewRoot(SmallGroup{
		Name:     "any",
		Leader:   leader,
		Outsider: cipher.SHA256{},
		FallGuy:  fallguy,
		Users:    users,
	}))
	return c
}

func TestContainer_singleHref(t *testing.T) {
	// TODO
}

func TestContainer_arrayHref(t *testing.T) {
	// TODO
}

func TestContainer_Childs(t *testing.T) {
	c := preparedContainer()
	// we will use root to explore
	root := c.Root()
	ch, err := c.Childs(root.Schema, root.Root)
	if err != nil {
		t.Error("unexpected error")
	}
	// TODO
	_ = ch
}
