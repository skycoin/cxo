package node

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

// A filler represents filler of Root objects.
// It is collector of skyobject.Filler, that
// requests CX objects. By design, the filler
// per connection requeired.
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

// add received data. It returns database
// saving error (that is fatal)
func (f *filler) add(data []byte) (err error) {
	hash := cipher.SumSHA256(data)

	if rs, ok := f.requests[hash]; ok {

		if err = f.c.Set(hash, data); err != nil {
			return
		}

		o := cxo{hash, data}
		for _, r := range rs {
			r <- o // wake up
		}
		delete(f.requests, hash)
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
		f.fillers[r.Hash] = f.c.NewFiller(r, f.wantq, f.drop, f.full, &f.wg)
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
