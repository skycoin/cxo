package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"sync"
)

const (
	inversedSeq uint64 = ^uint64(0)
)

type trackedFeed struct {
	last, full uint64
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
func (t *trackSeq) addSeq(pk cipher.PubKey, seq uint64, full bool) {
	t.mx.Lock()
	defer t.mx.Unlock()

	tf, ok := t.track[pk]
	if !ok {
		tf.last = inversedSeq
		tf.full = inversedSeq
	}

	if tf.last == inversedSeq || seq > tf.last {
		tf.last = seq
	}
	if full {
		if tf.full == inversedSeq || seq > tf.full {
			tf.full = seq
		}
	}

	t.track[pk] = tf
}

// del feed (stop tracking)
func (t *trackSeq) delFeed(pk cipher.PubKey) {
	t.mx.Lock()
	defer t.mx.Unlock()

	delete(t.track, pk)
}

// TakeLastSeq increments and returns last seq of a feed. Result
// can be 0 (despite incrementing).
func (t *trackSeq) TakeLastSeq(pk cipher.PubKey) (seq uint64) {
	t.mx.Lock()
	defer t.mx.Unlock()

	tf, ok := t.track[pk]
	if !ok {
		tf.last = inversedSeq
		tf.full = inversedSeq
	}

	tf.last++
	seq = tf.last

	t.track[pk] = tf
	return
}

func (t *trackSeq) takeLastSeqAndLock(pk cipher.PubKey) (seq uint64) {
	t.mx.Lock()

	tf, ok := t.track[pk]
	if !ok {
		tf.last = inversedSeq
		tf.full = inversedSeq
	}

	tf.last++
	seq = tf.last

	t.track[pk] = tf
	return
}

// LastSeq of a feed. The 'ok' is false if feed is not tracked
func (t *trackSeq) LastSeq(pk cipher.PubKey) (seq uint64, ok bool) {
	t.mx.Lock()
	defer t.mx.Unlock()

	tf := t.track[pk]

	if tf.last == inversedSeq {
		return
	}
	seq, ok = tf.last, true
	return
}

// LastFullSeq of a feed. The 'ok' is false if feed is not tracked
func (t *trackSeq) LastFullSeq(pk cipher.PubKey) (seq uint64, ok bool) {
	t.mx.Lock()
	defer t.mx.Unlock()

	tf := t.track[pk]

	if tf.full == inversedSeq {
		return
	}
	seq, ok = tf.full, true
	return
}
