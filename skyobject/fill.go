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
	ErrEmptyRegsitryRef = errors.New("empty registry reference")
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

	wantq chan<- WCXO          // reference
	fullq chan<- *Root         // reference
	dropq chan<- DropRootError // reference

	// own fields

	c *Container // back reference

	r   *Root     // filling Root
	reg *Registry // registry of the Root

	gotq chan []byte // reply

	closeq chan struct{}
	closeo sync.Once
}

func (c *Container) NewFiller(r *Root, wantq chan<- WCXO, fullq chan<- *Root,
	dropq chan<- DropRootError, wg *sync.WaitGroup) (fl *Filler) {

	c.Debugln(VerbosePin, "NewFiller", r.Short())

	fl = new(Filler)

	fl.c = c

	fl.wantq = wantq
	fl.dropq = dropq
	fl.fullq = fullq

	fl.r = r
	fl.gotq = make(chan []byte, 1)

	fl.closeq = make(chan struct{})

	wg.Add(1)
	go fl.fill(wg)

	return
}

func (f *Filler) drop(err error) {
	f.c.Debugln(VerbosePin, "(*Filler).drop", f.r.Short(), err)

	f.dropq <- DropRootError{f.r, err}
}

func (f *Filler) full() {
	f.c.Debugln(VerbosePin, "(*Filler).full", f.r.Short())

	if err := f.c.MarkFull(f.r); err != nil {
		// detailed error
		err = fmt.Errorf("can't mark root %s as full in DB", f.r.Short())
		f.drop(err) // can't mark as full
		return
	}
	f.fullq <- f.r
}

func (f *Filler) fill(wg *sync.WaitGroup) {

	f.c.Debugln(VerbosePin, "(*Filler).fill", f.r.Short())

	defer wg.Done()
	if f.r.Reg == (RegistryRef{}) {
		f.drop(ErrEmptyRegsitryRef)
		return
	}
	var val []byte
	var ok bool
	var err error
	if f.reg = f.c.Registry(f.r.Reg); f.reg == nil {
		if val, ok = f.request(cipher.SHA256(f.r.Reg)); !ok {
			return // drop
		}
		if f.reg, err = DecodeRegistry(val); err != nil {
			f.drop(err)
		}
		f.c.addRegistry(f.reg) // already saved by the request call
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

func (f *Filler) request(hash cipher.SHA256) (val []byte, ok bool) {
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

	f.c.Debugln(VerbosePin, "(*Filler).fillDynamic", f.r.Short(), dr.Short())

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
	return f.fillRef(sch, dr.Object)
}

func (f *Filler) fillRef(sch Schema, ref cipher.SHA256) (err error) {

	f.c.Debugln(VerbosePin, "(*Filler).fillRef", f.r.Short(), ref.Hex()[:7])

	if ref == (cipher.SHA256{}) {
		return // blank (represents nil)
	}
	var val []byte
	var ok bool
	if val, ok = f.get(ref); !ok {
		return
	}
	return f.fillData(sch, val)
}

func (f *Filler) fillData(sch Schema, val []byte) (err error) {

	f.c.Debugln(VerbosePin, "(*Filler).fillData", f.r.Short(), sch.String())

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

	f.c.Debugln(VerbosePin, "(*Filler).fillDataArray", f.r.Short(),
		sch.String())

	ln := sch.Len()  // length of the array
	el := sch.Elem() // schema of element
	if el == nil {
		err = fmt.Errorf("nil schema of element of array: %s", sch)
		return
	}
	return f.rangeArraySlice(el, ln, val)
}

func (f *Filler) fillDataSlice(sch Schema, val []byte) (err error) {

	f.c.Debugln(VerbosePin, "(*Filler).fillDataSlice", f.r.Short(),
		sch.String())

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

	f.c.Debugln(VerbosePin, "(*Filler).fillDataStruct", f.r.Short(),
		sch.String())

	var shift, s int
	for i, fl := range sch.Fields() {
		if shift > len(val) {
			err = fmt.Errorf("unexpected end of encoded struct <%s>, "+
				"field number: %d, field name: %q, schema of field: %s",
				sch.String(),
				i,
				fl.Name(),
				fl.Schema().String())
			return
		}
		if s, err = SchemaSize(fl.Schema(), val[shift:]); err != nil {
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

	f.c.Debugln(VerbosePin, "(*Filler).rangeArraySlice", f.r.Short(),
		ln, el.String())

	var shift, m int
	for i := 0; i < ln; i++ {
		if shift > len(val) {
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

	f.c.Debugln(VerbosePin, "(*Filler).fillDataRefSwitch", f.r.Short(),
		sch.String())

	switch rt := sch.ReferenceType(); rt {
	case ReferenceTypeSingle:
		return f.fillDataRef(sch, val)
	case ReferenceTypeSlice:
		return f.fillDataRefs(sch, val)
	case ReferenceTypeDynamic:
		return f.fillDataDynamic(val)
	default:
		return fmt.Errorf("[ERR] reference with invalid ReferenceType: %d", rt)
	}
}

func (f *Filler) fillDataRef(sch Schema, val []byte) (err error) {

	f.c.Debugln(VerbosePin, "(*Filler).fillDataRef", f.r.Short(),
		sch.String())

	var ref Ref
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
	return f.fillRef(el, ref.Hash)
}

func (f *Filler) fillDataRefs(sch Schema, val []byte) (err error) {

	f.c.Debugln(VerbosePin, "(*Filler).fillDataRefs", f.r.Short(),
		sch.String())

	var refs Refs
	if err = encoder.DeserializeRaw(val, &refs); err != nil {
		return
	}
	if refs.IsBlank() {
		return
	}
	var ok bool
	if val, ok = f.get(refs.Hash); !ok {
		return
	}
	var ers encodedRefs
	if err = encoder.DeserializeRaw(val, &ers); err != nil {
		return
	}
	return f.fillRefsNode(ers.Depth, ers.Nested, sch)
}

func (f *Filler) fillRefsNode(depth uint32, hs []cipher.SHA256,
	sch Schema) (err error) {

	f.c.Debugln(VerbosePin, "(*Filler).fillRefsNode", f.r.Short(),
		depth, len(hs), sch.String())

	if depth == 0 { // the leaf
		for _, hash := range hs {
			if err = f.fillRef(sch, hash); err != nil {
				return
			}
		}
		return
	}
	// the branch
	for _, hash := range hs {
		if hash == (cipher.SHA256{}) {
			continue
		}
		val, ok := f.get(hash)
		if !ok {
			return
		}
		var ers encodedRefs
		if err = encoder.DeserializeRaw(val, &ers); err != nil {
			return
		}
		if err = f.fillRefsNode(depth-1, ers.Nested, sch); err != nil {
			return
		}
	}

	return
}

func (f *Filler) fillDataDynamic(val []byte) (err error) {

	f.c.Debugln(VerbosePin, "(*Filler).fillDataDynamic")

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
	return fmt.Sprintf("drop Root %s:", d.Root.Short(), d.Err)
}
