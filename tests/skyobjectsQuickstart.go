package main

import (
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

func main() {
	// Create database.
	db := data.NewDB()

	// Create container with database.
	container := skyobject.SkyObjects(db)

	// Use container.
	container.Inspect()
}
