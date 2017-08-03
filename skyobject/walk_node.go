package skyobject

import (
	"reflect"
)

type unsaver interface {
	unsave()
}

type walkNode struct {
	value interface{} // golang value or Value (skyobject.Value)

	place reflect.Value // place of this (addressable)

	// upepr node (for References trees)
	upper unsaver

	// true if the node new or changed;
	// false if node is a copy
	unsaved bool

	sch  Schema // schema of the value
	pack *Pack  // back reference to related Pack
}

func (w *walkNode) set(i interface{}) {
	if w.place.IsValid() {
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
