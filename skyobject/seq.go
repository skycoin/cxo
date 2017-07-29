package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"sync"
)

// track seq number per root (feed)
type trackSeq struct {
	mx sync.Mutex

	// feed -> last
	track map[cipher.PubKey]uint64
}

func (t *trackSeq) init() {
	t.track = make(map[cipher.PubKey]uint64)
}

// addFeed or repalce last seq of existing one
func (t *trackSeq) addFeed(pk cipher.PubKey, seq uint64) {
	t.mx.Lock()
	defer t.mx.Unlock()

	t.track[pk] = seq
}

// take seq number of a feed, incrementing it after; method
// creates feed returning zero if no such feed
func (t *trackSeq) take(pk cipher.PubKey) (seq uint64) {
	t.mx.Lock()
	defer t.mx.Unlock()

	seq = t.track[pk]
	t.track[pk] = seq + 1

	return
}
