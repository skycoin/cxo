package skyobject

import (
	"testing"
)

func TestNewContainer(t *testing.T) {
	t.Run("with registry", func(t *testing.T) {
		r := NewRegistry()
		c := NewContainer(r)
		if c.db == nil {
			t.Error("misisng database")
		}
		if c.coreRegistry != r {
			t.Error("wrong core registry")
		}
		if len(c.registries) == 0 {
			t.Error("registry missing in map")
		} else if c.registries[r.Reference()] != r {
			t.Error("wrong registry in map")
		} else if r.done != true {
			t.Error("Done wasn't called")
		}
	})
	t.Run("wihtout registry", func(t *testing.T) {
		///
	})
}

func TestNewContainerDB(t *testing.T) {
	//
}

func TestContainer_CoreRegistry(t *testing.T) {
	//
}

func TestContainer_Registry(t *testing.T) {
	//
}

func TestContainer_DB(t *testing.T) {
	//
}

func TestContainer_Get(t *testing.T) {
	//
}

func TestContainer_Save(t *testing.T) {
	//
}

func TestContainer_SaveArray(t *testing.T) {
	//
}

func TestContainer_Dynamic(t *testing.T) {
	//
}

func TestContainer_NewRoot(t *testing.T) {
	//
}

func TestContainer_LastRoot(t *testing.T) {
	//
}

func TestContainer_LastFullRoot(t *testing.T) {
	//
}

func TestContainer_RootBySeq(t *testing.T) {
	//
}

func TestContainer_GC(t *testing.T) {
	//
}

func TestContainer_RootsGC(t *testing.T) {
	//
}

func TestContainer_RegsitryGC(t *testing.T) {
	//
}

func TestContainer_ObjectsGC(t *testing.T) {
	//
}
