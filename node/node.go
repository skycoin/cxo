package node

import (
	"errors"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

type Node struct {
	db    *data.DB
	so    *skyobject.Container
	feeds []cipher.PubKey
}

func NewNode(db *data.DB, so *skyobject.Container) (n *Node) {
	if db == nil {
		panic("nil db")
	}
	if so == nil {
		panic("nil so")
	}
	n = new(Node)
	n.db = db
	n.so = so
	return
}

// Subscribe to given feed. The method returns false
// if the Node already subscribed to the feed
func (n *Node) Subscribe(feed cipher.PubKey) (add bool) {
	for _, f := range n.feeds {
		if f == feed {
			return // false => already subscribed
		}
	}
	n.feeds, add = append(n.feeds, feed), true
	return
}

// Unsubscribe from given feed. The method returns
// false if given feed not found in ubscriptions
func (n *Node) Unsubscribe(feed cipher.PubKey) (remove bool) {
	for i, f := range n.feeds {
		if f == feed {
			n.feeds, remove = append(n.feeds[:i], n.feeds[i+1:]...), true
			return
		}
	}
	return // false => not subscribed
}

var (
	// ErrNoRootObject occurs when an action requires root object
	// but the root object doesn't exists
	ErrNoRootObject = errors.New("no root object")
	// ErrNotSubscribed occurs when an action requires subscription
	// to a feed, but the Node doesn't subscribed to the feed
	ErrNotSubscribed = errors.New("not subscribed")
)

// Want returns set of objects given feed
// don't have but knows about
func (n *Node) Want(feed cipher.PubKey) (wn []cipher.SHA256, err error) {
	for _, f := range n.feeds {
		if f == feed {
			root := n.so.Root(feed)
			if root == nil {
				err = ErrNoRootObject
				return
			}
			var set skyobject.Set
			if set, err = root.Want(); err != nil {
				return
			}
			if len(set) == 0 {
				return
			}
			wn = make([]cipher.SHA256, 0, len(set))
			for w := range set {
				wn = append(wn, cipher.SHA256(w))
			}
			return // done
		}
	}
	err = ErrNotSubscribed
	return
}

// Got returns set of objects given feed has got
func (n *Node) Got(feed cipher.PubKey) (gt []cipher.SHA256, err error) {
	for _, f := range n.feeds {
		if f == feed {
			root := n.so.Root(feed)
			if root == nil {
				err = ErrNoRootObject
				return
			}
			var set skyobject.Set
			if set, err = root.Got(); err != nil {
				return
			}
			if len(set) == 0 {
				return
			}
			gt = make([]cipher.SHA256, 0, len(set))
			for g := range set {
				gt = append(gt, cipher.SHA256(g))
			}
			return // done
		}
	}
	err = ErrNotSubscribed
	return
}

// Feeds returns list of subscriptions of the Node
func (n *Node) Feeds() []cipher.PubKey {
	return n.feeds
}

// Stat is short hand to get statistic of underlying database
func (n *Node) Stat() data.Stat {
	return n.db.Stat()
}
