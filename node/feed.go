package node

import (
	"sync"

	"guthub.com/skycoin/skycoin/src/cipher"
)

// feed of the Node
type nodeFeed struct {
	n *Node // back referece

	cs map[*Conn]struct{}   // connections of the feed
	hs map[uint64]*nodeHead // heads of the feed

	await sync.WaitGroup // wait for heads
}

func newNodeFeed(node *Node) (n *nodeFeed) {

	n = new(nodeFeed)
	n.n = node
	n.cs = make(map[*Conn]struct{})
	n.hs = make(map[uint64]*nodeHead)

	return
}

func (n *nodeFeed) addConn(c *Conn) {
	n.cs[c] = struct{}{}
}

func (n *nodeFeed) delConn(c *Conn) {
	delete(n.cs, c)

	for _, nh := range n.hs {
		nh.delConn(c)
	}

}

func (n *nodeFeed) hasConn(c *Conn) (ok bool) {
	_, ok = n.cs[c]
	return
}

func (n *nodeFeed) receivedRoot(cr connRoot) {

	var nh, ok = n.hs[cr.r.Nonce]

	if ok == false {
		nh = newNodeHead(n)
		n.hs[cr.r.Nonce] = nh
	}

	nh.receivedRoot(cr)

}
