package skyobject

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

// some of Root dropping reasons
var (
	ErrEmptyRegsitryRef = errors.New("empty registry reference")
)

//
// TODO (kostyarin): announce
//

// A WCXO is internal
type WCXO struct {
	Key  cipher.SHA256 // hash of wanted CX object
	GotQ chan []byte   // cahnnel to sent requested CX object

	Announce []cipher.SHA256 // announce get
}

// A FillingBus is internal
type FillingBus struct {
	WantQ chan WCXO
	FullQ chan FullRoot
	DropQ chan DropRootError
}

// A Filler is internal
type Filler struct {

	// references from host

	bus FillingBus

	// own fields

	c *Container // back reference

	r   *Root     // filling Root
	reg *Registry // registry of the Root

	incrs []cipher.SHA256 // decrement on fail

	gotq chan []byte // reply

	tp time.Time

	closeq chan struct{}
	closeo sync.Once
}

// NewFiller creates new filler instance
func (c *Container) NewFiller(r *Root, bus FillingBus) (fl *Filler) {

	c.Debugln(FillVerbosePin, "NewFiller", r.Short())

	fl = new(Filler)

	fl.c = c

	fl.bus = bus

	fl.r = r
	fl.gotq = make(chan []byte, 1)

	fl.closeq = make(chan struct{})

	fl.tp = time.Now() // time point

	go fl.fill()
	return
}

// Root retursn underlying filling Root obejct
func (f *Filler) Root() *Root {
	return f.r
}

func (f *Filler) drop(err error) {
	f.c.Debugln(FillVerbosePin, "(*Filler).drop", f.r.Short(), err)

	f.bus.DropQ <- DropRootError{f.r, f.incrs, err}
}

func (f *Filler) full() {
	f.c.Debugln(FillVerbosePin, "(*Filler).full", f.r.Short())
	// don't save: delegate it for node, because the node
	// can be unsubscribed from the feed and we should
	// return proper error
	f.bus.FullQ <- FullRoot{f.r, f.incrs}
	f.c.Debugf(FillPin, "%s filled after %v, %d obejcts", f.r.Short(),
		time.Now().Sub(f.tp), len(f.incrs))
}

func (f *Filler) fill() {

	f.c.Debugln(FillVerbosePin, "(*Filler).fill", f.r.Short())

	var err error

	// first of all we should save the Root itself
	if _, err = f.c.db.CXDS().Set(f.r.Hash, f.r.Encode()); err != nil {
		f.Terminate(err)
	}
	f.incrs = append(f.incrs, f.r.Hash) // saved

	if f.r.Reg == (RegistryRef{}) {
		f.Terminate(ErrEmptyRegsitryRef)
		return
	}

	var val []byte
	var ok bool

	// registry

	if val, ok = f.get(cipher.SHA256(f.r.Reg)); !ok {
		return // drop
	}

	if f.reg, err = DecodeRegistry(val); err != nil {
		f.Terminate(err)
		return
	}

	for _, dr := range f.r.Refs {
		if err = f.fillDynamic(dr); err != nil {
			f.Terminate(err)
			return
		}
	}

	f.full()
}

// Requset given object from remove peer. If announce is not
// blank, then request them too, but don't sent them through
// the got-channel
func (f *Filler) request(key cipher.SHA256,
	announce ...cipher.SHA256) (val []byte, ok bool) {

	select {
	case f.bus.WantQ <- WCXO{key, f.gotq, announce}:
	case <-f.closeq:
		return
	}
	select {
	case val = <-f.gotq:
		f.incrs = append(f.incrs, key)
		ok = true
	case <-f.closeq:
	}
	return
}

func (f *Filler) get(key cipher.SHA256,
	announce ...cipher.SHA256) (val []byte, ok bool) {

	// we should get and increment refs count because it's filling

	var err error
	if val, _, err = f.c.DB().CXDS().GetInc(key); val != nil {
		return val, true // found
	}
	if err != data.ErrNotFound {
		f.Terminate(err) // database error
		return           // nil, false
	}
	return f.request(key, announce...)
}

// Terminate filling by given reason
func (f *Filler) Terminate(err error) {
	f.closeo.Do(func() {
		f.drop(err)
		close(f.closeq)
	})
}

func (f *Filler) fillDynamic(dr Dynamic) (err error) {

	f.c.Debugln(FillVerbosePin, "(*Filler).fillDynamic", f.r.Short(),
		dr.Short())

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

	f.c.Debugln(FillVerbosePin, "(*Filler).fillRef", f.r.Short(), ref.Hex()[:7])

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

	f.c.Debugln(FillVerbosePin, "(*Filler).fillData", f.r.Short(), sch.String())

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

	f.c.Debugln(FillVerbosePin, "(*Filler).fillDataArray", f.r.Short(),
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

	f.c.Debugln(FillVerbosePin, "(*Filler).fillDataSlice", f.r.Short(),
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

	f.c.Debugln(FillVerbosePin, "(*Filler).fillDataStruct", f.r.Short(),
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
		if s, err = fl.Schema().Size(val[shift:]); err != nil {
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

	f.c.Debugln(FillVerbosePin, "(*Filler).rangeArraySlice", f.r.Short(),
		ln, el.String())

	var shift, m int
	for i := 0; i < ln; i++ {
		if shift > len(val) {
			err = fmt.Errorf("unexpected end of encoded  array or slice "+
				"of <%s>, length: %d, index: %d", el, ln, i)
			return
		}
		if m, err = el.Size(val[shift:]); err != nil {
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

	f.c.Debugln(FillVerbosePin, "(*Filler).fillDataRefSwitch", f.r.Short(),
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

	f.c.Debugln(FillVerbosePin, "(*Filler).fillDataRef", f.r.Short(),
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

	f.c.Debugln(FillVerbosePin, "(*Filler).fillDataRefs", f.r.Short(),
		sch.String())

	var refs Refs
	if err = encoder.DeserializeRaw(val, &refs); err != nil {
		return
	}
	if refs.IsBlank() {
		return
	}

	el := sch.Elem()
	if el == nil {
		err = fmt.Errorf("[ERR] schema of Refs [%s] without element: %s",
			refs.Short(),
			sch)
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
	return f.fillRefsNode(ers.Depth, ers.Nested, el)
}

func (f *Filler) fillRefsNode(depth uint32, hs []cipher.SHA256,
	sch Schema) (err error) {

	f.c.Debugln(FillVerbosePin, "(*Filler).fillRefsNode", f.r.Short(), depth,
		len(hs), sch.String())

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
		var ern encodedRefsNode
		if err = encoder.DeserializeRaw(val, &ern); err != nil {
			return
		}
		if err = f.fillRefsNode(depth-1, ern.Nested, sch); err != nil {
			return
		}
	}

	return
}

func (f *Filler) fillDataDynamic(val []byte) (err error) {

	f.c.Debugln(FillVerbosePin, "(*Filler).fillDataDynamic")

	var dr Dynamic
	if err = encoder.DeserializeRaw(val, &dr); err != nil {
		return
	}
	return f.fillDynamic(dr)

}

// StartTime returns time while the Filler starts
func (f *Filler) StartTime() time.Time {
	return f.tp
}

// A DropRootError is internal
type DropRootError struct {
	Root  *Root           // the Root in person
	Incrs []cipher.SHA256 // obejcts to decrement
	Err   error           // reason
}

// Error implements error interface
func (d *DropRootError) Error() string {
	return fmt.Sprintf("drop Root %s:", d.Root.Short(), d.Err)
}

// A FullRoot is internal
type FullRoot struct {
	Root  *Root           // the Root in person
	Incrs []cipher.SHA256 // objects to decrement
}
