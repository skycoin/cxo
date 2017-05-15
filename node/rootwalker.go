package node

import (
	"errors"
	"fmt"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/skycoin/src/cipher/encoder"
	"reflect"
	"strings"
)

var (
	// ErrRootNotFound happens when root is not found with public key.
	ErrRootNotFound = errors.New("root not found")

	// ErrObjNotFound happens when specified object is not found.
	ErrObjNotFound = errors.New("object not found")

	// ErrFieldNotFound happens when an object's field by name is not found.
	ErrFieldNotFound = errors.New("field not found")

	// ErrEmptyInternalStack occurs when an action performed on RootWalker
	// requires a non-empty internal stack, but internal stack is empty.
	ErrEmptyInternalStack = errors.New("internal stack of walker is empty")

	// ErrFieldHasWrongType occurs when the field in question has an unexpected
	// type.
	ErrFieldHasWrongType = errors.New("field has wrong type")
)

// RootWalker represents an object the walks a root's tree.
type RootWalker struct {
	r     *Root
	stack []*objWrap
}

// NewRootWalker creates a new walker with given container and root's public key
func NewRootWalker(r *Root) (w *RootWalker) {
	w = &RootWalker{
		r:   r,
	}
	return
}

// Size returns the size of the internal stack of walker.
func (w *RootWalker) Size() int {
	return len(w.stack)
}

// Clear clears the internal stack.
func (w *RootWalker) Clear() {
	w.stack = []*objWrap{}
}

// Helper function. Obtains top-most object from internal stack.
func (w *RootWalker) peek() (*objWrap, error) {
	if w.Size() == 0 {
		return nil, ErrEmptyInternalStack
	}
	return w.stack[w.Size()-1], nil
}

// AdvanceFromRoot advances the walker to a child object of the root.
// It uses a Finder implementation to find the child to advance to.
// This function auto-clears the internal stack.
// Input 'p' should be provided with a pointer to the object in which the
// chosen root's child should deserialize to
func (w *RootWalker) AdvanceFromRoot(p interface{},
	finder func(v *skyobject.Value) bool) error {

	// Clear the internal stack.
	w.Clear()

	// Check root.
	r := w.r
	if w.r == nil {
		return ErrRootNotFound
	}

	// Loop through direct children of root.
	for i, dRef := range r.Refs() {
		// See if it's the object needed with Finder.
		v, e := r.ValueByDynamic(dRef)
		if e != nil {
			return e
		}
		// If object is found, add to stack and return.
		if finder(v) {
			// Deserialize.
			if e := encoder.DeserializeRaw(v.Data(), p); e != nil {
				return e
			}
			obj := w.newObj(v.Schema().Reference(), p, "", i)
			w.stack = append(w.stack, obj)
			return nil
		}
	}
	return ErrObjNotFound
}

// AdvanceFromRefsField advances from a field of name 'prevFieldName' and of
// type 'skyobject.References'. It uses a Finder implementation to find the
// child to advance to. Input 'p' should be provided with a pointer to the
// object in which the chosen child object should deserialize to.
func (w *RootWalker) AdvanceFromRefsField(fieldName string, p interface{},
	finder func(v *skyobject.Value) bool) error {

	// Check root.
	r := w.r
	if w.r == nil {
		return ErrRootNotFound
	}

	// Obtain top-most object from internal stack.
	obj, e := w.peek()
	if e != nil {
		return e
	}

	// Obtain data from top-most object.
	// Obtain field's value and schema name.
	fRefs, fSchemaName, e := obj.getFieldAsReferences(fieldName)
	if e != nil {
		return e
	}

	// Get Schema of field references.
	schema, e := r.SchemaByName(fSchemaName)
	if e != nil {
		return e
	}

	// Loop through References and apply Finder.
	for i, ref := range fRefs {
		// Create dynamic reference.
		dynamic := skyobject.Dynamic{
			Object: ref,
			Schema: schema.Reference(),
		}
		// Obtain value from root.
		v, e := r.ValueByDynamic(dynamic)
		if e != nil {
			return e
		}
		// See if it's the object with Finder.
		if finder(v) {
			// Deserialize.
			if e := encoder.DeserializeRaw(v.Data(), p); e != nil {
				return e
			}
			// Add to stack.
			newObj := obj.generate(v.Schema().Reference(), p, fieldName, i)
			w.stack = append(w.stack, newObj)
			return nil
		}
	}
	return ErrObjNotFound
}

// AdvanceFromRefField advances from a field of name 'prevFieldName' and type
// 'skyobject.Reference'. No Finder is required as field is a single reference.
// Input 'p' should be provided with a pointer to the object in which the
// chosen child object should deserialize to
func (w *RootWalker) AdvanceFromRefField(fieldName string,
	p interface{}) error {

	// Check root.
	r := w.r
	if w.r == nil {
		return ErrRootNotFound
	}

	// Obtain top-most object from internal stack.
	obj, e := w.peek()
	if e != nil {
		return e
	}

	// Obtain data from top-most object.
	// Obtain field's value and schema name.
	fRef, fSchemaName, e := obj.getFieldAsReference(fieldName)
	if e != nil {
		return e
	}

	// Get Schema of field reference.
	schema, e := r.SchemaByName(fSchemaName)
	if e != nil {
		return e
	}

	// Create dynamic reference.
	dynamic := skyobject.Dynamic{
		Object: fRef,
		Schema: schema.Reference(),
	}
	// Obtain value from root.
	v, e := r.ValueByDynamic(dynamic)
	if e != nil {
		return e
	}

	// Deserialize.
	if e := encoder.DeserializeRaw(v.Data(), p); e != nil {
		return e
	}
	// Add to internal stack.
	newObj := obj.generate(v.Schema().Reference(), p, fieldName, -1)
	w.stack = append(w.stack, newObj)
	return nil
}

// AdvanceFromDynamicField advances from a field of name 'prevFieldName' and
// type 'skyobject.Dynamic'. No Finder is required as field is a single
// reference. Input 'p' should be provided with a pointer to the object in which
// the chosen child object should deserialize to
func (w *RootWalker) AdvanceFromDynamicField(fieldName string,
	p interface{}) error {

	// Check root.
	r := w.r
	if w.r == nil {
		return ErrRootNotFound
	}

	// Obtain top-most object from internal stack.
	obj, e := w.peek()
	if e != nil {
		return e
	}

	// Obtain data from top-most object.
	// Obtain field's value and schema name.
	fDyn, e := obj.getFieldAsDynamic(fieldName)
	if e != nil {
		return e
	}

	// Obtain value from root.
	v, e := r.ValueByDynamic(fDyn)
	if e != nil {
		return e
	}

	// Deserialize.
	if e := encoder.DeserializeRaw(v.Data(), p); e != nil {
		return e
	}
	// Add to internal stack.
	newObj := obj.generate(v.Schema().Reference(), p, fieldName, -1)
	w.stack = append(w.stack, newObj)
	return nil
}

// Retreat retreats one from the internal stack.
func (w *RootWalker) Retreat() {
	switch w.Size() {
	case 0:
		return
	case 1:
		w.stack = []*objWrap{}
	default:
		w.stack = w.stack[:len(w.stack)-1]
		w.stack[len(w.stack)-1].next = nil
	}
}

// RemoveCurrent removes the current object and retreats.
// TODO: Implement.
func (w *RootWalker) RemoveCurrent() error {
	return errors.New("not implemented")
}

// ReplaceCurrent replaces the current object with the reference of object pointed
// to in `p`.
// TODO: Implement.
func (w *RootWalker) ReplaceCurrent(p interface{}) error {
	return errors.New("not implemented")
}

// AppendToRefsField appends a reference to references field 'fieldName' of
// top-most object. The new reference will be generated automatically by saving
// the object which 'p' points to. This recursively replaces all the associated
// "references" of the object tree and hence, changes the root.
func (w *RootWalker) AppendToRefsField(fieldName string, p interface{}) (skyobject.Reference, error) {
	nRef := skyobject.Reference{}
	// Obtain top-most object.
	tObj, e := w.peek()
	if e != nil {
		return nRef, e
	}

	// Save new obj.
	nRef = w.r.Save(p)

	// Edit top-most object.
	tRefs, _, e := tObj.getFieldAsReferences(fieldName)
	if e != nil {
		return nRef, e
	}
	tRefs = append(tRefs, nRef)
	if e := tObj.replaceReferencesField(fieldName, tRefs); e != nil {
		return nRef, e
	}

	// Recursively save.
	_, e = tObj.save()
	return nRef, e
}

// ReplaceInRefsField replaces a reference in a field of type `skyobject.References`
// with the object that `p` points to. It uses a Finder implementation to find the
// old reference to replace.
// TODO: Implement.
func (w *RootWalker) ReplaceInRefsField(fieldName string, p interface{},
	finder func(v *skyobject.Value) bool) error {
	return errors.New("not implemented")
}

// RemoveInRefsField removes a reference in a field of type `skyobject.References`.
// It uses the Finder implementation to find the reference to remove.
// TODO: Implement.
func (w *RootWalker) RemoveInRefsField(fieldName string,
	finder func(v *skyobject.Value) bool) error {
	return errors.New("not implemented")
}

// ReplaceInRefField replaces the reference field of the top-most object with a
// new reference; one that is automatically generated when saving the object
// 'p' points to, in the container. This recursively replaces all the associated
// "references" of the object tree and hence, changes the root.
func (w *RootWalker) ReplaceInRefField(fieldName string, p interface{}) (
	skyobject.Reference, error,
) {
	nRef := skyobject.Reference{}

	// Obtain top-most object.
	tObj, e := w.peek()
	if e != nil {
		return nRef, e
	}

	// Save new obj.
	nRef = w.r.Save(p)
	if e := tObj.replaceReferenceField(fieldName, nRef); e != nil {
		return nRef, e
	}

	// Recursively save.
	_, e = tObj.save()
	return nRef, e
}

// ReplaceInDynamicField functions the same as 'ReplaceInRefField'. However, it
// replaces a dynamic reference field other than a static reference field.
func (w *RootWalker) ReplaceInDynamicField(fieldName, schemaName string,
	p interface{}) (skyobject.Dynamic, error) {
	nDyn := skyobject.Dynamic{}

	// Obtain top-most object.
	tObj, e := w.peek()
	if e != nil {
		return nDyn, e
	}

	// Save new object.
	nDyn, e = w.r.Dynamic(schemaName, p)
	if e != nil {
		return nDyn, e
	}
	if e := tObj.replaceDynamicField(fieldName, nDyn); e != nil {
		return nDyn, e
	}

	// Recursively save.
	_, e = tObj.save()
	return nDyn, e
}

// String creates a readable string that shows information of the internal stack
func (w *RootWalker) String() (out string) {
	tabs := func(n int) {
		for i := 0; i < n; i++ {
			out += "\t"
		}
	}
	out += fmt.Sprint("Root")
	size := w.Size()
	if size == 0 {
		return
	}
	out += fmt.Sprintf(".Refs[%d] ->\n", w.stack[0].prevInFieldIndex)
	for i, obj := range w.stack {
		schName := ""
		s, _ := w.r.SchemaByReference(obj.s)
		if s != nil {
			schName = s.Name()
		}

		tabs(i)
		out += fmt.Sprintf("  %s", schName)
		out += fmt.Sprintf(` = "%v"`+"\n", obj.p)

		tabs(i)
		if obj.next != nil {
			out += fmt.Sprintf("  %s", schName)
			out += fmt.Sprintf(".%s", obj.next.prevFieldName)
			if obj.next.prevInFieldIndex != -1 {
				out += fmt.Sprintf("[%d]", obj.next.prevInFieldIndex)
			}
			out += fmt.Sprint(" ->\n")
		}
	}
	return
}

/******************************************************************************
 *  TYPE: objWrap.                                                            *
 ******************************************************************************/

type objWrap struct {
	prev *objWrap
	next *objWrap

	s skyobject.SchemaReference
	p interface{}

	prevFieldName    string // Field name of prev obj used to find current.
	prevInFieldIndex int    // Index of prev obj's field's prevInFieldIndex. -1
	// if single reference (not array).

	w *RootWalker // Back reference.
}

func (w *RootWalker) newObj(s skyobject.SchemaReference, p interface{},
	fn string, i int) *objWrap {

	return &objWrap{
		s:                s,
		p:                p,
		prevFieldName:    fn,
		prevInFieldIndex: i,
		w:                w,
	}
}

func (o *objWrap) generate(s skyobject.SchemaReference, p interface{},
	fn string, i int) *objWrap {

	newO := o.w.newObj(s, p, fn, i)
	newO.prev = o
	o.next = newO
	return newO
}

func (o *objWrap) elem() reflect.Value {
	return reflect.ValueOf(o.p).Elem()
}

func (o *objWrap) getFieldAsReferences(fieldName string) (
	refs skyobject.References, schemaName string, e error,
) {
	v := o.elem()
	vt := v.Type()

	// Obtain field.
	ft, has := vt.FieldByName(fieldName)
	if has == false {
		e = ErrFieldNotFound
		return
	}
	// Check type of field.
	if ft.Type.Kind() != reflect.Slice {
		e = ErrFieldHasWrongType
		return
	}

	// Obtain schemaName from field tag.
	fStr := ft.Tag.Get("skyobject")
	schemaName = strings.TrimPrefix(fStr, "schema=")

	// Obtain field value.
	f := v.FieldByName(fieldName)
	refs = f.Interface().(skyobject.References)
	return
}

func (o *objWrap) getFieldAsReference(fieldName string) (
	ref skyobject.Reference, schemaName string, e error,
) {
	v := o.elem()
	vt := v.Type()

	// Obtain field.
	ft, has := vt.FieldByName(fieldName)
	if has == false {
		e = ErrFieldNotFound
		return
	}
	// Check type of field.
	if ft.Type.Kind() != reflect.Array {
		e = ErrFieldHasWrongType
		return
	}

	// Obtain schemaName from field tag.
	fStr := ft.Tag.Get("skyobject")
	schemaName = strings.TrimPrefix(fStr, "schema=")

	// Obtain field value.
	f := v.FieldByName(fieldName)
	ref = f.Interface().(skyobject.Reference)
	return
}

func (o *objWrap) getFieldAsDynamic(fieldName string) (
	dyn skyobject.Dynamic, e error,
) {
	v := o.elem()
	vt := v.Type()

	// Obtain field.
	ft, has := vt.FieldByName(fieldName)
	if has == false {
		e = ErrFieldNotFound
		return
	}
	// Check type of field.
	if ft.Type.Kind() != reflect.Struct {
		e = ErrFieldHasWrongType
		return
	}

	// Obtain field value.
	f := v.FieldByName(fieldName)
	dyn = f.Interface().(skyobject.Dynamic)
	return
}

func (o *objWrap) getSchema(ct *skyobject.Container) skyobject.Schema {
	s, _ := ct.CoreRegistry().SchemaByReference(o.s)
	return s
}

func (o *objWrap) replaceReferencesField(fieldName string,
	newRefs skyobject.References) (e error) {

	v := o.elem()
	vt := v.Type()

	// Obtain field.
	ft, has := vt.FieldByName(fieldName)
	if has == false {
		e = ErrFieldNotFound
		return
	}
	// Check type of field.
	if ft.Type.Kind() != reflect.Slice {
		e = ErrFieldHasWrongType
		return
	}

	v.FieldByName(fieldName).Set(reflect.ValueOf(newRefs))
	return
}

func (o *objWrap) replaceReferenceField(fieldName string,
	newRef skyobject.Reference) (e error) {

	v := o.elem()
	vt := v.Type()

	// Obtain field.
	ft, has := vt.FieldByName(fieldName)
	if has == false {
		e = ErrFieldNotFound
		return
	}
	// Check type of field.
	if ft.Type.Kind() != reflect.Array {
		e = ErrFieldHasWrongType
		return
	}

	v.FieldByName(fieldName).Set(reflect.ValueOf(newRef))
	return
}

func (o *objWrap) replaceDynamicField(fieldName string,
	newDyn skyobject.Dynamic) (e error) {

	v := o.elem()
	vt := v.Type()

	// Obtain field.
	ft, has := vt.FieldByName(fieldName)
	if has == false {
		e = ErrFieldNotFound
		return
	}
	// Check type of field.
	if ft.Type.Kind() != reflect.Struct {
		e = ErrFieldHasWrongType
		return
	}

	v.FieldByName(fieldName).Set(reflect.ValueOf(newDyn))
	return
}

func (o *objWrap) save() (skyobject.Dynamic, error) {
	// Create dynamic reference of current object.
	dyn := skyobject.Dynamic{
		Object: o.w.r.Save(o.p),
		Schema: o.s,
	}

	// If this object is the direct child of root, save to root and return.
	if o.prev == nil {
		r := o.w.r
		rDyns := r.Refs()
		rDyns[o.prevInFieldIndex] = dyn
		r.Replace(rDyns)
		return dyn, nil
	}

	// Get previous object's field type.
	v := o.prev.elem()
	vt := v.Type()

	sf, has := vt.FieldByName(o.prevFieldName)
	if has == false {
		return dyn, ErrFieldNotFound
	}

	switch sf.Type.Kind() {
	case reflect.Slice: // skyobject.References
		tRefs, _, e := o.prev.getFieldAsReferences(o.prevFieldName)
		if e != nil {
			return dyn, e
		}
		tRefs[o.prevInFieldIndex] = dyn.Object
		e = o.prev.replaceReferencesField(o.prevFieldName, tRefs)
		if e != nil {
			return dyn, e
		}
	case reflect.Array: // skyobject.Reference
		tRef, _, e := o.prev.getFieldAsReference(o.prevFieldName)
		if e != nil {
			return dyn, e
		}
		tRef = dyn.Object
		e = o.prev.replaceReferenceField(o.prevFieldName, tRef)
		if e != nil {
			return dyn, e
		}
	case reflect.Struct: // skyobject.Dynamic
		tDyn, e := o.prev.getFieldAsDynamic(o.prevFieldName)
		if e != nil {
			return dyn, e
		}
		tDyn = dyn
		e = o.prev.replaceDynamicField(o.prevFieldName, tDyn)
		if e != nil {
			return dyn, e
		}
	}

	return o.prev.save()
}
