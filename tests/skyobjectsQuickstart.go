package main

import (
	// "fmt"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

// Board is a test type.
type Board struct {
	Name  string
	Array []string
}

// Parent type.
type Parent struct {
	Name     string
	Children skyobject.HashArray
}

// type Parent struct {
// 	Name  string
// 	Child skyobject.HashObject
// }

// Child type.
type Child struct {
	Name string
}

func main() {
	c := skyobject.SkyObjects(data.NewDB())
	c.RegisterSchema(Board{})

	// Store object (Method 1):
	board1 := Board{"board1", []string{"1", "2", "3"}}
	boardSchemaKey, _ := c.GetSchemaKey("Board")
	c.SaveObject(boardSchemaKey, board1)

	// Store object (Method 2):
	board2 := Board{"board2", []string{"1", "2", "3"}}
	bl := skyobject.NewObject(board2)
	c.Save(&bl)

	children := []Child{{"Ben"}, {"James"}, {"Ryan"}}
	childrenRef := skyobject.NewArray(children)
	c.Save(&childrenRef)

	parent := Parent{Name: "Bob", Children: childrenRef}
	parentRef := skyobject.NewObject(parent)
	c.Save(&parentRef)

	// child := Child{Name: "Ben"}
	// childRef := skyobject.NewObject(child)
	// c.Save(&childRef)

	// parent := Parent{Name: "Bob", Child: childRef}
	// parentRef := skyobject.NewObject(parent)
	// c.Save(&parentRef)

	c.Inspect()

	// for _, key := range c.GetAllBySchema(boardSchemaKey) {
	// 	var bd Board
	// 	boardRef := c.GetObjRef(key)
	// 	boardRef.Deserialize(&bd)
	// 	fmt.Println("", "-", bd.Name)
	// }
}
