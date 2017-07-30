package skyobject

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// some of Root dropping reasons
var (
	ErrEmptyRegsitryReference = errors.New("empty registry reference")
)

// A WCXO represents wanted CX object
type WCXO struct {
	Hash cipher.SHA256 // hash of wanted CX object
	GotQ chan []byte   // cahnnel to sent requested CX object
}

// A Filler represents filling Root. The Filelr is
// not thread safe
type Filler struct {

	// references from host

	// TODO (kostyarin): organize them better way

	wantq chan<- WCXO     // reference
	fullq chan<- *Root    // reference
	dropq chan<- DropRoot // reference

	// own fields

	c *Container // back reference

	r   *Root     // filling Root
	reg *Registry // registry of the Root

	gotq chan []byte // reply

	closeq chan struct{}
	closeo sync.Once
}

func (c *Container) NewFiller(r *Root, wantq chan<- WCXO, fullq chan<- *Root,
	dropq chan<- DropRoot, wg *sync.WaitGroup) (fl *Filler) {

	fl = new(Filler)

	fl.c = c

	fl.wantq = wantq
	fl.dropq = dropq
	fl.fullq = fullq

	fl.r = r
	fl.gotq = make(chan CXO, 1)

	fl.closeq = make(chan struct{})

	wg.Add(1)
	go fl.fill(wg)

	return
}

func (f *Filler) drop(err error) {
	f.dropq <- DropRootError{f.r, err}
}

func (f *Filler) full() {
	// TODO (kostyarin): mark as full
	f.fullq <- f.r
}

func (f *Filler) fill(wg *sync.WaitGroup) {
	defer wg.Done()

	if f.r.Reg == (RegistryReference{}) {
		f.drop(ErrEmptyRegsitryReference)
		return
	}

	var val []byte
	var ok bool
	var err error

	if f.reg = f.c.Registry(f.r.Reg); f.reg == nil {
		if val, ok = f.request(hahs); !ok {
			return // drop
		}
		if f.reg, err = DecodeRegistry(val); err != nil {
			f.drop(err)
		}
		f.c.addRegistry(reg) // already saved by the request call
	}

	if len(f.r.Refs) == 0 {
		f.full()
		return
	}

	for _, dr := range f.r.Refs {
		if err = f.fillDynamic(dr); err != nil {
			f.drop(err)
			return
		}
	}

	f.full()
	return
}

func (f *Filler) request(hahs cipher.SHA256) (val []byte, ok bool) {
	select {
	case f.wantq <- WCXO{hash, f.gotq}:
	case <-f.closeq:
		return
	}
	select {
	case val = <-f.gotq:
		ok = true
	case <-f.closeq:
	}
	return
}

func (f *Filler) get(hash cipher.SHA256) (val []byte, ok bool) {
	if val = f.c.Get(hash); val != nil {
		return
	}
	val, ok = f.request(hash)
	return
}

// Close the Filler
func (f *Filler) Close() {
	f.closeo.Do(func() {
		close(f.closeq)
	})
}

func (f *Filler) fillDynamic(dr Dynamic) (err error) {
	if !dr.IsValid() {
		return ErrInvalidDynamicReference
	}
	if dr.IsBlank() {
		return
	}
	var sch Schema
	if sch, err = f.reg.SchemaByReference(dr.SchemaRef); err != nil {
		return
	}
	return f.fillReference(sch, dr.Object)
}

func (f *Filler) fillReference(sch Schema, ref cipher.SHA256) (err error) {
	if ref == (cipher.SHA256{}) {
		return // blank (represents nil)
	}
	var val []byte
	var ok bool
	if val, ok := f.get(ref); !ok {
		return
	}
	return f.fillData(sch, val)
}

func (f *Filler) fillData(sch Schema, val []byte) (err error) {

	if !sch.HasReferences() {
		return
	}

	if sch.IsReference() {
		return f.fillDataRefsSwitch(sch, val)
	}

	switch sch.Kind() {
	case reflect.Array:
		return f.fillDataArray(sch, val)
	case reflect.Slice:
		return f.fillDataSlice(sch, val)
	case reflect.Struct:
		return f.fillDataStruct(sch, val)
	}

	return fmt.Errorf("schema is not reference, array, slice or struct but"+
		"HasReferenes() retruns true: %s", sch)

}

func (f *Filler) fillDataArray(sch Schema, val []byte) (err error) {

	ln := sch.Len() // length of the array

	el := sch.Elem() // schema of element
	if el == nil {
		err = fmt.Errorf("nil schema of element of array: %s", sch)
		return
	}

	return f.rangeArraySlice(el, ln, val)

}

func (f *Filler) fillDataSlice(sch Schema, val []byte) (err error) {

	var ln int
	if ln, err = getLength(val); err != nil {
		return
	}

	el := sch.Elem() // schema of element
	if el == nil {
		err = fmt.Errorf("nil schema of element of slice: %s", sch)
		return
	}

	return f.rangeArraySlice(el, ln, val[4:])
}

func (f *Filler) fillDataStruct(sch Schema, val []byte) (err error) {

	var shift, s int

	for i, fl := range sch.Fields() {

		if shift >= len(val) {
			err = fmt.Errorf("unexpected end of encoded struct <%s>, "+
				"field number: %d, field name: %q, schema of field: %s",
				i,
				fl.Name(),
				fl.Schema())
			return
		}

		if s, err = SchemaSize(sch, val[shift:]); err != nil {
			return
		}

		if err = f.fillData(fl.Schema(), val[shift:shift+s]); err != nil {
			return
		}

		shift += s

	}

	return
}

func (f *Filler) rangeArraySlice(el Schema, ln int, val []byte) (err error) {
	var shift, m int
	for i := 0; i < ln; i++ {
		if shift >= len(val) {
			err = fmt.Errorf("unexpected end of encoded  array or slice "+
				"of <%s>, length: %d, index: %d", el, ln, i)
			return
		}
		if m, err = SchemaSize(el, val[shift:]); err != nil {
			return
		}
		if err = f.fillData(el, val[shift:shift+m]); err != nil {
			return
		}
		shift += m
	}
	return
}

func (f *Filler) fillDataRefsSwitch(sch Schema, val []byte) error {

	switch rt := sch.ReferenceType(); rt {
	case ReferenceTypeSingle:
		return f.fillDataReference(sch, val)
	case ReferenceTypeSlice:
		return f.fillDataReferences(sch, val)
	case ReferenceTypeDynamic:
		return f.fillDataDynamic(val)
	}

	return fmt.Errorf("[ERR] reference with invalid ReferenceType: %d", rt)

}

func (f *Filler) fillDataReference(sch Schema, val []byte) (err error) {

	var ref Reference
	if err = encoder.DeserializeRaw(val, &ref); err != nil {
		return
	}

	el := sch.Elem()
	if el == nil {
		err = fmt.Errorf("[ERR] schema of Reference [%s] without element: %s",
			ref.Short(),
			sch)
		return
	}

	return f.fillReference(el, ref)

}

func (f *Filler) fillDataReferences(sch Schema, val []byte) (err error) {

	var refs References
	if err = encoder.DeserializeRaw(val, &refs); err != nil {
		return
	}

	if refs.IsBlank() {
		return
	}

	for _, rn := range refs.Nodes {
		if err = f.fillRefsNode(refs.Depth, sch, rn); err != nil {
			return
		}
	}

	return

}

func (f *Filler) fillRefsNode(depth uint32, sch Schema,
	rn RefsNode) (err error) {

	if depth == 0 {
		// the leaf
		for _, hash := range rn.Hashes {
			if err = f.fillReference(sch, Reference{Hash: hash}); err != nil {
				return
			}
		}
		return
	}

	depth-- // go deepper

	for _, hash := range rn.Hashes {
		if err = f.fillHashRefsNode(depth, sch, hash); err != nil {
			return
		}
	}

	return

}

func (f *Filler) fillHashRefsNode(depth uint32, sch Schema,
	hash cipher.SHA256) (err error) {

	if hash == (cipher.SHA256{}) {
		return // empty reference
	}

	val, ok := f.get(hash)
	if !ok {
		return
	}

	var rn RefsNode
	if err = encoder.DeserializeRaw(val, &rn); err != nil {
		return
	}

	return f.fillRefsNode(depth, sch, rn)
}

func (f *Filler) fillDataDynamic(val []byte) (err error) {

	var dr Dynamic
	if err = encoder.DeserializeRaw(val, &dr); err != nil {
		return
	}

	return f.fillDynamic(dr)

}

// A DropRootError represents error
// of dropping a Root
type DropRootError struct {
	Root *Root
	Err  error
}

// Error implements error interface
func (d *DropRootError) Error() string {
	return fmt.Sprintf("drop Root %s:", d.Root.PH(), d.Err)
}
