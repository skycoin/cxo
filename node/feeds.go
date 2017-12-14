package node

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/registry"
)

// connection and feed
type connFeed struct {
	c *Conn
	f cipher.PubKey
}

// connection and received Root
type connRoot struct {
	c *Conn
	r *registry.Root
}

// feeds of the Node
type nodeFeeds struct {
	fs map[cipher.PubKey]*nodeFeed // feeds
	fl []cipher.PubKey             // clear-on-write

	addq chan cipher.PubKey // add feed
	delq chan cipher.PubKey // del feed

	addcfq chan connFeed // add connection to a feed
	delcfq chan connFeed // del connection from a feed

	listrq chan struct{}        // request list
	listq  chan []cipher.PubKey // receive list

	fr chan connRoot // received Root objects
}

func newNodeFeeds() (n *nodeFeeds) {

	n = new(nodeFeeds)

	n.fs = make(map[cipher.PubKey]*nodeFeed)

	// idle connection (not pending)
	n.fs[cipher.PubKey{}] = newNodeFeed()

	n.addq = make(chan cipher.PubKey) // add feed
	n.delq = make(chan cipher.PubKey) // del feed

	n.addcfq = make(chan connFeed) // add connection to a feed
	n.delcfq = make(chan connFeed) // del connection from a feed

	n.listrq = make(chan struct{})       // request list
	n.listq = make(chan []cipher.PubKey) // receive list

	n.fr = make(chan connRoot) // received Root objects

	return
}

func (n *nodeFeeds) handleAddFeed(pk cipher.PubKey) {

	if pk == (cipher.PubKey{}) {
		return // blank feed is special
	}

	if _, ok := n.fs[pk]; ok == true {
		return // already have the feed
	}

	n.fs[pk] = newNodeFeed()
	n.fl = nil
}

func (n *nodeFeeds) handleDelFeed(pk cipher.PubKey) {

	if pk == (cipher.PubKey) {
		return // blank feed is special
	}

	var nf, ok = n.fs[pk]

	if ok == false {
		return // doesn't have the feed, nothing ot delete
	}

	nf.close() // close the feed, terminating all internal

	delete(n.fs, pk)
	n.fl = nil

}

func (n *nodeFeeds) handleAddConnFeed(cf connFeed) {

	// if the cf.f is blank the this connection is new

	if cf.f == (cipher.PubKey{}) {
		n.fs[cipher.PubKey{}].addConn(cf.c)
		return
	}

	var nf, ok = n.fs[cf.f]

	if ok == false {
		nf = newNodeFeed()
		n.fs[cf.f] = nf
	}

	nf.addConn(c)
	n.fs[cipher.PubKey].delConn(cf.c) // delete from idle

}

func (n *nodeFeeds) handleDelConnFeed(cf connFeed) {

	// if the cf.f is blank the this is invalid

	if cf.f == (cipher.PubKey{}) {
		return
	}

	var nf, ok = n.fs[cf.f]

	if ok == false {
		return // no such feed
	}

	nf.delConn(c)

	// if the connection does't share a feed, the
	// we put it to idle

	for pk, nf := range n.fs {

		if nf.hasConn() == true {
			return
		}

	}

	n.fs[cipher.PubKey].addConn(cf.c) // add to idle

}

func (n *nodeFeeds) handleList() {

	if n.fl == nil && len(n.fl) > 1 {

		var fl = make([]cipher.PubKey, 0, len(n.fl))

		for pk := range n.fs {

			if pk == (cipher.PubKey) {
				continue // special case for idle connections
			}

			fl = append(fl, pk)
			n.fl = fl
		}

	}

	select {
	case n.listq <- n.fl:
	case <-n.closeq:
	}
	return

}

// get lsit of feeds
func (n *nodeFeeds) list() (list []cipher.PubKey) {

	select {
	case n.listrq <- struct{}{}:
	case <-n.closeq:
		return
	}

	select {
	case list <- n.listq:
	case <-n.closeq:
	}

	return
}

func (n *nodeFeeds) handleReceivedRoot(cr connRoot) {

	var nf, ok = n.fr[cr.r.Pub]

	if ok == false {
		return // no such feed (drop the Root)
	}

	nf.receivedRoot(cr)

}

// feed of the Node
type nodeFeed struct {
	cs map[*Conn]struct{}   // connections of the feed
	hs map[uint64]*nodeHead // heads of the feed
}

func newNodeFeed() (n *nodeFeed) {

	n = new(nodeFeed)
	n.cs = make(map[*Conn]struct{})
	n.hs = make(map[uint64]*nodeHead)

	return
}

// addConn used to add a connection that share the feed
// but we don't know about Root obejcts remote peer knows
// about and the connection can't be used to fill a Root
// object, but it can be filled to send new Root objects
func (n *nodeFeed) addConn(c *Conn) {
	n.cs[c] = struct{}{}
}

func (n *nodeFeed) delConn(c *Conn) {
	delete(n.cs, c)

	// TOOD (kostyarin): delete from fillers

}

func (n *nodeFeed) hasConn(c *Conn) (ok bool) {
	_, ok = n.cs[c]
	return
}

// a head
type nodeHead struct {
	r *registry.Root // filling Root
	p *registry.Root // pending to be filled

	f  *skyobject.Filler  // filler of the r
	rq chan cipher.SHA256 // request objects from a peer (used by the f)

	// connection -> [] known Root obejcts
	//
	// the known is seq numbers of Root objects remote peer
	// knows about (sending the Root obejcts to us); we drop
	// all seq of Root objects older than currently filling
	// Root, because who cares; using the list of know Root
	// obejcts we can request Root obejcts from a peer;
	// if peer can't send a requested obejct, then we remove
	// seq of known object (making it "unknown"); a remote peer
	// can remove a Root object and all related obejcts, in
	// this case we can't use the peer to fill a Root (thus,
	// we make it "unknown")
	//

	cs map[*Conn][]uint64 // connections used to fill the r

	addc chan *Conn // add connection
	rmc  chan *Conn // remove connection

	fr chan *registry.Root // received roots to be filled up

	closeq chan struct{} // terminate
}

func (n *nodeHead) receivedRoot(cr connRoot) {
	//
}
