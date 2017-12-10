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

func (s *Split) splitDynamicAsync(pack *Pack, dr *registry.Dynamic) {
	defer s.await.Done()
	s.splitDynamic(pack, dr)
}

func (s *Split) splitDynamic(pack *Pack, dr *registry.Dynamic) {

	if dr.IsValid() == false {
		s.fail(registry.ErrInvalidDynamicReference)
		return
	}

	if dr.IsBlank() == true {
		return // nothing to split
	}

	var (
		sch registry.Schema
		err error
	)

	if sch, err = pack.Registry().SchemaByReference(dr.Schema); err != nil {
		s.fail(err)
		return
	}

	s.splitSchemaHash(pack, sch, dr.Hash)

}

func (s *Split) splitSchemaHash(
	pack Pack, //           : pack
	sch registry.Schema, // : schema of the object
	hash cipher.SHA256, //  : hash of the object
) {

	if hash == (cipher.SHA256{}) {
		return
	}

	var (
		rc  uint32
		val []byte
	)

	if val, rc, err = s.get(hash); err != nil {
		s.fail(err)
		return
	}

	// TODO (kostyarin): if rc is greater then 1, then this obejct used
	//                   by another Root and probably has all its childs,
	//                   and in this case we should not go deepper to subtree,
	//                   but the object can be an object of a filling Root
	//                   that is not full yet and fails later; thus we have
	//                   to check all objets to be sure that subtree of
	//                   this objects present in DB; so, that is slow,
	//                   a very slow, for big objects; and the rc is not
	//                   a guarantee
	//                   ---
	//                   a way to avoid this unnecessary walking
	//                   ---
	//                   may be there is a better way, but for now we are
	//                   using provided function, that looks fillers for
	//                   objects the incremened, and using
	//
	//                       'real rc' = rc - (all inc)
	//
	//                   and if the 'real rc' is greater the 1, the we
	//                   can skip walking deepper, because object with
	//                   it's subtree in DB and it's guarantee
	//
	//                   so, but the looking uses many mutexes and can slow
	//                   down the filling, but we can avoid many disk acesses

	if rc > 1 && rc-uint32(s.incs.Incs(dr.Hash)) > 0 {

		// subtree of the obejct in DB, because
		// it used by another Root that full

		return
	}

	// go deepper

	s.splitSchemaData(pack, sch, val)

}

func (s *Split) splitSchemaDataAsync(
	pack Pack, //           :
	sch registry.Schema, // :
	val []byte, //          :
) {
	defer s.await.Done()
	s.splitSchemaData(pack, sch, val)
}

func (s *Split) splitSchemaData(
	pack Pack, //           : pack
	sch registry.Schema, // : schema of the object
	val []byte, //          : encoded object
) {

	if sch.HasReferences() == false {
		return // no references, no walking
	}

	// the object represents Ref, Refs or Dynamic
	if sch.IsReference() == true {
		s.splitSchemaReference(pack, sch, val)
		return
	}

	switch sch.Kind() {
	case reflect.Array:
		s.splitArray(pack, sch, val)
	case reflect.Slice:
		s.splitSlice(pack, sch, val)
	case reflect.Struct:
		s.splitStruct(pack, sch, val)
	default:
		s.fail(fmt.Errorf("invalid Schema to walk through: %s", sch))
	}

}

func (s *Split) splitSchemaReference(
	pack Pack, //         : pack to get
	sch Schema, //        : schema of the object
	val []byte, //        : encoded reference
	walkFunc WalkFunc, // : the function
) {

	var err error

	switch rt := sch.ReferenceType(); rt {

	case ReferenceTypeSingle: // Ref

		var el Schema
		if el = sch.Elem(); el == nil {
			s.fail(fmt.Errorf("sSchema of Ref with nil element: %s", sch))
			return
		}

		var ref registry.Ref
		if err = encoder.DeserializeRaw(val, &ref); err != nil {
			s.fail(err)
			return
		}

		s.splitSchemaHash(pack, el, ref.Hash)

	case ReferenceTypeSlice: // Refs

		var el Schema
		if el = sch.Elem(); el == nil {
			s.fail(fmt.Errorf("Schema of Ref with nil element: %s", sch))
			return
		}

		var refs Refs
		if err = encoder.DeserializeRaw(val, &refs); err != nil {
			s.fail(err)
			return
		}

		s.splitSchemaRefs(pack, el, &refs)

	case ReferenceTypeDynamic: // Dynamic

		var dr Dynamic
		if err = encoder.DeserializeRaw(val, &dr); err != nil {
			s.fail(err)
			return
		}

		s.splitDynamic(pack, &dr)

	default:

		s.fail(fmt.Errorf("invalid ReferenceType %d to walk through", rt))

	}

}

func (s *Split) splitArray(
	pack Pack, //           : pack to get
	sch registry.Schema, // : schema of the array
	val []byte, //          : encoded array
) {

	var el Schema // Schema of the element
	if el = sch.Elem(); el != nil {
		s.fail(fmt.Errorf("Schema of element of array %q is nil", sch))
		return
	}

	s.splitArraySlice(pack, el, sch.Len(), val)

}

func (s *Split) splitSlice(
	pack Pack, //           : pack to get
	sch registry.Schema, // : schema of the slice
	val []byte, //          : encoded slice
) {

	var ln int // length of the slice
	if ln, err = getLength(val); err != nil {
		s.fail(err)
		return
	}

	var el Schema // Schema of the element
	if el = sch.Elem(); el != nil {
		s.fail(fmt.Errorf("Schema of element of slice %q is nil", sch))
		return
	}

	s.splitArraySlice(pack, el, ln, val[4:], walkFunc)
}

func (s *Split) splitArraySlice(
	pack Pack, //          : pack to get
	el registry.Schema, // : shcema of an element
	ln int, //             : length of the array or slice (> 0)
	val []byte, //         : encoded array or slice starting from first element
) {

	// doesn't need to walk through the zero-length
	// array or slice even if it contains references

	if ln == 0 {
		return
	}

	var (
		shift, m int
		err      error
	)

	for i := 0; i < ln; i++ {

		if shift > len(val) {
			err = fmt.Errorf("unexpected end of encoded  array or slice "+
				"of <%s>, length: %d, index: %d", el, ln, i)
			s.fail(err)
			return
		}

		if m, err = el.Size(val[shift:]); err != nil {
			s.fail(err)
			return
		}

		// split
		s.await.Add(1)
		go s.splitSchemaDataAsync(pack, el, val[shift:shift+m])

		shift += m

	}

	return

}

func (s *Split) splitStruct(
	pack Pack, //           : pack to get
	sch registry.Schema, // : schema of the struct
	val []byte, //          : encoded struct
) {

	var (
		shift, z int
		err      error
	)

	for i, fl := range sch.Fields() {

		if shift > len(val) {
			err = fmt.Errorf("unexpected end of encoded struct <%s>, "+
				"field number: %d, field name: %q, schema of field: %s",
				sch.String(),
				i,
				fl.Name(),
				fl.Schema().String())
			s.fail(err)
			return
		}

		if z, err = fl.Schema().Size(val[shift:]); err != nil {
			s.fail(err)
			return
		}

		s.await.Add(1)
		go s.splitSchemaDataAsync(pack, fl.Schema(), val[shift:shift+z])

		shift += z

	}

}

//
// utils
//

// getLength of length prefixed values
// (like slice of string)
func getLength(p []byte) (l int, err error) {
	var u uint32
	err = encoder.DeserializeRaw(p, &u)
	l = int(u)
	return
}
