package skyobject

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

// An Incs used to keep all changes of
// references counter of object of filling
// Roots. The Incs used by node package
// to know what objects in DB belongs to
// full Root obejcts, and what to filling
// (and canbe removed if filling fails)
type Incs struct {
	mx   sync.Mutex
	incs map[cipher.SHA256]map[interface{}]int
}

//
// Developer note about the Incs
//
//
// if rc is greater then 1, then this obejct used
// by another Root and probably has all its childs,
// and in this case we should not go deepper to subtree,
// but the object can be an object of a filling Root
// that is not full yet and fails later; thus we have
// to check all objets to be sure that subtree of
// this objects present in DB; so, that is slow,
// a very slow, for big objects; and the rc is not
// a guarantee
// ---
// a way to avoid this unnecessary walking
// ---
// may be there is a better way, but for now we are
// using provided Incs, that used by fillers; thus
// we can get 'real rc'
//
//     'real rc' = rc - (all inc)
//
// and if the 'real rc' is greater the 1, the we
// can skip walking deepper, because object with
// it's subtree in DB and it's guarantee
//
// so, but the looking uses mutex and can slow down
// the filling, but we can avoid many disk acesses
//

// Inc increment given key
func (i *Incs) Inc(p interface{}, key cipher.SHA256) {
	i.mx.Lock()
	defer i.mx.Unlock()

	var im, ok = i.incs[key]

	if ok == false {
		im = make(map[interface{}]int)
		i.incs[key] = im
	}

	im[p]++
}

// Remove from DB (decrement) all objects related
// to given interface
func (i *Incs) Remove(c *Container, p interface{}) (err error) {
	i.mx.Lock()
	defer i.mx.Unlock()

	for k, im := range i.incs {

		var inc, ok = im[p]

		if ok == false {
			continue
		}

		if _, err = c.Inc(k, -inc); err != nil {
			return
		}

		delete(im, p)

		if len(im) == 0 {
			delete(i.incs, k)
		}

	}

	return

}

// Save removes all objects related to
// given interface from the Incs
func (i *Incs) Save(p interface{}) {
	i.mx.Lock()
	defer i.mx.Unlock()

	for k, im := range i.incs {

		delete(im, p)

		if len(im) == 0 {
			delete(i.incs, k)
		}

	}
}

// Incs is rc of filling Roots of the obejct
func (i *Incs) Incs(key cipher.SHA256) (incs int) {
	i.mx.Lock()
	defer i.mx.Unlock()

	for _, inc := range i.incs[key] {
		incs += inc
	}

	return
}
