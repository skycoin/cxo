package skyobjects

import (
	"testing"

	"github.com/skycoin/cxo/data"
)

// TestItem represents a test item.
type TestItem struct {
	A string
	B int
}

func TestStoreAndRetrieve(t *testing.T) {
	c := NewContainer(data.NewDB())

	item := TestItem{"test", 5}
	itemRef := NewObjectReference(item)
	c.Store(itemRef)
	// Item will only have key after we store it in container.
	itemKey := itemRef.GetKey()

	// Retrieve into retrievedItem.
	var retrievedItem TestItem
	retrievedItemRef := c.Get(itemKey)
	retrievedItemRef.Deserialize(&retrievedItem)

	if retrievedItem.A != item.A {
		t.Error("member value A does not match up")
	} else if retrievedItem.B != item.B {
		t.Error("member value B does not match up")
	}
}
