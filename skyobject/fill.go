package skyobject

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

const fillingCacheBoundry int = 32

// A Fillers represents collector of
// filling Roots of a connection. The
// Fillers is not thread safe
type Fillers struct {
	c *Container // back referecne

	requests map[cipher.SHA256][]chan cxo

	full chan *Root    // filled up Roots
	drop chan dropRoot // drop a Root

	fillers map[cipher.SHA256]*Filler // filling roots

	await sync.WaitGroup // all fillers
}

type cxo struct {
	hash cipher.SHA256
	data []byte
}

type dropRoot struct {
	r   *Root
	err error
}

func (c *Container) NewFillers() (fs *Fillers) {
	fs = new(Fillers)

	fs.c = c
	fs.requests = make(map[cipher.SHA256][]chan struct{})

	fs.full = make(cahn * Root)
	fs.drop = make(chan dropRoot)

	fs.fillers = make(map[cipher.SHA256]*Filler)

	return
}

// TODO: want/got api

func (f *Fillers) get(hash cipher.SHA256) (data []byte) {
	return f.c.Get(hash)
}

// Add received data
func (f *Fillers) Add(data []byte) {
	hash := cipher.SumSHA256(data)

	if rs, ok := f.requests[hash]; ok {
		//
		for _, r := range rs {
			r <- struct{}{} // trigger
		}
	}

}

// Fiil a Root
func (f *Fillers) Fill(r *Root) {
	if _, ok := f.fillers[r.Hash]; !ok {
		f.fillers[r.Hash] = f.newFiller(r)
	}
}

// Close the Fillers and all related Filler(s)
func (f *Fillers) Close() (err error) {
	for _, fl := range f.fillers {
		fl.Close()
	}
	f.await.Wait()
	return
}

// A Filler represents filling root object.
// The Filler is not thread safe and designed to be used
// inside message handling groutne of a connection.
// Use the filler following the examples below here:
//
//     filler := c.NewFilelr(r)
//
//     want, err := filler.Fill()
//     if err != nil {
//         // drop the Root, stop filling (failure)
//     }
//
//     for _, hash := ragne want {
//         // request data by the hash
//     }
//
//     if filler.IsFull() {
//         // the Root is full, stop filling (success)
//     }
//
// And while a response with data received
//
//     hash := cipher.SumSHA256(data)
//
//     for _, f := everyFillerOfConnection() {
//         if filler.Add(hash, data) {
//             // repeat Fill and IsFull steps
//         }
//     }
//
type Filler struct {
	c    *Container // related container
	Root *Root      // filling Root
	reg  *Registry  // registry of the Root

	wantq chan cipher.SHA256 //
	gotq  chan cxo           //

	closeq chan struct{}
	closeo sync.Once
}

// NewFiller creates Filler of given Root
func (c *Container) NewFiller(r *Root) (f *Filler) {
	f = new(Filler)
	f.c = c
	f.Root = r

	f.cache = make(map[cipher.SHA256][]byte)
	f.saveMap = make(map[cipher.SHA256][]byte)

	return
}

// get from cache or from database
// TODO (kostyarin): use transactions for filling step
func (f *Filler) get(hash cipher.SHA256) []byte {
	if v, ok := f.cache[hash]; ok {
		return v
	}
	return f.c.Get(hash)
}

// Add requested object. It returns true if given CX object
// has been requested by Root of the Filler
func (f *Filler) Add(hash cipher.SHA256, data []byte) (accept bool) {

	var i int
	for _, w := range f.want {
		if w == hash {
			accept = true
			continue // delete from the want slice
		}
		f.want[i] = w
		i++
	}
	f.want = f.want[:i]

	if accept {
		f.cache[hash] = data // cache it
	}

	return
}

// Fill performs next filling step or does nothing
// waiting for all requested CX objects. You must not
// modify the 'want' slice
func (f *Filler) Fill() (want []cipher.SHA256, err error) {

	if f.full {
		return // the Root is full (nothing to do)
	}

	if len(f.want) != 0 {
		return // waiting for all currently wanted CX objects
	}

	// has got all requested objects, let's go

	if len(f.stack) == 0 { // begining (if not full)

		// check out registry of the Root
		if f.reg = f.c.Registry(f.Root.Reg); f.reg == nil {

			// request registry
			f.want = append(f.want, cipher.SHA256(f.Root.Reg))
			want = f.want
			f.push(fillRegistry(f.Root.Reg))
			return

		} else {

			// check out Refs of the Root
			if err = f.fillRootRefs(); err != nil {
				return
			}

		}

	}

	// check out the stack

	for err == nil && f.full == false && len(want) == 0 {

		fn := f.pop()

		if fn == nil {
			f.full = true
			return
		}

		if want, err = fn.fill(f); len(want) != 0 {
			f.want = append(f.want, want...)
		}

	}

	if err == nil {
		if len(want) == 0 {
			err = f.saveCache()
		} else if len(f.cache) > fillingCacheBoundry {
			err = f.saveCache()
		}
	}

	return

}

// IsFull returns true if Root is full
func (f *Filler) IsFull() bool { return f.full }

func (f *Filler) push(fn fillingNode) {
	f.stack = append(f.stack, fn)
}

func (f *Filler) pop() (fn fillingNode) {
	if len(f.stack) == 0 {
		return // nil
	}

	fn = f.stack[len(f.stack)-1] // get last

	f.stack[len(f.stack)-1] = nil      // release value for GC
	f.stack = f.stack[:len(f.stack)-1] // delete last element

	return
}

func (f *Filler) saveCache() (_ error) {

	if len(f.cache) == 0 {
		return
	}

	return f.c.db.Update(func(tx data.Tu) (err error) {
		err = tx.Objects().SetMap(f.cache)
		f.cache = make(map[cipher.SHA256][]byte) // clear cache
		return
	})

}

type fillingNode interface {
	fill(f *Filler) (want []cipher.SHA256, err error)
}

type fillRegistry RegistryReference

func (fr fillRegistry) fill(f *Filler) (want []cipher.SHA256, err error) {
	val := f.get(cipher.SHA256(fr))

	if val == nil {
		panic("broken filling")
	}

	var reg *Registry
	if reg, err = DecodeRegistry(val); err != nil {
		return
	}

	if err = f.c.AddRegistry(reg); err != nil {
		return
	}

	f.reg = reg

	err = f.fillRootRefs()
	return
}

func (f *Filler) fillRootRefs() (err error) {
	if len(f.Root.Refs) == 0 {
		f.full = true
		return
	}

	err = f.fillSliceOfDynamic(f.Root.Refs)
	return
}

func (f *Filler) pushDynamic(dr Dynamic) (err error) {
	if !dr.IsValid() {
		err = ErrInvalidDynamicReference
		return
	}
	if dr.IsBlank() {
		return
	}
	f.push(fillDynamic(dr))
	return
}

func (f *Filler) pushReference(sch Schema, ref cipher.SHA256) {
	if ref == (cipher.SHA256{}) {
		return // blank
	}
	f.push(&fillReference{sch, ref})
}

func (f *Filler) pushReferences(sch Schema, refs References) {
	if refs.Len == 0 {
		return // no elements
	}
	for _, refsNode := range refs.Nodes {
		f.pushRefsNode(sch, refsNode)
	}
	return
}

func (f *Filler) pushRefsNode(sch Schema, rn RefsNode) {
	f.push(&fillRefsNode{sch, rn})
}

func (f *Filler) fillSliceOfDynamic(sl []Dynamic) (err error) {

	for _, dr := range sl {
		if err = f.pushDynamic(dr); err != nil {
			return
		}
	}

	return
}

type fillDynamic Dynamic // valid and non-blank

func (fd *fillDynamic) fill(f *Filler) (want []cipher.SHA256, err error) {
	if fd.Object.sch == nil {
		if fd.Object.sch, err = f.reg.SchemaByReference(sr); err != nil {
			return
		}
	}
	val := f.get(fd.Object.Hash)
	if val == nil {
		want = append(want, fd.Object.Hash)
		return
	}

	err = f.unpack(fd.Object.sch, val)
	return
}

type fillReference struct {
	sch Schema
	ref cipher.SHA256
}

func (f *Filler) unpackRefs(sch Schema, data []byte) (err error) {
	switch sch.ReferenceType() {
	case ReferenceTypeSingle:
		var ref Reference
		if err = encoder.DeserializeRaw(data, &ref); err != nil {
			return
		}
		f.pushReference(sch.Elem(), ref.Hash)
	case ReferenceTypeSlice:
		var refs References
		if err = encoder.DeserializeRaw(data, &refs); err != nil {
			return
		}
		f.pushReferences(sch, refs)
	case ReferenceTypeDynamic:
		var dr Dynamic
		if err = encoder.DeserializeRaw(data, &dr); err != nil {
			return
		}
		err = f.pushDynamic(dr)
	default:
		err = ErrInvalidSchema
	}
	return
}

func (f *Filler) unpackRefsNode(sch Schema, data []byte) (err error) {
	var rn RefsNode
	if err = encoder.DeserializeRaw(data, &rn); err != nil {
		return
	}
	f.pushRefsNode(sch, rn)
	return
}

func (f *Filler) unpack(sch Schema, data []byte) (err error) {

	if !sch.HasReferences() {
		return // don't need to decode value without references
	}

	if sch.IsReference() {
		return f.unpackRefs(sch, data)
	}

	switch sch.Kind() {
	case reflect.Array:
		//
	case reflect.Slice:
		//
	case reflect.Struct:
		//
	default:
		// nothing to do for flat types
	}

	return

}

type fillRefsNode struct {
	sch Schema
	rn  RefsNode
}

func (fr *fillRefsNode) fill(f *Filler) (want []cipher.SHA256, err error) {
	if fr.rn.IsLeaf {
		for _, hash := range fr.rn.Hashes {
			//
		}
		return
	}
	for _, hash := range fr.rn.Hashes {
		if val := f.get(hash); val != nil {
			if err = f.unpackRefsNode(fr.sch, val); err != nil {
				return
			}
			f.push(&fillRefsNode{sch})
			continue
		}
		want = append(want, hash)
		f.push(&waitRefsNode{fr.sch, hash})
	}
	return
}

type waitRefsNode struct {
	sch Schema
	rn  cipher.SHA256
}
