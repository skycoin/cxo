package skyobject

import (
	"bytes"
	"sort"

	"github.com/skycoin/skycoin/src/cipher"
)

type sortedListOfHashes []cipher.SHA256

func newSortedListOfHashes(in map[cipher.SHA256]int) (sl sortedListOfHashes) {
	if len(in) == 0 {
		return
	}
	sl = make(sortedListOfHashes, 0, len(in))
	for k := range in {
		sl = append(sl, k)
	}
	sort.Sort(sl)
	return
}

// for sorting

func (s sortedListOfHashes) Len() int { return len(s) }

func (s sortedListOfHashes) Less(i, j int) bool {
	return bytes.Compare(s[i].key[:], s[j].key[:]) == -1
}

func (s sortedListOfHashes) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
