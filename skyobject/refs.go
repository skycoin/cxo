package skyobject

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/disiqueira/gotree"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

type Refs struct {
	Hash cipher.SHA256 // hash of the refsNode (not stored)

	depth  int `enc:"-"` // not stored
	degree int `enc:"-"` // not stored

	length   int     `enc:"-"`
	branches []*Refs `enc:"-"` // nodes
	leafs    []*Ref  `enc:"-"`

	upper unsaver                `enc:"-"` // upper node
	wn    *walkNode              `enc:"-"` // wn of Refs
	index map[cipher.SHA256]*Ref `enc:"-"` // index (or reference)
}

func (r *Refs) rebuildIfNeed() (err error) {
	if r.depth == 0 {
		return
	}
	if pow(r.degree, r.depth) <= r.length {
		err = r.cahngeDepth(r.depth - 1)
	}
	return
}

func (r *Refs) items() int {
	return pow(r.degree, r.depth+1)
}

func (r *Refs) unsave() {

	var ln int // new length
	var ns []cipher.SHA256

	if len(r.branches) == 0 {
		for _, h := range r.leafs {
			if h.Hash == (cipher.SHA256{}) {
				continue
			}
			ln++
			ns = append(ns, h.Hash)
			if r.index != nil {
				if r.index[h.Hash] != h {
					delete(r.index, h.Hash)
					r.index[h.Hash] = h
				}
			}
		}
	} else {
		for _, h := range r.branches {
			if h.Hash == (cipher.SHA256{}) {
				continue
			}
			ln += h.length
			ns = append(ns, h.Hash)
		}
	}

	r.length = ln

	var er encodedRefs

	er.Degree = uint32(r.degree)
	er.Depth = uint32(r.depth)
	er.Length = uint32(r.length)
	er.Nested = ns

	val := encoder.Serialize(er)
	key := cipher.SumSHA256(val)

	if key == r.Hash {
		return // no changes
	}

	r.wn.pack.set(key, val)
	r.Hash = key

	if up := r.upper; up != nil {
		up.unsave()
	}

}

// unsave each leaf
//
// TODO (kostyarin): unsave last n-th leafs
func (r *Refs) unaveAll(depth int) {
	if depth == 0 {
		r.unsave()
		return
	}
	for _, rr := range r.branches {
		rr.unaveAll(depth - 1)
	}
	return
}

func (p *Pack) getRefs(sch Schema, hash cipher.SHA256,
	place reflect.Value) (r *Refs, err error) {

	r = new(Refs)
	r.Hash = hash
	if p.flags&HashTableIndex != 0 {
		r.index = make(map[cipher.SHA256]*Ref)
	}

	if r.Hash == (cipher.SHA256{}) {
		r.depth = 0
		r.degree = p.c.conf.MerkleDegree
		r.length = 0
		r.wn = &walkNode{
			sch:  sch,
			pack: p,
		}
		return
	}

	var val []byte
	if val, err = p.get(hash); err != nil {
		r = nil
		return
	}

	var er encodedRefs
	if err = encoder.DeserializeRaw(val, &er); err != nil {
		return
	}

	r.depth = int(er.Depth)
	r.degree = int(er.Degree)
	r.length = int(er.Length)
	r.wn = &walkNode{
		sch:  sch,
		pack: p,
	}

	if len(er.Nested) == 0 {
		return
	}

	if r.depth == 0 {
		r.leafs = make([]*Ref, 0, len(er.Nested))
		for _, hash := range er.Nested {
			var ref *Ref
			if ref, err = r.getRef(hash); err != nil {
				return
			}
			r.leafs = append(r.leafs, ref)
		}
	} else {
		r.branches = make([]*Refs, 0, len(er.Nested))
		for _, hash := range er.Nested {
			var refs *Refs
			if refs, err = r.getRefs(hash); err != nil {
				return
			}
			r.branches = append(r.branches, refs)
		}
	}
	return
}

func (r *Refs) getRef(hash cipher.SHA256) (ref *Ref, err error) {
	ref = new(Ref)
	ref.Hash = hash
	ref.walkNode = &walkNode{
		sch:   r.wn.sch,
		upper: r,
		pack:  r.wn.pack,
	}
	if r.wn.pack.flags&EntireTree != 0 {
		if _, err = ref.Value(); err != nil {
			return
		}
	}
	if r.wn.pack.flags&HashTableIndex != 0 {
		r.index[hash] = ref
	}
	return
}

// TODO: load by needs (instead of entire tree laoding)
func (r *Refs) getRefs(hash cipher.SHA256) (refs *Refs, err error) {
	refs = new(Refs)
	refs.Hash = hash
	refs.wn = r.wn
	refs.index = r.index
	refs.depth = r.depth - 1
	refs.degree = r.degree
	refs.upper = r

	var val []byte
	if val, err = r.wn.pack.get(hash); err != nil {
		refs = nil
		return
	}

	var er encodedRefs
	if err = encoder.DeserializeRaw(val, &er); err != nil {
		return
	}

	refs.length = int(er.Length)

	if len(er.Nested) == 0 {
		return
	}

	if refs.depth == 0 {
		refs.leafs = make([]*Ref, 0, len(er.Nested))
		for _, hash := range er.Nested {
			var ref *Ref
			if ref, err = r.getRef(hash); err != nil {
				return
			}
			refs.leafs = append(refs.leafs, ref)
		}
		return
	}
	refs.branches = make([]*Refs, 0, len(er.Nested))
	for _, hash := range er.Nested {
		var rs *Refs
		if rs, err = r.getRefs(hash); err != nil {
			return
		}
		refs.branches = append(refs.branches, rs)
	}
	return
}

type encodedRefs struct {
	Depth  uint32
	Degree uint32
	Length uint32
	Nested []cipher.SHA256
}

// IsBlank returns true if the Refs represent nil
func (r *Refs) IsBlank() bool {
	return r.Hash == (cipher.SHA256{})
}

// Short string
func (r *Refs) Short() string {
	return r.Hash.Hex()[:7]
}

// String implements fmt.Stringer interface. The
// method returns Refs.Hash.Hex()
func (r *Refs) String() string {
	return r.Hash.Hex()
}

// Eq returns true if these Refs equal to given
func (r *Refs) Eq(x *Refs) bool {
	return r.Hash == x.Hash
}

// Schema of the Referenes. It returns nil
// if the Refs are not unpacked
func (r *Refs) Schema() Schema {
	if r.wn != nil {
		return r.wn.sch
	}
	return nil
}

// Len returns length of the Refs
func (r *Refs) Len() int {
	return r.length
}

func (r *Refs) RefByIndex(i int) (ref *Ref, err error) {
	if i < 0 {
		err = fmt.Errorf("negative index: %d", i)
		return
	}
	if i >= r.length {
		err = fmt.Errorf("index out of range: %d (len %d)", i, r.length)
		return
	}
	return r.refByIndex(0, i)
}

func (r *Refs) refByIndex(shift, i int) (ref *Ref, err error) {
	if r.depth == 0 {
		if i-shift > len(r.leafs) {
			err = errors.New("malformed tree")
			return
		}
		ref = r.leafs[i-shift]
		return
	}
	for _, b := range r.branches {
		if i < shift+b.length {
			return b.refByIndex(shift, i)
		}
		shift += b.length
	}
	return
}

func (r *Refs) DelByIndex(i int) (err error) {
	var ref *Ref
	if ref, err = r.refByIndex(0, i); err != nil {
		return
	}
	err = ref.SetValue(nil)
	return
}

func (r *Refs) RefByHash(hash cipher.SHA256) (ref *Ref, err error) {
	if r.index != nil {
		var ok bool
		if ref, ok = r.index[hash]; ok == true {
			return
		}
		err = fmt.Errorf("object [%s] not found in Refs [%s]",
			hash.Hex()[:7],
			r.Short())
		return
	}
	err = r.Range(func(_ int, r *Ref) (_ error) {
		if ref.Hash == hash {
			ref = r
			return ErrStopRange
		}
		return
	})
	return
}

// func (r *Refs) LastRefByHash(hash cipher.SHA256) (ref *Ref, err error) {
// 	//
// }

// RangeRefsFunc used to itterate over all Refs
type RangeRefsFunc func(i int, ref *Ref) (err error)

func (r *Refs) Range(rrf RangeRefsFunc) (err error) {
	if err = r.rangef(0, rrf); err == ErrStopRange {
		err = nil
	}
	return
}

func (r *Refs) rangef(shift int, rrf RangeRefsFunc) (err error) {
	if r.length == 0 {
		return
	}
	if r.depth == 0 {
		for i, ref := range r.leafs {
			if err = rrf(shift+i, ref); err != nil {
				return
			}
		}
		return
	}
	for _, rr := range r.branches {
		if err = rr.rangef(shift, rrf); err != nil {
			return
		}
		shift += rr.length
	}
	return
}

func (r *Refs) Reverse(rrf RangeRefsFunc) (err error) {
	if err = r.rangeb(r.length, rrf); err != ErrStopRange {
		err = nil
	}
	return
}

func (r *Refs) rangeb(shift int, rrf RangeRefsFunc) (err error) {
	if r.length == 0 {
		return
	}
	if r.depth == 0 {
		for k := len(r.leafs) - 1; k >= 0; k-- {
			if err = rrf(shift, r.leafs[k]); err != nil {
				return
			}
			shift--
		}
		return
	}
	for k := len(r.branches) - 1; k >= 0; k-- {
		rr := r.branches[k]
		if err = rr.rangeb(shift, rrf); err != nil {
			return
		}
		shift -= rr.length
	}
	return
}

// Append given object to the Refs. The objects must not be nils
// or nil pointers. And all objects must be of the same type
func (r *Refs) Append(objs ...interface{}) (err error) {
	if len(objs) == 0 {
		return
	}

	var sch Schema
	for i, obj := range objs {
		if obj == nil {
			err = fmt.Errorf("can't Append nil interface to Refs (index: %d)",
				i)
			return
		}
		val := reflect.ValueOf(obj)
		if val.Kind() == reflect.Ptr && val.IsNil() {
			err = fmt.Errorf("can't Append nil value of %T to Refs (index %d)",
				obj, i)
			return
		}
		if sch, err = r.wn.pack.schemaOf(obj); err != nil {
			return
		}
		// TODO (kostyarin): if the Refs has not got a Schema
		if sch != r.wn.sch {
			err = fmt.Errorf(
				"can't insert object of different type: want %s, got %s",
				r.wn.sch, sch)
			return
		}
	}

	for r.items() < len(objs)+r.length {
		if err = r.cahngeDepth(r.depth + 1); err != nil {
			return
		}
	}

	for _, obj := range objs {
		ref := Ref{
			walkNode: &walkNode{
				sch:  r.wn.sch,
				pack: r.wn.pack,
			},
		}
		if err = ref.SetValue(obj); err != nil {
			return
		}
		if err = r.insertRef(r.depth, &ref); err != nil {
			return
		}
	}

	r.unaveAll(r.depth)

	return
}

func (r *Refs) cahngeDepth(depth int) (err error) {
	nr := new(Refs)
	nr.depth = depth
	nr.degree = r.degree
	nr.wn = r.wn
	if r.wn.pack.flags&HashTableIndex != 0 {
		nr.index = make(map[cipher.SHA256]*Ref)
	}

	err = r.Range(func(_ int, ref *Ref) (_ error) {
		return nr.insertRef(depth, ref)
	})
	if err != nil {
		return
	}

	// don't unsave all here (caller should to do that)

	*r = *nr // replace
	return
}

func (r *Refs) insertRef(depth int, ref *Ref) (err error) {
	var ok bool
	if ok, err = r.tryInsertRef(depth, ref); err != nil {
		return
	}
	if !ok {
		err = errors.New("can't insert: check depth or degree")
	}
	return
}

func (r *Refs) tryInsertRef(depth int, ref *Ref) (ok bool, err error) {
	if depth == 0 {
		if len(r.leafs) == r.degree {
			return // false, nil
		}
		// rebuld the ref
		rr := new(Ref)
		rr.Hash = ref.Hash
		rr.walkNode = &walkNode{
			sch:   r.wn.sch,
			upper: r,
			pack:  r.wn.pack,
		}
		if rwn := ref.walkNode; rwn != nil {
			rr.walkNode.value = rwn.value
		}
		if r.wn.pack.flags&EntireTree != 0 {
			if _, err = rr.Value(); err != nil {
				return
			}
		}
		if r.wn.pack.flags&HashTableIndex != 0 {
			r.index[ref.Hash] = rr
		}
		r.leafs = append(r.leafs, rr)
		return true, nil
	}

	// - try insert to last branch
	if len(r.branches) > 0 {
		last := r.branches[len(r.branches)-1]
		if ok, err = last.tryInsertRef(depth-1, ref); err != nil || ok == true {
			return
		}
	}
	// - create branch if possible and insert to it
	if len(r.branches) < r.degree {
		rr := new(Refs)

		// rr.Hash --> no hash
		rr.wn = r.wn
		rr.index = r.index
		rr.depth = r.depth - 1
		rr.degree = r.degree
		rr.upper = r

		r.branches = append(r.branches, rr)
		return rr.tryInsertRef(depth-1, ref) // insert
	}
	// - return false
	return

}

// func (r *Refs) Slice(i, j int) (err error) {
// 	// TODO (kostyarin): speed up
//
// 	//
//
// 	return
// }

// Celar the Refs making them empty
func (r *Refs) Clear() {
	r.Hash = (cipher.SHA256{})
	r.depth = 0
	r.degree = r.wn.pack.c.conf.MerkleDegree
	r.length = 0
}

func pow(a, b int) (p int) {
	p = 1
	for b > 0 {
		if b&1 != 0 {
			p *= a
		}
		b >>= 1
		a *= a
	}
	return p
}

// DebugString returns string that represents
// Merkle-tree of the Refs
func (r *Refs) DebugString() string {
	var gt gotree.GTStructure
	gt.Name = "(refs) " + r.Short()
	gt.Items = r.debugItems(r.depth)
	return gotree.StringTree(gt)
}

func (r *Refs) debugItems(depth int) (its []gotree.GTStructure) {
	if depth == 0 {
		for _, ref := range r.leafs {
			its = append(its, gotree.GTStructure{
				Name: "(leaf) " + ref.Hash.Hex()[:7],
			})
		}
		return
	}
	for _, ref := range r.branches {
		its = append(its, gotree.GTStructure{
			Name:  "(branch) " + ref.Hash.Hex()[:7],
			Items: ref.debugItems(depth - 1),
		})
	}
	return
}
