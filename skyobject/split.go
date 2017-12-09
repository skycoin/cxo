package skyobject

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject/registry"
)

// TODO (kostyarin): fix the docs
//
// Split the Root to subtrees (and so on) to fill
// the Root calling Get emthod of provided Getter
// and obaining vlaues from DB when it's possible.
// The Split method used to fill the Root. The Split
// method used by Node package. The error means
// that the Root has erros and can't be filled (e.g.
// the Root is malformed). But, if the Getter returns
// an error, then the Split fails with this error.
// The Split method walks all possible subtrees at
// the same time and the Split. If the Getter returns
// an error, then the Split fails with this error.
// The Split method blocks until first error or the
// end of the Root. If an error occured, then the
// Split decrements rc of all obejct received by the
// Getter.
//
// Short wrods, the split calls Get method of given
// Getter for all obejct of given Root the DB does
// not have.
//
// By design, the Getter is wrapper for many
// connections to peers that have the Root and can
// share it.
//
// The Split also track DB changes, and if an obejct
// received (added to DB) other way, neither then
// requesting it using the Getter, then the Split
// use it instead of waiting the Get.
//
// The Get method of the Getter will be called from
// many goroutines, since the Split fills every
// subtree in it's own goroutine. The Getter should
// block if all underlying connections are performing
// previous request.
func (c *Container) Split(
	r *registry.Root, //        : Root to split
	rq chan<- cipher.SHA256, // : request objects from peers
) (
	s *Split, //                : the Split
	err error, //               : malformed Root
) {

	s = new(Split)

	s.c = c
	s.r = r

	s.rq = rq

	s.incs = make(map[cipher.SHA256]int) // inc of a received object

	s.quit = make(chan struct{}) // terminate

	return s
}

// A Split used to fill a Root.
// See (*Container).Split for details
type Split struct {
	c *Container     // back reference
	r *registry.Root // the Root

	pack *Pack // Pack

	rq chan cipher.SHA256 // request

	// had incremented
	mx   sync.Mutex
	incs map[cipher.SHA256]int

	errc chan error // errors

	await  sync.WaitGroup // wait goroutines
	quit   chan struct{}  // quit (terminate)
	closeo sync.Once      // close once (terminate)
}

func (s *Split) inc(key cipher.SHA256) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.rq[key]++
}

func (s *Split) requset(key cipher.SHA256) {
	s.rq <- key
}

func (s *Split) get(key cipher.SHA256) (err error) {

	// try to get from DB first

	_, _, err = s.c.Get(key, 1)

	if err == nil {
		s.inc(key)
		return // got
	}

	if err != data.ErrNotFound {
		fatal("DB failure:", err) // fatality
	}

	// not found

	var want = make(chan []byte, 1) // wait for the object

	// requset the object using the rq channel
	s.requset(key)

	select {
	case <-want:
	case <-s.quit:
		return ErrTerminated
	}

	s.inc(key)
	return // got

}

func (s *Split) close() {
	close(s.quit)
	s.await.Wait()

	s.mx.Lock()
	defer s.mx.Unlock()

	for key, inc := range s.incs {

		_, err = s.c.Inc(key, -inc)
		fatal("DB failure:", err)

	}
}

// Clsoe terminates the Split walking and waits for
// goroutines the split creates
func (s *Split) Close() {
	s.closeo.Do(s.close) // once
}

// Walk through the Root. The method blocks
// and returns first error. If the error is
// ErrTerminated, then the Walk has been
// terminated by the Close mehtod
func (s *Split) Walk() (err error) {

	// (1) don't request the Root itself, since we alredy have it

	// (2) registry first

	var reg *registry.Registry
	if reg, err = s.getRegistry(); err != nil {
		return
	}

	// (3) create pack

	var pack = s.c.getPack(reg)

	// (4) errors flow

	s.errc = make(chan error, 1)

	// (5) walk subtrees

	for _, dr := range s.r.Refs {
		s.await.Add(1)
		go s.walkDynamic(pack, dr)
	}

	// (6) fail on first error
	// (7) or wait the end

	var done = make(chan struct{})

	go func() {
		s.await.Wait()
		close(done)
	}()

	select {
	case err = <-s.errc:
		s.Close() // terminate
	case <-done:
	}

	return

}

func (s *Split) fail(err error) {
	select {
	case s.errc <- err:
	case <-s.quit:
	}
}

func (s *Split) getRegistry() (reg *registry.Registry, err error) {

	if reg, err = s.c.Registry(s.r.Reg); err != nil {

		if err != data.ErrNotFound {
			fatal("DB failure:", err) // fatality
		}

		if err = s.get(cipher.SHA256(r.Reg)); err != nil {
			return
		}

		reg, err = s.c.Registry(s.r.Reg)
	}

	return

}

func (s *Split) walkDynamic(pack *Pack, dr *registry.Dynamic) {

	//

}
