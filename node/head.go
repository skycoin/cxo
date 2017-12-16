package node

import (
	"container/list"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

// a head
type nodeHead struct {
	n *nodeFeed // back reference

	delcq chan *Conn    // delete connection
	rrq   chan connRoot // received roots

	// closing
	await  sync.WaitGroup // wait goroutines
	clsoeo sync.Once      // close once
	closeq chan struct{}  // terminate
}

func newNodeHead(nf *nodeFeed) (n *nodeHead) {

	n = new(nodeHead)

	n.n = nf

	n.rq = make(chan cipher.SHA256)
	n.cs = make(map[*Conn][]uint64)

	n.delcq = make(chan *Conn)
	n.rrq = make(chan connRoot)

	n.closeq = make(chan struct{})

	n.await.Add(1)
	go n.handle()

	return
}

// (api)
func (n *nodeHead) delConn(c *Conn) {

	select {
	case <-n.closeq:
	case n.delcq <- c:
	}

}

// (api)
func (n *nodeHead) receivedRoot(cr connRoot) {

	select {
	case <-n.closeq:
	case n.rrq <- cr:
	}

}

// (api)
func (n *nodeHead) close() {
	n.clsoeo.Do(func() {
		close(n.closeq)
	})
	n.await.Wait()
}

func (n *nodeHead) terminate() {

	// todo

}

type failedRequest struct {
	c   *Conn         // connection
	seq uint64        // seq of the filling Root
	key cipher.SHA256 // requested object
	err error         // failed if the err is not nil
}

// handle local "fields" of the nodeHead
type fillHead struct {
	*nodeHead

	r  *registry.Root     // filling Root
	f  *skyobject.Filler  // filler of the r
	rq chan cipher.SHA256 // request objects (TODO: maxParall)

	p *registry.Root // waits to be filled

	cs knownRoots // conn -> known root objects (seq)

	successq chan *Conn         // succeeded requests
	failureq chan failedRequest // failed requests

	rqo *list.List // request objects (cipher.SHA256)
	fc  *list.List // conenction to fill from (*Conn)

	requesting int // number of running requests
}

func (n *nodeHead) handle() {

	defer n.await.Done()
	defer n.terminate()

	var (
		delcq  = n.delcq  //
		rrq    = n.rrq    //
		closeq = n.closeq //

		f = fillHead{
			nodeHead: n,
			rq:       make(chan cipher.SHA256, 10),
			cs:       make(knownRoots),
		}

		key cipher.SHA256
		c   *Conn
		cr  connRoot
		fc  fillConn
		ok  bool
	)

	for {
		select {
		case key = <-f.rq:
			//
		case fc = <-f.success:
			//
		case fc = <-f.fail:
			// failed
		case cr = <-rrq: // root received
			//
		case c = <-delcq: // delete connection
			//
		case <-closeq: // terminate
			return
		}
	}

}

func (f *fillHead) handleRequest(key cipher.SHA256) (ok bool) {
	f.rqo = append(f.rqo, key)

	if len(f.fc) == 0 { // no conenctions to request from
		ok = (f.requesting != 0) // wait conenctions (or terminate)
		return
	}

	var c = f.fc[0]

	copy(f.fc, f.fc[1:])
	f.fc[len(f.fc)]

}

type knownRoots map[*Conn][]uint64

func (k knownRoots) addKnown(c *Conn, seq uint64) {

	var known, ok = k[c]

	if ok == false {
		k[c] = []uint64{c}
		return
	}

	for i, ks := range known {

		// already have
		if ks == seq {
			return
		}

		// middle
		if ks > seq {
			known = append(known[:i], append([]uint64{seq}, known[i:]...)...)
			k[c] = known
			return
		}

	}

	// tail
	known = append(known, seq)
	k[c] = known

}

// remove known Root object from a connection, from which
// we can't request an object (request failure)
func (k knownRoots) removeKnown(c *Conn, seq uint64) {

	var (
		known = k[c]
		ks    uint64
		i     int
	)

	for i, ks = range known {

		if ks == seq {
			k[c] = append(known[:i], known[i+1:]...)
			return
		}

	}

}

// a Root filled, and we can rid out of old known
// Root objects of peers
func (k knownRoots) moveForward(seq uint64) {

	var (
		known = k[c]
		ks    uint64
		i     int
	)

	for c, known := range k {

		for i, ks = range known {

			if ks >= seq {
				k[c] = append(known[:i], known[i+1:]...)
				break
			}

		}

	}

}

// build list of connections to fill Root with given seq
func (k knownRoots) buildConnsList(seq uint64) (list chan Conn) {

	var pre *Conn // prepare

	for c, known := range k {

		for _, ks := range known {

			if ks == seq {
				pre = append(pre, c)
				break
			}

		}
	}

	if len(pre) == 0 {
		return // nil
	}

	for _, c := range pre {
		list <- fillConn{c: c} // add to the queue
	}

	return
}
