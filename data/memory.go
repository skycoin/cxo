package data

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

type memDB struct {
	omx     sync.RWMutex
	objects map[cipher.SHA256][]byte

	frmx  sync.RWMutex
	feeds map[cipher.PubKey][]cipher.SHA256
	roots map[cipher.SHA256]RootPack
}

func NewMemoryDB() DB {
	return &memDB{
		objects: make(map[cipher.SHA256][]byte),
		feeds:   make(map[cipher.PubKey][]cipher.SHA256),
		roots:   make(map[cipher.SHA256]RootPack),
	}
}

func (m *memDB) Del(key cipher.SHA256) {
	m.omx.Lock()
	defer m.omx.Unlock()
	delete(m.objects, key)
}

func (m *memDB) Find(filter func(key cipher.SHA256, value []byte) bool) []byte {
	m.omx.RLock()
	defer m.omx.RUnlock()
	for k, v := range m.objects {
		if filter(k, v) {
			return v
		}
	}
	return nil
}

func (m *memDB) ForEach(fn func(k cipher.SHA256, v []byte)) {
	m.omx.RLock()
	defer m.omx.RUnlock()
	for k, v := range m.objects {
		fn(k, v)
	}
}

func (m *memDB) Get(key cipher.SHA256) (data []byte, ok bool) {
	m.omx.RLock()
	defer m.omx.RUnlock()
	data, ok = m.objects[key]
	return
}

func (m *memDB) GetAll() (all map[cipher.SHA256][]byte) {
	m.omx.RLock()
	defer m.omx.RUnlock()
	all = make(map[cipher.SHA256][]byte)
	for k, v := range m.objects {
		all[k] = v
	}
	return
}

func (m *memDB) GetSlice(keys []cipher.SHA256) (slice [][]byte) {
	m.omx.RLock()
	defer m.omx.RUnlock()
	if len(m.objects) == 0 {
		return
	}
	slice = make([][]byte, 0, len(m.objects))
	for _, k := range keys {
		slice = append(slice, m.objects[k])
	}
	return
}

func (m *memDB) IsExist(key cipher.SHA256) (ok bool) {
	m.omx.RLock()
	defer m.omx.RUnlock()
	_, ok = m.objects[key]
	return
}

func (m *memDB) Len() (ln int) {
	m.omx.RLock()
	defer m.omx.RUnlock()
	ln = len(m.objects)
	return
}

func (m *memDB) Set(key cipher.SHA256, value []byte) {
	m.omx.Lock()
	defer m.omx.Unlock()
	m.objects[key] = value
}

func (m *memDB) DelFeed(pk cipher.PubKey) {
	m.frmx.Lock()
	defer m.frmx.Unlock()
	for _, rh := range m.feeds[pk] {
		delete(m.roots, rh)
	}
	delete(m.feeds, pk)
}

func (m *memDB) AddRoot(pk cipher.PubKey, rp RootPack) (err error) {
	m.frmx.Lock()
	defer m.frmx.Unlock()
	// check hash of the rp
	if rp.Hash == (cipher.SHA256{}) {
		err = newRootError(pk, &rp, "empty hash")
		return
	}
	// roots of feeds
	var rs []cipher.SHA256 = m.feeds[pk]
	// persistence (check conflicts)
	for _, rh := range rs {
		if rh == rp.Hash {
			err = newRootError(pk, &rp, "already exists")
			return
		}
	}
	// seq and prev
	if rp.Seq == 0 {
		if len(rs) > 0 {
			err = newRootError(pk, &rp, "given root is not first of the feed")
			return
		}
		if rp.Prev != (cipher.SHA256{}) {
			err = newRootError(pk, &rp, "unexpected reference to previous Root")
			return
		}
	} else { // seq > 0
		// must have previous root
		if len(rs) == 0 {
			err = newRootError(pk, &rp, "missing previous root")
			return
		}
		if rp.Prev == (cipher.SHA256{}) {
			err = newRootError(pk, &rp, "missing reference to previous Root")
			return
		}
		// the rp must be latest of the feed
		if rs[len(rs)-1] != rp.Prev { // may be the rp is not latest
			err = newRootError(pk, &rp, "unexpected previous root of the feed")
			return
		}
		// we must have hash of previous root to check prev reference
		if p, ok := m.roots[rp.Prev]; !ok {
			err = newRootError(pk, &rp, "missing previous root object")
			return
		} else if p.Next == (cipher.SHA256{}) {
			p.Next = rp.Hash // set
		} else if p.Next != rp.Hash {
			err = newRootError(pk,
				&rp,
				"previous root has another Next reference")
			return
		}
	}
	m.feeds[pk] = append(rs, rp.Hash) // add to feed
	m.roots[rp.Hash] = rp             // store
	return
}

func (m *memDB) LastRoot(pk cipher.PubKey) (rp RootPack, ok bool) {
	m.frmx.RLock()
	defer m.frmx.RUnlock()
	rs := m.feeds[pk]
	if len(rs) == 0 {
		return
	}
	rp, ok = m.roots[rs[len(rs)-1]] // must have
	return
}

func (m *memDB) ForEachRoot(pk cipher.PubKey,
	fn func(hash cipher.SHA256, rp RootPack) (stop bool)) {

	m.frmx.RLock()
	defer m.frmx.RUnlock()
	for _, rh := range m.feeds[pk] {
		if fn(rh, m.roots[rh]) {
			return
		}
	}
	return
}

func (m *memDB) Feeds() (pks []cipher.PubKey) {
	m.frmx.RLock()
	defer m.frmx.RUnlock()
	if len(m.feeds) == 0 {
		return
	}
	pks = make([]cipher.PubKey, 0, len(m.feeds))
	for pk, rs := range m.feeds {
		if len(rs) > 0 {
			pks = append(pks, pk)
		}
	}
	return
}

func (m *memDB) GetRoot(hash cipher.SHA256) (rp RootPack, ok bool) {
	m.frmx.RLock()
	defer m.frmx.RUnlock()
	rp, ok = m.roots[hash]
	return
}

func (m *memDB) DelRoot(hash cipher.SHA256) {
	m.frmx.Lock()
	defer m.frmx.RUnlock()
	// TODO
}

func (m *memDB) Stat() (s Stat) {
	m.frmx.RLock()
	defer m.frmx.RUnlock()
	m.omx.RLock()
	defer m.omx.RUnlock()
	s.Total = len(m.objects)
	s.Roots = len(m.roots)
	s.Feeds = len(m.feeds)
	for _, rp := range m.roots {
		s.Memory += len(rp.Root) + (len(cipher.SHA256{}) * 3) + 8 +
			len(cipher.Sig{})
	}
	for _, data := range m.objects {
		s.Memory += len(data)
	}
	return
}

func (m *memDB) Close() (_ error) {
	return
}
