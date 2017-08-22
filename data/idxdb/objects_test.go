package idxdb

import (
	"fmt"
	"os"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func testObjectsInc(t *testing.T, idx IdxDB) {

	key, o := testKeyObject("ha")

	t.Run("not exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if rc, err := objs.Inc(key); err != nil {
				t.Error(err)
			} else if rc != 0 {
				t.Error("wrong rc", rc)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := idx.Tx(func(tx Tx) error { return tx.Objects().Set(key, o) })
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if rc, err := objs.Inc(key); err != nil {
				t.Error(err)
			} else if rc != 2 {
				t.Error("wrong rc", rc)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestObjects_Inc(t *testing.T) {
	// Inc(key cipher.SHA256) (rc uint32, err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsInc(t, idx)
	})
}

func testObjectsGet(t *testing.T, idx IdxDB) {
	key, o := testKeyObject("ha")

	t.Run("not exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if o, err := objs.Get(key); err != nil {
				t.Error(err)
			} else if o != nil {
				t.Error("unexpected object")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := idx.Tx(func(tx Tx) error { return tx.Objects().Set(key, o) })
	if err != nil {
		t.Error(err)
		return
	}

	var acst int64 // keep acess time

	t.Run("exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if x, err := objs.Get(key); err != nil {
				t.Error(err)
			} else if *x != *o {
				t.Error("wrong object")
			} else {
				acst = x.AccessTime // store
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if t.Failed() {
		return
	}

	t.Run("access time", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if x, err := objs.Get(key); err != nil {
				t.Error(err)
			} else {
				xAcst := x.AccessTime
				x.AccessTime = acst // set to compare
				if *x != *o {
					t.Error("wrong")
				}
				if xAcst <= acst {
					t.Error("wrong new AccessTime")
				}
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})
}

func TestObjects_Get(t *testing.T) {
	// Get(key cipher.SHA256) (o *Object, err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsGet(t, idx)
	})
}

func testObjectsMultiGet(t *testing.T, idx IdxDB) {

	var keys []cipher.SHA256
	var vals []*Object

	for i := 0; i < 8; i++ {
		if i%2 != 0 {
			keys = append(keys, cipher.SumSHA256(
				[]byte(fmt.Sprintf("ha #%d", i))),
			)
			vals = append(vals, nil)
			continue
		}
		key, o := testKeyObject(fmt.Sprintf("ha #%d", i))
		keys = append(keys, key)
		vals = append(vals, o)
	}

	t.Run("not exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if os, err := objs.MultiGet(keys); err != nil {
				t.Error(err)
			} else if len(os) != len(keys) {
				t.Error("wrong length", len(os))
			} else {
				for _, o := range os {
					if o != nil {
						t.Error("not nil")
					}
				}
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := idx.Tx(func(tx Tx) (_ error) {
		objs := tx.Objects()
		for i, val := range vals {
			if val == nil {
				continue
			}
			if err := objs.Set(keys[i], val); err != nil {
				t.Error(err)
			}
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
	if t.Failed() {
		return
	}

	acsts := make([]int64, 8) // keep acess time

	t.Run("exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if xs, err := objs.MultiGet(keys); err != nil {
				t.Error(err)
			} else if len(xs) != len(keys) {
				t.Error("wrong length")
			} else {
				for i, x := range xs {
					if i%2 != 0 {
						if x != nil {
							t.Error("unexpected value")
						}
						continue
					}
					if x == nil {
						t.Error("misisng object")
						continue
					}
					acsts[i] = x.AccessTime // store
					if *x != *vals[i] {
						t.Error("wrong")
					}
				}
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if t.Failed() {
		return
	}

	t.Run("access time", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if xs, err := objs.MultiGet(keys); err != nil {
				t.Error(err)
			} else if len(xs) != len(keys) {
				t.Error("wrong length")
			} else {
				for i, x := range xs {
					if i%2 != 0 {
						if x != nil {
							t.Error("unexpected value")
						}
						continue
					}
					if x == nil {
						t.Error("mising obejct")
						continue
					}
					xAcst := x.AccessTime
					x.AccessTime = acsts[i] // set to compare
					if *x != *vals[i] {
						t.Error("wrong")
					}
					if xAcst <= acsts[i] {
						t.Error("wrong new AccessTime")
					}
				}
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})
}

func TestObjects_MultiGet(t *testing.T) {
	// MultiGet(keys []cipher.SHA256) (os []*Object, err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsMultiGet(t, idx)
	})
}

func testObjectsMultiInc(t *testing.T, idx IdxDB) {

	var keys []cipher.SHA256
	var vals []*Object

	for i := 0; i < 8; i++ {
		if i%2 != 0 {
			keys = append(keys, cipher.SumSHA256(
				[]byte(fmt.Sprintf("ha #%d", i))),
			)
			vals = append(vals, nil)
			continue
		}
		key, o := testKeyObject(fmt.Sprintf("ha #%d", i))
		keys = append(keys, key)
		vals = append(vals, o)
	}

	t.Run("not exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			if err := tx.Objects().MultiInc(keys); err != nil {
				t.Error(err)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := idx.Tx(func(tx Tx) (_ error) {
		objs := tx.Objects()
		for i, val := range vals {
			if val == nil {
				continue
			}
			if err := objs.Set(keys[i], val); err != nil {
				t.Error(err)
			}
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
	if t.Failed() {
		return
	}

	t.Run("exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if err := objs.MultiInc(keys); err != nil {
				t.Error(err)
				return
			}
			if xs, err := objs.MultiGet(keys); err != nil {
				t.Error(err)
			} else if len(xs) != len(keys) {
				t.Error("wrong length")
			} else {
				for i, x := range xs {
					if i%2 != 0 {
						if x != nil {
							t.Error("unexpected value")
						}
						continue
					}
					if x == nil {
						t.Error("misisng object")
						continue
					}
					if x.RefsCount != 2 {
						t.Error("wrong rc", x.RefsCount)
					}
					xAcst := x.AccessTime

					x.AccessTime = vals[i].AccessTime // to compare
					x.RefsCount = 1                   // to compare
					if *x != *vals[i] {
						t.Error("wrong")
					}
					if xAcst <= vals[i].AccessTime {
						t.Error("wrong new AccessTime")
					}
				}
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if t.Failed() {
		return
	}

	t.Run("access time", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if err := objs.MultiInc(keys); err != nil {
				t.Error(err)
				return
			}
			if xs, err := objs.MultiGet(keys); err != nil {
				t.Error(err)
			} else if len(xs) != len(keys) {
				t.Error("wrong length")
			} else {
				for i, x := range xs {
					if i%2 != 0 {
						if x != nil {
							t.Error("unexpected value")
						}
						continue
					}
					if x == nil {
						t.Error("mising obejct")
						continue
					}
					if x.RefsCount != 3 {
						t.Error("wrong rc", x.RefsCount)
					}
					xAcst := x.AccessTime

					x.AccessTime = vals[i].AccessTime // to compare
					x.RefsCount = 1                   // to compare
					if *x != *vals[i] {
						t.Error("wrong")
					}
					if xAcst <= vals[i].AccessTime {
						t.Error("wrong new AccessTime")
					}
				}
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})
}

func TestObjects_MultiInc(t *testing.T) {
	// MultiInc(keys []cipher.SHA256) (err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsMultiInc(t, idx)
	})
}

func testObjectsIterate(t *testing.T, idx IdxDB) {

	t.Run("empty", func(t *testing.T) {
		var called int
		err := idx.Tx(func(tx Tx) (_ error) {
			return tx.Objects().Iterate(func(cipher.SHA256, *Object) (_ error) {
				called++
				return
			})
		})
		if err != nil {
			t.Error(err)
		}
		if called != 0 {
			t.Error("called")
		}
	})

	var keys []cipher.SHA256
	var vals []*Object

	for i := 0; i < 8; i++ {
		key, o := testKeyObject(fmt.Sprintf("ha #%d", i))
		keys = append(keys, key)
		vals = append(vals, o)
	}

	err := idx.Tx(func(tx Tx) (_ error) {
		objs := tx.Objects()
		for i, val := range vals {
			if err := objs.Set(keys[i], val); err != nil {
				t.Error(err)
			}
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
	if t.Failed() {
		return
	}

	t.Run("stop iteration", func(t *testing.T) {
		var called int
		err := idx.Tx(func(tx Tx) (_ error) {
			return tx.Objects().Iterate(func(cipher.SHA256, *Object) (_ error) {
				called++
				return ErrStopIteration
			})
		})
		if err != nil {
			t.Error(err)
		}
		if called != 1 {
			t.Error("called wrong times", called)
		}
	})

	t.Run("pass error through", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			err := tx.Objects().Iterate(func(cipher.SHA256, *Object) (_ error) {
				return errTestError
			})
			if err != errTestError {
				t.Error("wrong error:", err)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	kvs := make(map[cipher.SHA256]*Object)
	for i := 0; i < len(keys); i++ {
		kvs[keys[i]] = vals[i]
	}

	actms := make(map[cipher.SHA256]int64, len(keys))

	t.Run("iterate", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			var i int
			err := objs.Iterate(func(key cipher.SHA256, o *Object) (_ error) {
				if i > len(keys) {
					t.Error("iterates too long", i)
					return ErrStopIteration
				}
				if x, ok := kvs[key]; !ok {
					t.Error("wrong key")
				} else {
					actms[key] = o.AccessTime   // store for next test
					x.AccessTime = o.AccessTime // to compare
					if *x != *o {
						t.Error("wrong value", i, *x, *o)
					}
				}
				delete(kvs, key)
				i++
				return
			})
			if err != nil {
				t.Error(err)
			}
			if len(kvs) != 0 {
				t.Error("short iteration")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("access time", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			var i int
			err := objs.Iterate(func(key cipher.SHA256, o *Object) (_ error) {
				if i > len(keys) {
					t.Error("iterates too long", i)
					return ErrStopIteration
				}
				if at, ok := actms[key]; !ok {
					t.Error("wrong key")
				} else if at >= o.AccessTime {
					t.Error("wrong access time")
				}
				delete(actms, key)
				i++
				return
			})
			if err != nil {
				t.Error(err)
			}
			if len(actms) != 0 {
				t.Error("short iteration")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestObjects_Iterate(t *testing.T) {
	// Iterate(IterateObjectsFunc) (err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsIterate(t, idx)
	})
}

func testObjectsDec(t *testing.T, idx IdxDB) {

	key, o := testKeyObject("ha")

	t.Run("not exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if rc, err := objs.Dec(key); err != nil {
				t.Error(err)
			} else if rc != 0 {
				t.Error("wrong rc", rc)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := idx.Tx(func(tx Tx) error { return tx.Objects().Set(key, o) })
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if rc, err := objs.Inc(key); err != nil {
				t.Error(err)
			} else if rc != 2 {
				t.Error("wrong rc", rc)
			}
			if rc, err := objs.Dec(key); err != nil {
				t.Error(err)
			} else if rc != 1 {
				t.Error("wrong rc", rc)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("delete", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if rc, err := objs.Dec(key); err != nil {
				t.Error(err)
			} else if rc != 0 {
				t.Error("wrong rc", rc)
			}
			if o, err := objs.Get(key); err != nil {
				t.Error(err)
			} else if o != nil {
				t.Error("object was not deleted")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestObjects_Dec(t *testing.T) {
	// Dec(key cipher.SHA256) (rc uint32, err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsDec(t, idx)
	})
}

func testObjectsSet(t *testing.T, idx IdxDB) {

	key, o := testKeyObject("zorro!")

	t.Run("set", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if err := objs.Set(key, o); err != nil {
				t.Error(err)
				return
			}
			if o.RefsCount != 1 {
				t.Error("wrong RefsCount", o.RefsCount)
			}
			if o.AccessTime == 776 {
				t.Error("access time was not changed")
			}
			if o.CreateTime == 778 {
				t.Error("CreateTime was not changed")
			}
			if x, err := objs.Get(key); err != nil {
				t.Error(err)
			} else if *x != *o {
				t.Error("wrong value")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if t.Failed() {
		return
	}

	t.Run("increment", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if err := objs.Set(key, o); err != nil {
				t.Error(err)
				return
			}
			if o.RefsCount != 2 {
				t.Error("wrong RefsCount", o.RefsCount)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestObjects_Set(t *testing.T) {
	// Set(key cipher.SHA256, o *Object) (err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsSet(t, idx)
	})
}

func testObjectsMultiSet(t *testing.T, idx IdxDB) {

	var keys []cipher.SHA256
	var vals []*Object

	for i := 0; i < 8; i++ {
		key, o := testKeyObject(fmt.Sprintf("ha #%d", i))
		keys = append(keys, key)
		vals = append(vals, o)
	}

	kos := make([]KeyObject, len(keys))
	for i := 0; i < len(keys); i++ {
		kos[i] = KeyObject{keys[i], vals[i]}
	}

	t.Run("set", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if err := objs.MultiSet(kos); err != nil {
				t.Error(err)
				return
			}
			for _, ko := range kos {
				if ko.Object.RefsCount != 1 {
					t.Error("wrong RefsCount", ko.Object.RefsCount)
				}
				if ko.Object.AccessTime == 776 {
					t.Error("access time was not changed")
				}
				if ko.Object.CreateTime == 778 {
					t.Error("CreateTime was not changed")
				}
				if x, err := objs.Get(ko.Key); err != nil {
					t.Error(err)
				} else if *x != *ko.Object {
					t.Error("wrong value")
				}
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if t.Failed() {
		return
	}

	t.Run("increment", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if err := objs.MultiSet(kos); err != nil {
				t.Error(err)
				return
			}
			for _, ko := range kos {
				if ko.Object.RefsCount != 2 {
					t.Error("wrong RefsCount", ko.Object.RefsCount)
				}
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestObjects_MultiSet(t *testing.T) {
	// MultiSet(ko []KeyObject) (err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsMultiSet(t, idx)
	})
}

func testObjectsMulitDec(t *testing.T, idx IdxDB) {

	var keys []cipher.SHA256
	var vals []*Object

	for i := 0; i < 8; i++ {
		key, o := testKeyObject(fmt.Sprintf("ha #%d", i))
		keys = append(keys, key)
		vals = append(vals, o)
	}

	t.Run("not exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) error {
			return tx.Objects().MulitDec(keys)
		})
		if err != nil {
			t.Error(err)
		}
	})

	if t.Failed() {
		return
	}

	err := idx.Tx(func(tx Tx) (err error) {
		objs := tx.Objects()
		for i := 0; i < len(keys); i++ {
			if err = objs.Set(keys[i], vals[i]); err != nil {
				return
			}
		}
		return objs.MultiInc(keys)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("decrement", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			objs := tx.Objects()
			if err = objs.MulitDec(keys); err != nil {
				return
			}
			for i := 0; i < len(keys); i++ {
				var o *Object
				if o, err = objs.Get(keys[i]); err != nil {
					return
				} else if o.RefsCount != 1 {
					t.Error("wrong rc", o.RefsCount)
				}
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if t.Failed() {
		return
	}

	t.Run("delete", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			objs := tx.Objects()
			if err = objs.MulitDec(keys); err != nil {
				return
			}
			for i := 0; i < len(keys); i++ {
				var o *Object
				if o, err = objs.Get(keys[i]); err != nil {
					return
				} else if o != nil {
					t.Error("not deleted")
				}
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestObjects_MulitDec(t *testing.T) {
	// MulitDec(keys []cipher.SHA256) (err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsMulitDec(t, idx)
	})
}

func testObjectsAmount(t *testing.T, idx IdxDB) {

	var keys []cipher.SHA256
	var vals []*Object

	for i := 0; i < 8; i++ {
		key, o := testKeyObject(fmt.Sprintf("ha #%d", i))
		keys = append(keys, key)
		vals = append(vals, o)
	}

	t.Run("empty", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			if tx.Objects().Amount() != 0 {
				t.Error("wrong amount")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := idx.Tx(func(tx Tx) (err error) {
		objs := tx.Objects()
		for i := 0; i < len(keys); i++ {
			if err = objs.Set(keys[i], vals[i]); err != nil {
				return
			}
		}
		return objs.MultiInc(keys)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("full", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			if tx.Objects().Amount() != Amount(len(keys)) {
				t.Error("wrong Amount")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestObjects_Amount(t *testing.T) {
	// Amount() Amount

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsAmount(t, idx)
	})
}

func testObjectsVolume(t *testing.T, idx IdxDB) {

	var keys []cipher.SHA256
	var vals []*Object

	for i := 0; i < 8; i++ {
		key, o := testKeyObject(fmt.Sprintf("ha #%d", i))
		keys = append(keys, key)
		vals = append(vals, o)
	}

	t.Run("empty", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			if tx.Objects().Volume() != 0 {
				t.Error("wrong volume")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	var total Volume

	err := idx.Tx(func(tx Tx) (err error) {
		objs := tx.Objects()
		for i := 0; i < len(keys); i++ {
			if err = objs.Set(keys[i], vals[i]); err != nil {
				return
			}
			total += vals[i].Vol
		}
		return objs.MultiInc(keys)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("full", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			if vol := tx.Objects().Volume(); vol != total {
				t.Error("wrong Volume:", vol, total)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestObjects_Volume(t *testing.T) {
	// Volume() Volume

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsVolume(t, idx)
	})
}
