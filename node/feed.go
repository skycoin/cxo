package node

import (
	"errors"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/msg"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/registry"
)

func (n *Node) fillRoot(r *registry.Root) {
	n.await.Add(1)
	go n.fill(r)

	n.broadcastRoot(r)
}

func (n *Node) broadcastRoot(r *registry.Root) {

	//

}

// a feed represents feed of the Node,
// the feed uses mutex of the Node
type feed struct {
	// feed -> head -> filler
	fillers map[uint64]*filler

	// connections of the feed
	cs map[*Conn]struct{}
}

// fill Root
type filler struct {
	r *registry.Root // fill
	p *registry.Root // pending

	f  *skyobject.Filler // current filler
	cs []*Conn           // connections to fetch

	rq chan cipher.SHA256

	fillRoot chan *registry.Root // new Root to fill

	addc chan *Conn // add connection to the list
	delc chan *Conn // delete connection from the list

	closeq chan struct{} // terminate all fillers
}

func (n *Node) fill(r *registry.Root) (err error) {
	n.mx.Lock()
	defer n.mx.Unlock()

	var f, ok = n.fc[r.Pub]

	if ok == false {
		return // don't share the feed
	}

	var fl *filler
	if fl, ok = f.fillers[r.Nonce]; ok == false {
		fl = new(filler)

		fl.rq = make(chan cipher.SHA256, 1)

		fl.addc = make(chan *Conn, 1)
		fl.delc = make(chan *Conn, 1)

		fl.closeq = make(chan struct{})

		fl.r = r
		fl.f = n.c.Fill(r, fl.rq, 0) // TOOD (kostyarin): config max parall
	}

}
