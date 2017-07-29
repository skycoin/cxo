package skyobject

import (
	"reflect"
)

// refNode
type walkNode struct {
	value interface{} // golang value or Value (skyobject.Value)

	place reflect.Value // place of this (addressable)

	// TOTH (kostyarin): can we use "upper" and "unsaved fields"
	//                   to speed up packing; may be we can
	//                   use only "unsaved"?

	// upper   *walkNode // upper node

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

// func (w *walkNode) unsave() {
// 	// using non-recursive algorithm
// 	for up, i := w, 0; up != nil; up, i = up.upper, i+1 {
// 		if i > 0 && up.unsaved == true {
// 			break
// 		}
// 		up.unsaved = true
// 	}
// }
