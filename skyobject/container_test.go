package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	//"github.com/skycoin/cxo/data"
)

func TestNewContainer(t *testing.T) {
	//
}

func TestContainer_AddRegistry(t *testing.T) {
	//
}

func TestContainer_CoreRegistry(t *testing.T) {
	//
}

func TestContainer_Registry(t *testing.T) {
	//
}

func TestContainer_WantRegistry(t *testing.T) {
	//
}

func TestContainer_Registries(t *testing.T) {
	//
}

func TestContainer_DB(t *testing.T) {
	//
}

func TestContainer_Get(t *testing.T) {
	//
}

func TestContainer_Set(t *testing.T) {
	//
}

func TestContainer_NewRoot(t *testing.T) {
	//
}

func TestContainer_NewRootReg(t *testing.T) {
	//
}

func TestContainer_AddRootPack(t *testing.T) {
	//
}

func TestContainer_LastRoot(t *testing.T) {
	//
}

func TestContainer_LastRootSk(t *testing.T) {
	//
}

func TestContainer_LastFullRoot(t *testing.T) {
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	r, err := c.NewRoot(pk, sk)
	if err != nil {
		t.Error(err)
		return
	}
	r.Inject("cxo.User", User{"Alice", 15, nil})
	r.Inject("cxo.User", User{"Eva", 16, nil})
	r.Inject("cxo.User", User{"Ammy", 17, nil})
	full := c.LastFullRoot(pk)
	if full == nil {
		t.Error("misisng last full root")
	}
}

func TestContainer_Feeds(t *testing.T) {
	//
}

func TestContainer_WantFeed(t *testing.T) {
	//
}

func TestContainer_GotFeed(t *testing.T) {
	//
}

func TestContainer_DelFeed(t *testing.T) {
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
