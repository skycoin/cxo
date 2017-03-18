package skyobject

//
// set of keys
//

// keys set
type Set map[Reference]struct{}

func (s Set) Add(k Reference) {
	s[k] = struct{}{}
}

// Want returns set of keys of missing objects. The set is empty if root is
// nil or full. The set can be incomplite.
func (r *Root) Want() (set Set, err error) {
	if r == nil {
		return // don't want anything (has no root object)
	}
	set = make(Set)
	// TODO: redo using travers functions
	return
}
