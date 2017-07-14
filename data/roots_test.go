package data

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

//
// helper
//

func testFillWithExampleFeed(t *testing.T, pk cipher.PubKey, db DB) {
	// add feed and root
	err := db.Update(func(tx Tu) (err error) {
		feeds := tx.Feeds()
		if err = feeds.Add(pk); err != nil {
			return
		}
		roots := feeds.Roots(pk)
		for _, rp := range []RootPack{
			getRootPack(0, "hey"),
			getRootPack(1, "hoy"),
			getRootPack(2, "gde kon' moy voronoy"),
		} {
			if err = roots.Add(&rp); err != nil {
				return
			}
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
}

//
// ViewRoots
//

func testViewRootsFeed(t *testing.T, db DB) {
	pk, _ := cipher.GenerateKeyPair()

	testFillWithExampleFeed(t, pk, db)
	if t.Failed() {
		return
	}

	//

}

func TestViewRoots_Feed(t *testing.T) {
	// Feed() cipher.PubKey

	t.Run("memory", func(t *testing.T) {
		testViewRootsFeed(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewRootsFeed(t, db)
	})

}

func TestViewRoots_Last(t *testing.T) {
	// Last() (rp *RootPack)

	//

}

func TestViewRoots_Get(t *testing.T) {
	// Get(seq uint64) (rp *RootPack)

	//

}

func TestViewRoots_Range(t *testing.T) {
	// Range(func(rp *RootPack) (err error)) error

	//

}

func TestViewRoots_Reverse(t *testing.T) {
	// Reverse(fn func(rp *RootPack) (err error)) error

	//

}

//
// UpdateRoots
//

// inherited from ViewRoots

func TestUpdateRoots_Feed(t *testing.T) {
	// Feed() cipher.PubKey

	t.Skip("inherited from ViewRoots")

}

func TestUpdateRoots_Last(t *testing.T) {
	// Last() (rp *RootPack)

	t.Skip("inherited from ViewRoots")

}

func TestUpdateRoots_Get(t *testing.T) {
	// Get(seq uint64) (rp *RootPack)

	t.Skip("inherited from ViewRoots")

}

func TestUpdateRoots_Range(t *testing.T) {
	// Range(func(rp *RootPack) (err error)) error

	t.Skip("inherited from ViewRoots")

}

func TestUpdateRoots_Reverse(t *testing.T) {
	// Reverse(fn func(rp *RootPack) (err error)) error

	t.Skip("inherited from ViewRoots")

}

// UpdateRoots

func TestUpdateRoots_Add(t *testing.T) {
	// Add(rp *RootPack) (err error)

	//

}

func TestUpdateRoots_Del(t *testing.T) {
	// Del(seq uint64) (err error)

	//

}

func TestUpdateRoots_RangeDel(t *testing.T) {
	// RangeDel(fn func(rp *RootPack) (del bool, err error)) error

	//

}

func TestUpdateRoots_DelBefore(t *testing.T) {
	// DelBefore(seq uint64) (err error)

	//

}
