package skyobjects

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

type TestItem struct {
	A string
	B int64
}

// Before testing `GetAllOfSchema` function, we first need to save the schema of
// 'TestItem', and then save some objects to the container. Here we demonstate
// 3 different methods of storing objects in the container; 'Save', 'SaveData',
// and 'SaveObject'.
func TestGetAllOfSchema(t *testing.T) {
	c := NewContainer(data.NewDB())
	schemaKey := c.SaveSchema(TestItem{})

	test0 := TestItem{A: "0", B: 0}
	test0Data := encoder.Serialize(test0)
	test0Href := Href{SchemaKey: schemaKey, Data: test0Data}
	c.Save(test0Href)

	test1 := TestItem{A: "1", B: 1}
	test1Data := encoder.Serialize(test1)
	c.SaveData(schemaKey, test1Data)

	test2 := TestItem{A: "2", B: 2}
	c.SaveObject(schemaKey, test2)

	testKeys := c.GetAllOfSchema(schemaKey)
	for i, key := range testKeys {
		h, e := c.Get(key)
		if e != nil {
			t.Error(e)
		}
		var test TestItem
		e = encoder.DeserializeRaw(h.Data, &test)
		if e != nil || test.A == "" && test.B == 0 {
			t.Error("deserialize failed")
		}
		t.Logf("i = %d, object = %v", i, test)
	}
}

type Node struct {
	A string
}

type Ref struct {
	A string
	B []cipher.SHA256 `skyobjects:"href"`
}

func TestReferencing(t *testing.T) {
	c := NewContainer(data.NewDB())
	nodeSchKey := c.SaveSchema(Node{})
	refSchKey := c.SaveSchema(Ref{})

	node := Node{"main"}
	nodeKey := c.SaveObject(nodeSchKey, node)

	ref1 := Ref{"1", []cipher.SHA256{nodeKey}}
	ref1Key := c.SaveObject(refSchKey, ref1)

	ref2 := Ref{"2", []cipher.SHA256{nodeKey, ref1Key}}
	ref2Key := c.SaveObject(refSchKey, ref2)

	ref3 := Ref{"3", []cipher.SHA256{nodeKey, ref1Key, ref2Key}}
	c.SaveObject(refSchKey, ref3)

	t.Log("Finding objects that reference 'node':")
	refsToNode := c.GetReferencesFor(nodeKey)
	if n := len(refsToNode); n != 3 {
		t.Errorf("expected 3, got %d", n)
	} else {
		t.Log("OK")
	}
}

type Ref2 struct {
	A string
	B cipher.SHA256 `skyobjects:"href"`
}

func TestReferencing2(t *testing.T) {
	c := NewContainer(data.NewDB())
	nodeSchKey := c.SaveSchema(Node{})
	refSchKey := c.SaveSchema(Ref2{})

	node := Node{"main"}
	nodeDat := encoder.Serialize(node)
	nodeKey := c.SaveData(nodeSchKey, nodeDat)
	nkey := nodeKey

	ref1 := Ref2{"1", nkey}
	ref1Dat := encoder.Serialize(ref1)
	c.SaveData(refSchKey, ref1Dat)

	ref2 := Ref2{"2", nkey}
	ref2Dat := encoder.Serialize(ref2)
	c.SaveData(refSchKey, ref2Dat)

	ref3 := Ref2{"3", nkey}
	ref3Dat := encoder.Serialize(ref3)
	c.SaveData(refSchKey, ref3Dat)

	t.Log("Finding objects that reference 'node':")
	refsToNode := c.GetReferencesFor(nodeKey)
	if n := len(refsToNode); n != 3 {
		t.Errorf("expected 3, got %d", n)
	} else {
		t.Log("OK")
	}
}

type Child struct {
	Name     string
	Age      uint32
	Children []cipher.SHA256 `skyobjects:"href,child"`
}

func TestGetChildren(t *testing.T) {
	c := NewContainer(data.NewDB())
	_childType := c.SaveSchema(Child{})

	var (
		grandchildrenKeys []cipher.SHA256
		childrenKeys      []cipher.SHA256
	)

	// Make some grandchildren.
	for i := 0; i < 20; i++ {
		grandchild := Child{Name: "Grandchild" + strconv.Itoa(i), Age: 20}
		grandchildrenKeys = append(grandchildrenKeys, c.SaveObject(_childType, grandchild))
	}

	// Make some children.
	for i := 0; i < 5; i++ {
		child := Child{Name: "Child" + strconv.Itoa(i), Age: 40}
		for j := i * 4; j < i*4+4; j++ {
			child.Children = append(child.Children, grandchildrenKeys[j])
		}
		childrenKeys = append(childrenKeys, c.SaveObject(_childType, child))
	}

	// Make root.
	root := NewRoot(0)
	root.AddChildren(childrenKeys...)
	c.SaveRoot(root)

	// Print tree.
	for _, childKey := range c.GetCurrentRootChildren() {
		var child Child
		childRef, _ := c.Get(childKey)
		childRef.Deserialize(&child)
		fmt.Printf("CHILD: Name = %s, Age = %d\n", child.Name, child.Age)

		for grandchildKey := range c.GetChildren(childRef) {
			var grandchild Child
			grandchildRef, _ := c.Get(grandchildKey)
			grandchildRef.Deserialize(&grandchild)
			fmt.Printf("\tGRANDCHILD: Name = %s, Age = %d\n", grandchild.Name, grandchild.Age)
		}
	}
}
