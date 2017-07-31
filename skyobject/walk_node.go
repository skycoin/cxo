package skyobject

import (
	"reflect"
)

type walkNode struct {
	value interface{} // golang value or Value (skyobject.Value)

	place reflect.Value // place of this (addressable)

	// upper node if this walkNode is part of Reference that
	// is part of References array (merkle-tree);
	// the upper used to track changes to allow
	// fast updating (packing, publishing) roots
	// skipping already saved branched of References
	upper *refsWalkNode

	// true if the node new or changed;
	// false if node is a copy
	unsaved bool

	sch  Schema // schema of the value
	pack *Pack  // back reference to related Pack
}

func (w *walkNode) set(i interface{}) {
	if w.place != nil {
		val := reflect.ValueOf(i)
		val = reflect.Indirect(val)
		w.place.Set(val)
	}
}

func (w *walkNode) unsave() {
	w.unsaved = true
	if up := w.upper; up != nil {
		up.unsave()
	}
}
