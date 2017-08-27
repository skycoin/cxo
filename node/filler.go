package node

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"
)

// A filler represents filler of Root objects.
// It is collector of skyobject.Filler, that
// requests CX objects. By design, the filler
// per connection requeired.
//
// TODO: terminate by DelFeed
//
type filler struct {
	c     *skyobject.Container // access DB and registries
	wantq chan skyobject.WCXO  // request wanted CX object

	requests map[cipher.SHA256][]chan []byte // wait reply

	// must drain
	full chan *skyobject.Root
	drop chan skyobject.DropRootError // root reference with error (reason)

	// filling Roots (hash of Root -> *Filler)
	fillers map[cipher.SHA256]*skyobject.Filler

	wg sync.WaitGroup
}

func (n *Node) newFiller() (f *filler) {
	f = new(filler)
	f.c = n.so
	f.wantq = make(chan skyobject.WCXO, 10)
	f.requests = make(map[cipher.SHA256][]chan []byte)
	f.full = make(chan *skyobject.Root)
	f.drop = make(chan skyobject.DropRootError)
	f.fillers = make(map[cipher.SHA256]*skyobject.Filler)
	return
}

// full/drop
func (f *filler) del(r *skyobject.Root) {
	fr := f.fillers[r.Hash]
	fr.Close()
	delete(f.fillers, r.Hash)
}

func (f *filler) waiting(wcxo skyobject.WCXO) {
	f.requests[wcxo.Hash] = append(f.requests[wcxo.Hash], wcxo.GotQ)
}

// add received data. It returns database
// saving error (that is fatal)
func (f *filler) add(val []byte) (err error) {
	key := cipher.SumSHA256(val)
	if rs, ok := f.requests[key]; ok {
		if err = f.c.SaveObject(key, val); err != nil {
			return
		}
		for _, r := range rs {
			r <- val // wake up
		}
		delete(f.requests, key)
	}
	return
}

// fill a *skyobject.Root
func (f *filler) fill(r *skyobject.Root) {
	if _, ok := f.fillers[r.Hash]; !ok {
		// arguments is:
		// - *Root to fill in person
		// - wanted objects (chan of skyobject.WCXO)
		// - drop Root (chan of skyobject.DropRootError that is {*Root, err})
		// - a Root is full (chan of *Root)
		// - wait group
		f.fillers[r.Hash] = f.c.NewFiller(r,
			skyobject.FillingBus{f.wantq, f.full, f.drop, &f.wg})
	}
}

func (f *filler) clsoe() {
	for _, fl := range f.fillers {
		fl.Close()
	}
}

func (f *filler) wait() {
	f.wg.Wait()
	return
}
