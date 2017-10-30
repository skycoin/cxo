package registry

import (
	"github.com/skycoin/skycoin/src/cipher"
)

type refsIndex map[cipher.SHA256][]*refsNode

// add element to the index
func (r refsIndex) addElementToIndex(re *refsElement) {
	r[hash] = append(r[re.Hash], re)
}

// delete all elements from the index by given hash
func (r refsIndex) delFromIndex(hash cipher.SHA256) (res []*refsElement) {
	res = r[hash]
	delete(r, hash)
	return
}

// delete a particular element from the index
func (r refsIndex) delElementFromIndex(re *refsElement) {

	res = r[re.Hash]

	for k, node := range res {
		if node == rn {

			copy(res[k:], res[k+1:]) // delete
			res[len(res)-1] = nil    // from
			res = res[:len(res)-1]   // the array

			break
		}
	}

	if len(res) == 0 {
		delete(r, re.Hash)
		return
	}

	r[re.Hash] = res
}
