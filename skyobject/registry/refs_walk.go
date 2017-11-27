package registry

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// A WalkFunc used to walk through Refs, Ref, Root or
// Dynamic, decoding underlying values and walking from
// them too. The WalkFunc walks through all hashes including
// internal hashes of a Refs that does not represent a
// end-user provided value but used for internals instead
//
// Any time the WalkFunc can return ErrStopIteration to
// stop walking. And this error will not bubble from
// caller. But any other error returned from the WalkFunc
// will be returned from the caller
//
// The deepper erply used to walk deepper. Thus, giving
// a hash the WalkFunc can choose, want it go deepper
// (decode value  and explore its fields for example)
// or not
//
// The depth argument is zero if the WalkFunc got hash
// that points to non-internal value. E.g. if the depth is
// zero, then hash represents some element provided by
// end-user. Otherwise, the depth is level of Refs tree
// (in the Refs tree root is above all)
//
// The hash, with which the WalkFunc called, can be not
// found in DB, e.g. the caller calls the WalkFunc before
// it requests the value. And the caller request the value
// only if deepper is not false (and hash is not blank)
//
// A caller ignores the deepper reply if going deepper is
// not possible
type WalkFunc func(hash cipher.SHA256, depth int) (deepper bool, err error)

// walk if the Refs is blank
func (r *Refs) walkBlank(walkFunc WalkFunc) (err error) {
	// the depth can't be zero
	if _, err = walkFunc(r.Hash, 1); err == ErrStopIteration {
		err = nil
	}
	return // done
}

// Walk is service method and used for filling, updating and similar.
// The Walk walks through the Refs tree invoking given function for every
// node (e.g. for elements and nodes) starting from root of the tree.
// The depth argument starts from depth of the Refs and reduces to zero
// step by step, wlking down from root to leafs. If the depth is zero, then
// the hash is hash of element (not a node). The Walk updates hashes of the
// Refs tree (if LazyUpdating flag set, for example). If given RefsWalkFunc
// returns deeper == true, then the Walk goes deeper, otherwise, subtree will
// be skipped. It's impossible to walks deepper for elements, and in this
// case the deepper reply will be ignored. The Walk never load branches
// if they are not loaded (and the deepper is false). But the Walk initializes
// the Refs and if the Refs is not initialized and your Pack contains flags like
// HashTableIndex or EntireRefs, the initialization loads entire refs tree
// (that can cost to much for DB and memeory, and can be unnecessary).
//
// The first hash, with which the RefsWlkFunc will be called, is hash of
// the Refs (e.g. Refs.Hash). And if the hash is blank the Walk does not
// initialize the Refs. More then that if the Refs is not initialized, but
// the Refs.Hash is not blank, then after the initialization the Refs will
// be reset. This wa, end-user will not have a Refs with flags he don't want
// (becuase, the initialization saves flags inside the Refs).
//
// Feel free to use ErrStopIteration to break the Walk. Any error returned
// by the RefsWalkFunc (excepth the ErrStopIteratio) will be passed through
func (r *Refs) Walk(
	pack Pack, //             : pack to save and load
	walkFunc RefsWalkFunc, // : the function
) (
	err error, //             : an error
) {

	if walkFunc == nil {
		panic("walkFunc is nil") // for developers
	}

	var resetRequired bool

	defer func() {
		if resetRequired == true {
			if err != nil {
				r.Reset() // ignore the error (can't happen)
			} else {
				err = r.Reset()
			}
		}
	}()

	if r.Hash == (cipher.SHA256{}) {

		if r.refsNode != nil && r.mods&loadedMod != 0 {

			// the Refs is loaded (e.g. initialized) and we have to
			// check its state and update all hashes that not updated yet
			// to get actual hashes of nodes

			if err = r.walkUpdating(pack); err != nil {
				return // an error
			}

			if r.Hash == (cipher.SHA256{}) {
				// So, the Refs is initialized and updated and still blank
				return r.walkBlank(walkFunc)
			}

			// else -> the Refs already initialized, let's walk

		} else {

			// the Refs represents blank Refs and we should not initialize it;
			// the only one hash of the Refs is Refs.Hash that is blank. So
			// let's call the function
			return r.walkBlank(walkFunc)

		}

	} else if r.refsNode == nil {

		// the Refs is not initialized

		resetRequired = true // reset it after

		// and initialize
		if err = r.initialize(pack); err != nil {
			return
		}

	}

	// ok, let's walk
	var deepper bool

	// starting from Root (depth + 1 for the root)
	if deepper, err = walkFunc(r.Hash, r.depth+1); err != nil {
		if err == ErrStopIteration {
			err = nil
		}
		return
	} else if deepper == false {
		return // done
	}

	// walk from nodes
	err = r.walkNode(pack, r.refsNode, r.depth, walkFunc)

	if err == ErrStopIteration {
		err = nil
	}

	return

}

func (r *Refs) walkNode(
	pack Pack, //             : pack to load
	rn *refsNode, //          : the node
	depth int, //             : depth of the node
	walkFunc RefsWalkFunc, // : the function
) (
	err error, //             : an error
) {

	if depth == 0 {

		// the deepper ignored

		for _, leaf := range rn.leafs {
			if _, err = walkFunc(leaf.Hash, depth); err != nil {
				if err == ErrStopIteration {
					err = nil
				}
				return
			}
		}

	}

	// else if depth > 0 -> { branches }

	var deepper bool

	for _, br := range rn.branches {

		// the br contains hash, but the br can be not loaded;
		// here the walkFunc called with dept of this node, e.g.
		// the br is branch, not leaf, but the br can contians
		// leafs

		if deepper, err = walkFunc(br.hash, depth); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return
		} else if deepper == false {
			continue // let's look at the next brnahc
		}

		// ok, let's load the branch if need and walk through it

		if err = r.loadNodeIfNeed(pack, br, depth-1); err != nil {
			return
		}

		if err = r.walkNode(pack, br, depth-1, walkFunc); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return
		}

		// continue

	}

	return

}
