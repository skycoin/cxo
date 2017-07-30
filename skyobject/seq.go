package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"sync"
)

type trackedFeed struct {
	lastSeq  uint64 // last seq
	lastFull uint64 // seq of last full root
	hasFull  bool   // really has last full root (if lastFull is 0)
}

// track seq number per root (feed)
type trackSeq struct {
	mx sync.Mutex

	// feed -> last
	track map[cipher.PubKey]trackedFeed
}

func (t *trackSeq) init() {
	t.track = make(map[cipher.PubKey]trackedFeed)
}

// set seq of a feed
func (t *trackSeq) setSeq(pk cipher.PubKey, seq uint64) {
	t.mx.Lock()
	defer t.mx.Unlock()

	tf := t.track[pk]

	tf.lastSeq = seq
	t.track[pk] = tf
}

// get seq and increment
func (t *trackSeq) takeSeq(pk cipher.PubKey) (seq uint64) {
	t.mx.Lock()
	defer t.mx.Unlock()

	tf := t.track[pk]
	seq = tf.lastSeq

	tf.lastSeq++
	t.track[pk] = tf

	return
}

// set seq of last full of a feed
func (t *trackSeq) setFull(pk cipher.PubKey, seq uint64, ok bool) {
	t.mx.Lock()
	defer t.mx.Unlock()

	tf := t.track[pk]

	tf.lastFull = seq
	tf.hasFull = ok
	t.track[pk] = tf
}

// seq of last full root
func (t *trackSeq) getFull(pk cipher.PubKey) (seq uint64, ok bool) {
	t.mx.Lock()
	defer t.mx.Unlock()

	tf := t.track[pk]
	seq = tf.lastFull
	ok = tf.hasFull
	return
}
