package skyobject

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject/registry"
)

// Split ... TOOD ...
//
// this is CXO internal
func (c *Container) Split(
	r *registry.Root, //         : Root to split
	rq chan<- cipher.SHA256, //  : request objects from peers
	incs *Incs, //               : incs of fillers including this
) (
	s *Split, //                : the Split
	err error, //               : malformed Root
) {

	s = new(Split)

	s.c = c
	s.r = r

	s.rq = rq

	s.incs = incs

	s.quit = make(chan struct{}) // terminate

	return
}

// A Split used to fill a Root.
// See (*Container).Split for details
type Split struct {
	c *Container     // back reference
	r *registry.Root // the Root

	pack *Pack // Pack

	rq   chan<- cipher.SHA256 // request
	incs *Incs                // had incremented

	errc chan error // errors

	await  sync.WaitGroup // wait goroutines
	quit   chan struct{}  // quit (terminate)
	closeo sync.Once      // close once (terminate)
}

func (s *Split) inc(key cipher.SHA256) {
	s.incs.Inc(s, key)
}

func (s *Split) requset(key cipher.SHA256) {
	s.rq <- key
}

func (s *Split) get(key cipher.SHA256) (val []byte, rc uint32, err error) {

	// try to get from DB first

	val, rc, err = s.c.Get(key, 1) // incrementing the rc to hold the object

	if err == nil {
		s.inc(key)
		return // got
	}

	if err != data.ErrNotFound {
		fatal("DB failure:", err) // fatality
	}

	// not found

	var want = make(chan Object, 1) // wait for the object

	// requset the object using the rq channel
	s.requset(key)

	select {
	case obj := <-want:
		val, rc = obj.Val, obj.RC
	case <-s.quit:
		err = ErrTerminated
		return
	}

	s.inc(key)
	return // got

}

// Clsoe terminates the Split walking and waits for
// goroutines the split creates
func (s *Split) Close() {
	s.closeo.Do(func() {
		close(s.quit)
		s.await.Wait()
	})
}

// Run through the Root. The method blocks
// and returns first error. If the error is
// ErrTerminated, then the Run has been
// terminated by the Close mehtod. If error
// is nil, then it's done
func (s *Split) Run() (err error) {

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
		go s.splitDynamicAsync(pack, &dr)
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
		s.Close()             // terminate
		s.incs.Remove(s.c, s) // decrement received objects
	case <-done:
		s.incs.Save(s) // remove incremented objects from the Incs
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

		if _, _, err = s.get(cipher.SHA256(s.r.Reg)); err != nil {
			return
		}

		reg, err = s.c.Registry(s.r.Reg)
	}

	return

}
