package skyobjects

import (
	"testing"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// // TestItem represents a test item.
type TestItem struct {
	A string
	B int64
}

func TestGetAllOfSchema(t *testing.T) {
	c := NewContainer(data.NewDB())
	schemaKey := c.SaveSchema(TestItem{})

	test0 := TestItem{A: "0", B: 0}
	test0Data := encoder.Serialize(test0)
	c.Save(schemaKey, test0Data)

	test1 := TestItem{A: "1", B: 1}
	test1Data := encoder.Serialize(test1)
	c.Save(schemaKey, test1Data)

	test2 := TestItem{A: "2", B: 2}
	test2Data := encoder.Serialize(test2)
	c.Save(schemaKey, test2Data)

	testKeys := c.GetAllOfSchema(schemaKey)
	for i, key := range testKeys {
		_, data, e := c.Get(key)
		if e != nil {
			t.Error(e)
		}
		var test TestItem
		e = encoder.DeserializeRaw(data, &test)
		if e != nil || test.A == "" && test.B == 0 {
			t.Error("deserialize failed")
		}
		t.Logf("i : %d, object: %v", i, test)
	}
}

type Node struct {
	A string
}

type Ref struct {
	A HashArray
}

func TestReferencing(t *testing.T) {
	c := NewContainer(data.NewDB())
	nodeSchKey := c.SaveSchema(Node{})
	refSchKey := c.SaveSchema(Ref{})

	node := Node{"main"}
	nodeDat := encoder.Serialize(node)
	nodeKey := c.Save(nodeSchKey, nodeDat)
	t.Log(nodeKey)

	ref1 := Ref{HashArray{nodeKey}}
	ref1Dat := encoder.Serialize(ref1)
	ref1Key := c.Save(refSchKey, ref1Dat)
	t.Log(ref1Key)

	ref2 := Ref{HashArray{nodeKey}}
	ref2Dat := encoder.Serialize(ref2)
	ref2Key := c.Save(refSchKey, ref2Dat)
	t.Log(ref2Key)

	ref3 := Ref{HashArray{nodeKey}}
	ref3Dat := encoder.Serialize(ref3)
	ref3Key := c.Save(refSchKey, ref3Dat)
	t.Log(ref3Key)

}
