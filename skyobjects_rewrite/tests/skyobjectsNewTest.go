package main

import (
	"fmt"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobjects"
)

// TestType1 is for testing.
type TestType1 struct {
	A string
	B string
}

// TestType2 is for testing.
type TestType2 struct {
	A string
	B int16
}

// TestType3 is for testing.
type TestType3 struct {
	A int
	B int
}

func main() {
	// container := skyobjects.NewContainer(data.NewDB())
	//
	// fmt.Print("\n[TEST: container.Schema.Register]\n\n")
	// container.Schema.Register(TestType1{}, TestType2{}, TestType3{})
	// for _, schema := range container.Schema.GetAll() {
	// 	fmt.Println(schema.String())
	// }
	//
	// fmt.Print("\n[TEST: container.Schema.GetOfType]\n\n")
	// _, schema := container.Schema.GetOfType("TestType1")
	// fmt.Println(schema.String())
	// _, schema = container.Schema.GetOfType("TestType2")
	// fmt.Println(schema.String())
	// _, schema = container.Schema.GetOfType("TestType3")
	// fmt.Println(schema.String())
	//
	// fmt.Print("\n[TEST: add objects]\n\n")
	//
	// obj := TestType1{"yooo", "yaaa"}
	// objRef := skyobjects.NewObjectReference(obj)
	// container.Store(objRef)
	//
	// obj0 := TestType2{"a", 0}
	// obj1 := TestType2{"b", 1}
	// obj2 := TestType2{"c", 2}
	// obj3 := TestType2{"d", 3}
	// obj4 := TestType2{"e", 4}
	// obj5 := TestType2{"f", 5}
	// arrRef := skyobjects.NewArrayReference(obj0, obj1, obj2, obj3, obj4, obj5)
	// container.Store(arrRef)
	//
	// fmt.Println("Keys of objects stored in array:")
	// for _, key := range arrRef.GetValues() {
	// 	fmt.Println(" -", key.GetKey().Hex())
	// }
	//
	// fmt.Println("\nInspect container:")
	// container.Inspect()
	//
	// fmt.Print("\n[TEST: container.Get]\n\n")
	// retrieved := container.Get(arrRef.GetKey())
	// fmt.Println(retrieved.GetType().Hex())
	//
	// for _, objRef := range retrieved.GetValues() {
	// 	var obj TestType2
	// 	if e := objRef.Deserialize(&obj); e != nil {
	// 		fmt.Println(e)
	// 	}
	// 	fmt.Println(" -", obj)
	// }

	func() {
		// fmt.Print("\n[TEST: linking objects]\n\n")

		type Board struct {
			Name    string
			Threads skyobjects.ArrayReference
		}

		type Thread struct {
			Name string
		}

		c := skyobjects.NewContainer(data.NewDB())
		c.Schema.Register(Board{}, Thread{})

		t1 := Thread{"t1"}
		t2 := Thread{"t2"}
		t3 := Thread{"t3"}
		t4 := Thread{"t4"}
		t5 := Thread{"t5"}

		ta1 := skyobjects.NewArrayReference([]Thread{t1, t2, t3})
		ta2 := skyobjects.NewArrayReference(t4, t5)
		c.Store(ta1)
		c.Store(ta2)

		b1 := Board{"b1", *ta1}
		b2 := Board{"b2", *ta2}

		bo1 := skyobjects.NewObjectReference(b1)
		bo2 := skyobjects.NewObjectReference(b2)
		c.StoreToRoot(bo1)
		c.StoreToRoot(bo2)

		// boardRef := c.Get(bo1.GetKey())
		// for _, threadRef := range boardRef.GetField("Threads").GetValues() {
		// 	var thread Thread
		// 	threadRef.Deserialize(&thread)
		// 	fmt.Println(thread)
		// }
		// fmt.Println()

		// c.Inspect()

		rootValues := c.GetRootValues()
		c.RemoveFromRoot(rootValues[0].GetKey())

		// fmt.Print("\n[TEST: getting root values]\n\n")
		//
		// boardRefs := c.GetRootValues()
		// for _, boardRef := range boardRefs {
		// 	var board Board
		// 	boardRef.Deserialize(&board)
		// 	fmt.Println(" -", board.Name)
		//
		// 	for _, threadRef := range boardRef.GetField("Threads").GetValues() {
		// 		var thread Thread
		// 		threadRef.Deserialize(&thread)
		// 		fmt.Println(" \t-", thread.Name)
		// 	}
		// }
		//
		// fmt.Print("\n[TEST: for exceptions]\n\n")
		// ref := c.Get(cipher.SHA256{})
		// if ref != nil {
		// 	fmt.Println(ref.GetKey())
		// }
		//
		// fmt.Print("\n[TEST: get all of schema]\n\n")
		//
		// fmt.Println("Boards:")
		// schemaKey, _ := c.Schema.GetOfType("Board")
		// for _, k := range c.GetAllOfSchema(schemaKey) {
		// 	var board Board
		// 	c.Get(k).Deserialize(&board)
		// 	fmt.Println(" -", board.Name)
		// }
		//
		// fmt.Println("\nThreads:")
		// schemaKey, _ = c.Schema.GetOfType("Thread")
		// for _, k := range c.GetAllOfSchema(schemaKey) {
		// 	var thread Thread
		// 	c.Get(k).Deserialize(&thread)
		// 	fmt.Println(" -", thread.Name)
		// }

		fmt.Print("\n[TEST: for garbage collection 1]\n\n")

		rmMap := c.GetAllToRemove()
		for k, v := range rmMap {
			fmt.Printf(" - key: '%s', name: `%s`\n", k.Hex(), v.GetSchema(c).Name)
		}

		schemaKey, _ := c.Schema.GetOfType("RootReference")
		for _, key := range c.GetAllOfSchema(schemaKey) {
			fmt.Println(" - RootRef:", key.Hex())
		}

		c.Inspect()

		fmt.Print("\n[TEST: for garbage collection 2]\n\n")

		c.CollectGarbage(1000000)
		// c.ClearRoot()

		rmMap = c.GetAllToRemove()
		for k, v := range rmMap {
			fmt.Printf(" - key: '%s', name: `%s`\n", k.Hex(), v.GetSchema(c).Name)
		}

		schemaKey, _ = c.Schema.GetOfType("RootReference")
		for _, key := range c.GetAllOfSchema(schemaKey) {
			fmt.Println(" - RootRef:", key.Hex())
		}

		c.Inspect()
	}()
}
