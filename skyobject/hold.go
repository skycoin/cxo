package skyobject

import (
	"fmt"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

type rootsHolder struct {
	hmx    sync.Mutex
	holded map[holdedRoot]int
}

type holdedRoot struct {
	Seq uint64
	Pub cipher.PubKey
}

// Hold Root object by feed and seq number to prevent
// removig it for this session (until the Container
// closed). It's possible to hold a Root many times
func (r *rootsHolder) Hold(pk cipher.PubKey, seq uint64) {
	r.hmx.Lock()
	defer r.hmx.Unlock()

	r.holded[holdedRoot{seq, pk}]++
}

// Unhold Root object by feed and seq number
func (r *rootsHolder) Unhold(pk cipher.PubKey, seq uint64) {
	r.hmx.Lock()
	defer r.hmx.Unlock()

	hr := holdedRoot{seq, pk}

	if i, ok := r.holded[hr]; ok {
		if i > 1 {
			r.holded[hr] = i - 1
			return
		}
		delete(r.holded, hr)
	}
}

// IsHolded check is Root object holded or not
func (r *rootsHolder) IsHolded(pk cipher.PubKey, seq uint64) (yep bool) {
	r.hmx.Lock()
	defer r.hmx.Unlock()

	return (r.holded[holdedRoot{seq, pk}] > 0)
}

// CanRemove is like IsHolded but it returns
// *HoldedRootError if Root can't be removed.
// And it returns nothing if Root is not
// holded and can be removed
func (r *rootsHolder) CanRemove(pk cipher.PubKey, seq uint64) (err error) {
	r.hmx.Lock()
	defer r.hmx.Unlock()

	if r.holded[holdedRoot{seq, pk}] > 0 {
		err = &HoldedRootError{pk, seq}
	}
	return
}

// A HoldedRootError represents error
// removing a holded Root obejct
type HoldedRootError struct {
	Pub cipher.PubKey // feed ot the holded Root
	Seq uint64        // seq of the holded Root
}

// Error implements error interface
func (h *HoldedRootError) Error() string {
	return fmt.Sprintf("can't remove holded Root {%s:%d}", h.Pub.Hex()[:7],
		h.Seq)
}
