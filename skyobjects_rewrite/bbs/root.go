package bbs

import (
	"fmt"
	"time"

	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/skycoin/src/cipher"
)

// Root is the Bbs reference tree used for retrieving dependencies.
type Root struct {
	boards  []*RefNode
	threads []*RefNode
	updated uint64
	count   uint64
	c       skyobject.ISkyObjects
}

// NewRoot creates a new Root.
func NewRoot(c skyobject.ISkyObjects) Root { return Root{c: c} }

// GetDescendants gets all the descendants.
func (t *Root) GetDescendants() []*RefNode { return append(t.boards, t.threads...) }

// GetDescendant gets a descendant of specified key.
func (t *Root) GetDescendant(key cipher.SHA256) *RefNode {
	for _, oldNode := range append(t.boards, t.threads...) {
		if oldNode.key == key {
			return oldNode
		}
	}
	return nil
}

// GetBoards gets all the boards.
func (t *Root) GetBoards() []*RefNode { return t.boards }

// GetBoard gets a board of specified key.
func (t *Root) GetBoard(key cipher.SHA256) *RefNode {
	for _, board := range t.boards {
		if board.key == key {
			return board
		}
	}
	return nil
}

// GetCount gets the number of descendants.
func (t *Root) GetCount() uint64 { return t.count }

// GetLastUpdated gets the epoch time of last update.
func (t *Root) GetLastUpdated() uint64 { return t.updated }

// Update updates the root.
func (t *Root) Update() {
	t.threads = []*RefNode{}
	for _, child := range t.boards {
		child.Update(t)
	}
	t.count = uint64(len(t.boards) + len(t.threads))
	t.updated = uint64(time.Now().UnixNano())
}

// Has determines whether we have this node.
func (t *Root) Has(newNode *RefNode) bool {
	switch newNode.level {
	case 1: // Is Board -- >
		for _, oldNode := range t.boards {
			if oldNode.key == newNode.key {
				return true
			}
		}
	case 2: // Is Thread -- >
		for _, oldNode := range t.threads {
			if oldNode.key == newNode.key {
				return true
			}
		}
	}
	return false
}

func (t *Root) addBoard(key cipher.SHA256) error {
	for _, board := range t.boards {
		if board.key == key {
			return fmt.Errorf("child with key %s already exists", key.Hex())
		}
	}
	board := &RefNode{key: key, level: 1}
	t.boards = append(t.boards, board)
	return nil
}

func (t *Root) removeBoard(key cipher.SHA256) error {
	for i, board := range t.boards {
		if board.key == key {
			t.boards[len(t.boards)-1], t.boards[i] = t.boards[i], t.boards[len(t.boards)-1]
			t.boards = t.boards[:len(t.boards)-1]
			return nil
		}
	}
	return fmt.Errorf("child with key %s already exists", key.Hex())
}

func (t *Root) addDescendant(newNode *RefNode) *RefNode {
	switch newNode.level {
	case 1: // Is Board -- >
		for _, oldNode := range t.boards {
			if oldNode.key == newNode.key {
				return oldNode
			}
		}
		t.boards = append(t.boards, newNode)

	case 2: // Is Thread -- >
		for _, oldNode := range t.threads {
			if oldNode.key == newNode.key {
				return oldNode
			}
		}
		t.threads = append(t.threads, newNode)
	}
	return newNode
}

// RefNode is a node of the reference tree.
type RefNode struct {
	key      cipher.SHA256
	children []*RefNode
	level    int
}

// GetKey retrieves the key.
func (n *RefNode) GetKey() cipher.SHA256 { return n.key }

// GetChildren retrieves all the children.
func (n *RefNode) GetChildren() []*RefNode { return n.children }

// GetLevel retrieves the level within tree of node.
func (n *RefNode) GetLevel() int { return n.level }

// Update updates the node.
func (n *RefNode) Update(t *Root) {
	n.children = []*RefNode{}

	// Get fields of current objects which are either 'hashobject' or 'hasharray'.
	objref := t.c.GetObjRef(n.key)
	for _, field := range objref.GetSchema().Fields {
		switch field.Type {
		case "hashobject":
			fieldref, _ := objref.GetFieldAsObj(field.Name)
			child, e := n.addChild(t, fieldref.GetKey(), n.level+1)
			if e != nil {
				fmt.Println(e)
				break
			}
			child.Update(t)

		case "hasharray":
			fieldref, _ := objref.GetFieldAsObj(field.Name)
			keyarray, _ := fieldref.GetValuesAsKeyArray()
			for _, key := range keyarray {
				child, e := n.addChild(t, key, n.level+1)
				if e != nil {
					fmt.Println(e)
					break
				}
				child.Update(t)
			}
		}
	}
}

// Add adds a child to the node.
func (n *RefNode) addChild(t *Root, key cipher.SHA256, lvl int) (child *RefNode, e error) {
	child = &RefNode{key: key, level: lvl}
	if t.Has(child) {
		return nil, fmt.Errorf("key %s already exisits", key.Hex())
	}
	n.children = append(n.children, t.addDescendant(child))
	return
}
