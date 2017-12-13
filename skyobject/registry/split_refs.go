package registry

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// fake pack to load Refs for the
// Split method
type fakePack struct {
	s Splitter
}

func (f *fakePack) Registry() (r *Registry) {
	return f.s.Registry()
}

func (f *fakePack) Get(key cipher.SHA256) (val []byte, err error) {
	val, _, err = f.s.Get(key)
	return
}

func (*fakePack) Set(cipher.SHA256, []byte) error {
	panic("fake method called")
}

func (*fakePack) Add([]byte) (cipher.SHA256, error) {
	panic("fake method called")
}

func (*fakePack) Degree() Degree         { return MaxDegree /* any valid */ }
func (*fakePack) SetDegree(Degree) error { panic("fake method called") }
func (*fakePack) Flags() (_ Flags)       { return }
func (*fakePack) AddFlags(Flags)         { panic("fake method called") }
func (*fakePack) ClearFlags(Flags)       { panic("fake method called") }

func (r *Refs) splitHash(s Splitter, hash cipher.SHA256) (load bool) {

	var _, rc, err = s.Get(hash)

	if err != nil {
		s.Fail(err)
		return
	}

	load = (rc == 0)
	return

}

// Split used by the node package to fill the Dynamic.
// Unlike the Walk the Split desigend for fresh Refs
// received by network the the Split never reset the
// Refs if it loads it and never updates hashes of
// the Refs if they are not actual
func (r *Refs) Split(s Splitter, el Schema) {

	s.Go(func() {

		var fp = fakePack{s} // fake Pack

		if r.Hash == (cipher.SHA256{}) {
			return // done
		}

		// first of all, check the Refs.Hash

		if r.splitHash(s, r.Hash) == false {
			return
		}

		// laod to walk
		if err := r.initialize(&fp); err != nil {
			s.Fail(err)
			return
		}

		r.splitNode(s, &fp, el, r.refsNode, r.depth)

	})

}

func (r *Refs) splitNodeAsync(
	s Splitter, //   : the Splitter
	fp *fakePack, // : fake pack to load
	sch Schema, //   : schema of elements
	rn *refsNode, // : the node
	depth int, //    : depth of the node
) {
	s.Go(func() { r.splitNode(s, fp, sch, rn, depth) })
}

func (r *Refs) splitNode(
	s Splitter, //   : the Splitter
	fp *fakePack, // : fake pack to load
	sch Schema, //   : schema of elements
	rn *refsNode, // : the node
	depth int, //    : depth of the node
) {

	if depth == 0 {

		for _, leaf := range rn.leafs {
			s.Go(func() { splitSchemaHashAsync(s, sch, leaf.Hash) })
		}

		return
	}

	// else if depth > 0 -> { branches }

	for _, br := range rn.branches {

		if r.splitHash(s, br.hash) == false {
			return
		}

		if err := r.loadNodeIfNeed(fp, br, depth-1); err != nil {
			s.Fail(err)
			return
		}

		s.Go(func() { r.splitNodeAsync(s, fp, sch, rn, depth) })
	}

}
