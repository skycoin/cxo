package cxds

import (
	"encoding/hex"

	"github.com/tidwall/buntdb"

	"github.com/skycoin/skycoin/src/cipher"
)

type memoryCXDS struct {
	b *buntdb.DB
}

// NewMemoryCXDS creates CXDS in memory
func NewMemoryCXDS() (ds CXDS) {
	b, err := buntdb.Open(":memory:")
	if err != nil {
		panic(err)
	}
	ds = &memoryCXDS{b}
	return
}

func (m *memoryCXDS) decodeString(s string) (got []byte) {
	var err error
	if got, err = hex.DecodeString(s); err != nil {
		panic(err)
	}
	return
}

func (m *memoryCXDS) Get(key cipher.SHA256) (val []byte, rc uint32, err error) {
	err = m.b.View(func(tx *buntdb.Tx) (err error) {
		var s string
		if s, err = tx.Get(key.Hex()); err != nil {
			if err == buntdb.ErrNotFound {
				return ErrNotFound // replace error
			}
			return
		}
		got := m.decodeString(s)
		rc, val = getRefsCount(got), got[4:]
		return
	})
	return
}

func (m *memoryCXDS) Set(key cipher.SHA256, val []byte) (rc uint32, err error) {
	err = m.b.Update(func(tx *buntdb.Tx) (err error) {
		var s, k string = "", key.Hex()
		if s, err = tx.Get(k); err != nil {
			if err == buntdb.ErrNotFound {
				rc = 1
				sval := hex.EncodeToString(append(one, val...))
				_, _, err = tx.Set(k, sval, nil)
				return
			}
			return // some error
		}
		got := m.decodeString(s)
		rc = incRefsCount(got) // increment refs count
		_, _, err = tx.Set(k, hex.EncodeToString(got), nil)
		return
	})
	return
}

func (m *memoryCXDS) Add(val []byte) (key cipher.SHA256, rc uint32, err error) {
	key = getHash(val)
	rc, err = m.Set(key, val)
	return
}

func (m *memoryCXDS) Inc(key cipher.SHA256) (rc uint32, err error) {
	err = m.b.Update(func(tx *buntdb.Tx) (err error) {
		var s, k string = "", key.Hex()
		if s, err = tx.Get(k); err != nil {
			if err == buntdb.ErrNotFound {
				return ErrNotFound // replace
			}
		}
		got := m.decodeString(s)
		rc = incRefsCount(got)
		_, _, err = tx.Set(k, hex.EncodeToString(got), nil)
		return
	})
	return
}

func (m *memoryCXDS) Dec(key cipher.SHA256) (rc uint32, err error) {
	err = m.b.Update(func(tx *buntdb.Tx) (err error) {
		var s, k string = "", key.Hex()
		if s, err = tx.Get(k); err != nil {
			if err == buntdb.ErrNotFound {
				return ErrNotFound // replace
			}
		}
		got := m.decodeString(s)
		rc = decRefsCount(got)
		if rc == 0 {
			_, err = tx.Delete(k)
			return
		}
		_, _, err = tx.Set(k, hex.EncodeToString(got), nil)
		return
	})
	return
}

// TODO (kostyarin): ordered get to speed up get, because of B+-tree index
func (m *memoryCXDS) MultiGet(keys []cipher.SHA256) (vals [][]byte, err error) {
	if len(keys) == 0 {
		return
	}
	vals = make([][]byte, len(keys))
	err = m.b.View(func(tx *buntdb.Tx) (err error) {
		var s, k string
		for i, key := range keys {
			k = key.Hex()
			if s, err = tx.Get(k); err != nil {
				if err == buntdb.ErrNotFound {
					err = nil
					continue
				}
				return // some error
			}
			vals[i] = m.decodeString(s)[4:]
		}
		return
	})
	return
}

// TODO (kostyarin): ordered add to speed up insert, because of B+-tree index
func (m *memoryCXDS) MultiAdd(vals [][]byte) (err error) {
	if len(vals) == 0 {
		return
	}
	err = m.b.Update(func(tx *buntdb.Tx) (err error) {
		var s, k string
		for _, val := range vals {
			key := getHash(val)
			k = key.Hex()

			if s, err = tx.Get(k); err != nil {
				if err == buntdb.ErrNotFound {
					sval := hex.EncodeToString(append(one, val...))
					if _, _, err = tx.Set(k, sval, nil); err != nil {
						return
					}
					continue
				}
				return // some error
			}
			got := m.decodeString(s)
			incRefsCount(got) // increment refs count
			if _, _, err = tx.Set(k, hex.EncodeToString(got), nil); err != nil {
				return
			}
		}
		return
	})
	return
}

// TODO (kostyarin): ordered get to speed up get, because of B+-tree index
func (m *memoryCXDS) MultiInc(keys []cipher.SHA256) (err error) {
	if len(keys) == 0 {
		return
	}
	err = m.b.Update(func(tx *buntdb.Tx) (err error) {
		var s, k string
		for _, key := range keys {
			k = key.Hex()
			if s, err = tx.Get(k); err != nil {
				if err == buntdb.ErrNotFound {
					err = nil
					continue
				}
				return // some error
			}
			got := m.decodeString(s)
			incRefsCount(got)
			if _, _, err = tx.Set(k, hex.EncodeToString(got), nil); err != nil {
				return
			}
		}
		return
	})
	return
}

// TODO (kostyarin): ordered get to speed up get, because of B+-tree index
func (m *memoryCXDS) MultiDec(keys []cipher.SHA256) (err error) {
	if len(keys) == 0 {
		return
	}
	err = m.b.Update(func(tx *buntdb.Tx) (err error) {
		var s, k string
		for _, key := range keys {
			k = key.Hex()
			if s, err = tx.Get(k); err != nil {
				if err == buntdb.ErrNotFound {
					err = nil
					continue
				}
				return // some error
			}
			got := m.decodeString(s)
			if rc := decRefsCount(got); rc == 0 {
				if _, err = tx.Delete(k); err != nil {
					return
				}
				continue
			}
			if _, _, err = tx.Set(k, hex.EncodeToString(got), nil); err != nil {
				return
			}
		}
		return
	})
	return
}

func (m *memoryCXDS) Close() (err error) {
	if err = m.b.Close(); err == buntdb.ErrDatabaseClosed {
		err = nil // suppress this error
	}
	return
}
