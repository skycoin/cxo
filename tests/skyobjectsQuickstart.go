package main

import (
	"fmt"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

// Board is a test type.
type Board struct {
	Name string
}

func main() {
	c := skyobject.SkyObjects(data.NewDB())
	c.RegisterSchema(Board{})

	// Store object (Method 1):
	board1 := Board{"board1"}
	boardSchemaKey, _ := c.GetSchemaKey("Board")
	c.SaveObject(boardSchemaKey, board1)

	// Store object (Method 2):
	board2 := Board{"board2"}
	bl := skyobject.NewObject(board2)
	c.Save(&bl)

	c.Inspect()

	for _, key := range c.GetAllBySchema(boardSchemaKey) {
		var bd Board
		boardRef := c.GetObjRef(key)
		boardRef.Deserialize(&bd)
		fmt.Println("", "-", bd.Name)
	}
}
