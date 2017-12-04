package skyobject

import (
	"errors"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject/registry"
)

type unpackItem struct {
	inc     int  // saved times
	dec     int  // used times
	created bool // created
}

// An Unpack implements registry.Pack
// and used to cahnge or cerate a Roots
type Unpack struct {
	m     map[cipher.SHA256]*unpackItem // hash -> rc
	c     *Container                    // Set method
	*Pack                               // other methods
	sk    cipher.SecKey                 // owner
}

func (u *Unpack) reset() {
	for _, ui := range u.m {
		ui.dec = 0
	}
}

// Set value
func (u *Unpack) Set(key cipher.SHA256, val []byte) (err error) {

	if len(val) > u.c.conf.MaxObjectSize {
		return &ObjectIsTooLargeError{key}
	}

	var rc uint32
	if rc, err = u.c.Set(key, val, 1); err != nil {
		return
	}

	var ui, ok = u.m[key]

	if ok == false {
		ui = new(unpackItem)
		u.m[key] = ui
	}

	ui.inc++
	ui.created = (rc == 1)

	return
}

// Add value
func (u *Unpack) Add(val []byte) (key cipher.SHA256, err error) {

	key = cipher.SumSHA256(val)
	err = u.Set(key, val) // use Set of the Unpack
	return

}

// Unpack creates Unpack using given registry. Use
// the Unapck to modify a Root object and to save
// cahnges after. For every Root new, separate
// Unpack required
func (c *Container) Unpack(
	sk cipher.SecKey,
	reg *registry.Registry,
) (
	up *Unpack,
	err error,
) {

	if reg == nil {
		err = errors.New("regsitry is nil")
	}

	if err = sk.Verify(); err != nil {
		return
	}

	up = &Unpack{
		sk:   sk,
		c:    c,
		m:    make(map[cipher.SHA256]*unpackItem),
		Pack: c.getPack(reg),
	}

	return

}

// Save cahnges of given Root updating seq number and
// timestamp of the Root. The Root should have correct
// Pub, and Nonce fields. The Seq field will be set
// to next inside the Save. The Save also set Hash and
// Prev fields of the Root, and signs the Root
func (c *Container) Save(up *Unpack, r *registry.Root) (err error) {

	// save the Root recursive

	if r.Pub == (cipher.PubKey{}) {
		return errors.New("blank Pub field of the Root")
	}

	if r.Nonce == 0 {
		return errors.New("zero Nonce field of the Root")
	}

	// check out Registry

	if rr := up.Registry().Reference(); r.Reg == (registry.RegistryRef{}) {

		r.Reg = rr // set

	} else if r.Reg != rr {

		if len(r.Refs) != 0 {
			return errors.New("can't change Registry of non-blank Root")
		}

		r.Reg = rr // set this if the Root is empty

	}

	// walk the Root first

	for _, dr := range r.Refs {

		err = dr.Walk(up, func(
			hash cipher.SHA256, // :
			_ int, //              :
			_ ...cipher.SHA256, // :
		) (
			deepper bool, //       :
			err error, //          :
		) {

			// go deepper only if the obejct was created

			var ui, ok = up.m[hash]

			if ok == false {
				// this obejct was not created, then it already
				// exists in the CXDS, and we can leave it as is
				return // false, nil
			}

			// here we reduce the ui.inc; if end-user saves an
			// obejct many times (or the obejct saved by Refs
			// modifications, for exmaple if it's hash of
			// node of the Refs), then the inc will be greater
			// then one
			//
			// ui.inc - times saved
			// ui.dec - times used
			//
			// at the end of the Save we call c.Inc(key, ui.dec - ui.inc)
			// for every value (if the difference is not zero) to make
			// values in CXDS actual

			ui.dec++ // used
			deepper = ui.created
			return

		})

		if err != nil {
			up.reset() // <= reset
			return
		}

	}

	// ok, let's save the Root

	// check out Index first (has feed)
	if c.HasFeed(r.Pub) == false {
		up.reset()
		return data.ErrNoSuchFeed
	}

	var val []byte // encoded Root

	err = c.db.IdxDB().Tx(func(fs data.Feeds) (err error) {
		var hs data.Heads
		if hs, err = fs.Heads(r.Pub); err != nil {
			return // no such feed
		}
		var roots data.Roots
		if roots, err = hs.Add(r.Nonce); err != nil {
			return
		}

		var (
			lastSeq  uint64
			lastHash cipher.SHA256
		)

		// get last
		err = roots.Descend(func(dr *data.Root) (err error) {
			lastSeq = dr.Seq
			lastHash = dr.Hash
			return data.ErrStopIteration // enough
		})

		if lastHash != (cipher.SHA256{}) {
			r.Seq = lastSeq + 1
			r.Prev = lastHash
		}

		// else -> 0 and blank

		r.Time = time.Now().UnixNano()

		// hash of the Root

		val = r.Encode()
		r.Hash = cipher.SumSHA256(val)
		r.IsFull = true

		// sign

		r.Sig = cipher.SignHash(r.Hash, up.sk)

		var dr = new(data.Root)

		dr.Seq = r.Seq
		dr.Prev = r.Prev
		dr.Hash = r.Hash
		dr.Sig = r.Sig

		return roots.Set(dr) // save

	})

	if err != nil {
		up.reset()
		return
	}

	// save the Root in CXDS

	if _, err = c.db.CXDS().Set(r.Hash, val, 1); err != nil {
		up.reset()
		return
	}

	// make rc of related obejct actual

	for key, ui := range up.m {

		var inc = ui.dec - ui.inc

		if inc == 0 {
			continue // ok
		}

		if _, err = c.db.CXDS().Inc(key, inc); err != nil {
			up.reset()
			return
		}

		// leave the CXDS broken if an error occured

	}

	return

}

/*

TODO (kostyarin): the NewRoot is just usablility trick

// NewRoot creates new blank Root object
func (c *Container) NewRoot(
	feed cipher.PubKey,
	nonce uint64,
	reg *registry.Registry,
) (
	r *registry.Root,
	up *Unpack,
	err error,
) {

	//

	return

}

*/
