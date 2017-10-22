package registry

import (
	"github.com/skycoin/skycoin/src/cipher"
)

type refsIndex map[cipher.SHA256][]*refsNode

// add element to the index
func (r refsIndex) addElementToIndex(hash cipher.SHA256, re *refsElement) {
	r[hash] = append(r[hash], re)
}

// delete all elements from the index by given hash
func (r refsIndex) delFromIndex(hash cipher.SHA256) (res []*refsElement) {
	res = r[hash]
	delete(r, hash)
	return
}

// delete a particular element from the index
func (r refsIndex) delElementFromIndex(hash cipher.SHA256, re *refsElement) {

	res = r[hash]

	for k, node := range res {
		if node == rn {

			copy(res[k:], res[k+1:]) // delete
			res[len(res)-1] = nil    // from
			res = res[:len(res)-1]   // the array

			break
		}
	}

	if len(res) == 0 {
		delete(r, hash)
		return
	}

	r[hash] = res
}
