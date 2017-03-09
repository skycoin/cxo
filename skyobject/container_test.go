package skyobject

import (
	"testing"

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

func TestContainer_Root(t *testing.T) {
	c := NewContainer(data.NewDB())
	if c.Root() != nil {
		t.Error("unexpected root")
	}
	// TODO
}

func TestContainer_SetRoot(t *testing.T) {
	// TODO
}

func TestContainer_Save(t *testing.T) {
	// TODO
}

func TestContainer_SaveArray(t *testing.T) {
	// TODO
}

func TestContainer_Want(t *testing.T) {
	// TODO
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
