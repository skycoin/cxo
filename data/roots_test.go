package data

import (
	"testing"
)

//
// ViewRoots
//

func TestViewRoots_Feed(t *testing.T) {
	// Feed() cipher.PubKey

	//

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

	//

}

func TestUpdateRoots_Last(t *testing.T) {
	// Last() (rp *RootPack)

	//

}

func TestUpdateRoots_Get(t *testing.T) {
	// Get(seq uint64) (rp *RootPack)

	//

}

func TestUpdateRoots_Range(t *testing.T) {
	// Range(func(rp *RootPack) (err error)) error

	//

}

func TestUpdateRoots_Reverse(t *testing.T) {
	// Reverse(fn func(rp *RootPack) (err error)) error

	//

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
