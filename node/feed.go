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
	ho []uint64             // heads order (max heads limit)

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

	// do we have the connection?

	if _, ok := n.cs[cr.c]; ok == false {
		return // the connection is not subscribed to the feed
	}

	var nh, ok = n.hs[cr.r.Nonce]

	if ok == false {

		// max heads limit
		if mh := n.n.config.MaxHeads; mh > 0 && len(n.ho) == mh {

			// max heads
			var torm = n.ho[0]             // to remove
			copy(n.ho, n.ho[1:])           // shift all
			n.ho[len(n.ho)-1] = cr.r.Nonce // push

			var tormh = n.hs[torm]

			tormh.errq <- ErrMaxHeadsLimit
			tormh.close() // wait

			delete(n.hs, torm) // remove

		}

		nh = newNodeHead(n)
		n.hs[cr.r.Nonce] = nh
	}

	nh.receivedRoot(cr)

}
