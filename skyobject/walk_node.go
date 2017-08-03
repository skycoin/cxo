package skyobject

type unsaver interface {
	unsave()
}

type walkNode struct {
	value interface{} // golang value or Value (skyobject.Value)
	upper unsaver     // upepr node (for Refs trees)
	sch   Schema      // schema of the value
	pack  *Pack       // back reference to related Pack
}

func (w *walkNode) unsave() {
	if up := w.upper; up != nil {
		up.unsave()
	}
}

// TODO (kostyarin): track modifications
//
// type UpdateStack struct {
// 	obj []interface{}
// }
//
// func (p *Pack) UpdateStack() *UpdateStack {
// 	//
// }
