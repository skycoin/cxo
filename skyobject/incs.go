package registry

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

// An Incer represents represents
// skyobejct.Container (only Inc)
// method used
type Incer interface {
	Inc(key cipher.SHA256, inc int) (rc uint32, err error)
}

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
func (i *Incs) Remove(c Incer, p interface{}) (err error) {
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
