package tests

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func testKeyValue(s string) (key cipher.SHA256, val []byte) {
	val = []byte(s)
	key = cipher.SumSHA256(val)
	return
}

// CXDSGet tests Get method of CXDS
func CXDSGet(t *testing.T, ds data.CXDS) {

	key, value := testKeyValue("something")

	t.Run("not exist", func(t *testing.T) {
		if val, rc, err := ds.Get(key); err == nil {
			t.Error("missing error")
		} else if err != data.ErrNotFound {
			t.Error("unexpected error:", err)
		} else if rc != 0 {
			t.Error("wrong rc", rc)
		} else if val != nil {
			t.Error("not nil")
		}
	})

	if _, err := ds.Set(key, value); err != nil {
		t.Error(err)
		return
	}

	t.Run("existing", func(t *testing.T) {
		if val, rc, err := ds.Get(key); err != nil {
			t.Error(err)
		} else if rc != 1 {
			t.Error("wrong rc", rc)
		} else if want, got := string(value), string(val); want != got {
			t.Errorf("wrong value: want %q, got %q", want, got)
		}
	})

}

// CXDSSet tests Set method of CXDS
func CXDSSet(t *testing.T, ds data.CXDS) {

	key, value := testKeyValue("something")

	t.Run("new", func(t *testing.T) {
		if rc, err := ds.Set(key, value); err != nil {
			t.Error(err)
		} else if rc != 1 {
			t.Error("wrong rc", rc)
		}

		if val, rc, err := ds.Get(key); err != nil {
			t.Error(err)
		} else if rc != 1 {
			t.Error("wrong rc", rc)
		} else if want, got := string(value), string(val); want != got {
			t.Errorf("wrong value: want %q, got %q", want, got)
		}
	})

	t.Run("twice", func(t *testing.T) {
		if rc, err := ds.Set(key, value); err != nil {
			t.Error(err)
		} else if rc != 2 {
			t.Error("wrong rc", rc)
		}

		if val, rc, err := ds.Get(key); err != nil {
			t.Error(err)
		} else if rc != 2 {
			t.Error("wrong rc", rc)
		} else if want, got := string(value), string(val); want != got {
			t.Errorf("wrong value: want %q, got %q", want, got)
		}
	})

}

// CXDSAdd tests Add method of CXDS
func CXDSAdd(t *testing.T, ds data.CXDS) {

	key, value := testKeyValue("something")

	t.Run("new", func(t *testing.T) {
		if k, rc, err := ds.Add(value); err != nil {
			t.Error(err)
		} else if rc != 1 {
			t.Error("wrong rc", rc)
		} else if k != key {
			t.Error("wrong key returned")
		}

		if val, rc, err := ds.Get(key); err != nil {
			t.Error(err)
		} else if rc != 1 {
			t.Error("wrong rc", rc)
		} else if want, got := string(value), string(val); want != got {
			t.Errorf("wrong value: want %q, got %q", want, got)
		}
	})

	t.Run("twice", func(t *testing.T) {
		if k, rc, err := ds.Add(value); err != nil {
			t.Error(err)
		} else if rc != 2 {
			t.Error("wrong rc", rc)
		} else if k != key {
			t.Error("wrong key returned")
		}

		if val, rc, err := ds.Get(key); err != nil {
			t.Error(err)
		} else if rc != 2 {
			t.Error("wrong rc", rc)
		} else if want, got := string(value), string(val); want != got {
			t.Errorf("wrong value: want %q, got %q", want, got)
		}
	})

}

// CXDSInc tests Inc method of CXDS
func CXDSInc(t *testing.T, ds data.CXDS) {

	key, value := testKeyValue("something")

	t.Run("not exist", func(t *testing.T) {
		if rc, err := ds.Inc(key); err == nil {
			t.Error("missing error")
		} else if err != data.ErrNotFound {
			t.Error("unexpected error:", err)
		} else if rc != 0 {
			t.Error("wrong rc", rc)
		}

		if val, rc, err := ds.Get(key); err == nil {
			t.Error("missing error")
		} else if err != data.ErrNotFound {
			t.Error("unexpected error:", err)
		} else if rc != 0 {
			t.Error("wrong rc", rc)
		} else if got := string(val); got != "" {
			t.Errorf("unexpected value %q", got)
		}
	})

	if _, err := ds.Set(key, value); err != nil {
		t.Error(err)
		return
	}

	t.Run("exist", func(t *testing.T) {
		if rc, err := ds.Inc(key); err != nil {
			t.Error(err)
		} else if rc != 2 {
			t.Error("wrong rc", rc)
		}

		if val, rc, err := ds.Get(key); err != nil {
			t.Error(err)
		} else if rc != 2 {
			t.Error("wrong rc", rc)
		} else if want, got := string(value), string(val); want != got {
			t.Errorf("wrong value: want %q, got %q", want, got)
		}
	})

}

// CXDSDec tests Dec method of CXDS
func CXDSDec(t *testing.T, ds data.CXDS) {

	key, value := testKeyValue("something")

	t.Run("not exist", func(t *testing.T) {
		if rc, err := ds.Dec(key); err == nil {
			t.Error("missing error")
		} else if err != data.ErrNotFound {
			t.Error("unexpected error:", err)
		} else if rc != 0 {
			t.Error("wrong rc", rc)
		}

		if val, rc, err := ds.Get(key); err == nil {
			t.Error("missing error")
		} else if err != data.ErrNotFound {
			t.Error("unexpected error:", err)
		} else if rc != 0 {
			t.Error("wrong rc", rc)
		} else if got := string(val); got != "" {
			t.Errorf("unexpected value %q", got)
		}
	})

	if _, err := ds.Set(key, value); err != nil {
		t.Error(err)
		return
	}

	if _, err := ds.Inc(key); err != nil {
		t.Error(err)
		return
	}

	t.Run("decrement", func(t *testing.T) {
		if rc, err := ds.Dec(key); err != nil {
			t.Error(err)
		} else if rc != 1 {
			t.Error("wrong rc", rc)
		}

		if val, rc, err := ds.Get(key); err != nil {
			t.Error(err)
		} else if rc != 1 {
			t.Error("wrong rc", rc)
		} else if want, got := string(value), string(val); want != got {
			t.Errorf("wrong value: want %q, got %q", want, got)
		}
	})

	t.Run("delete", func(t *testing.T) {
		if rc, err := ds.Dec(key); err != nil {
			t.Error(err)
		} else if rc != 0 {
			t.Error("wrong rc", rc)
		}

		if val, rc, err := ds.Get(key); err == nil {
			t.Error("missing error")
		} else if err != data.ErrNotFound {
			t.Error("unexpected error:", err)
		} else if rc != 0 {
			t.Error("wrong rc", rc)
		} else if got := string(val); got != "" {
			t.Errorf("unexpected value %q", got)
		}
	})

}

// CXDSMultiGet tests MultiGet method of CXDS
func CXDSMultiGet(t *testing.T, ds data.CXDS) {

	k1, v1 := testKeyValue("something")
	k2, v2 := testKeyValue("another one")

	keys := []cipher.SHA256{k1, k2}
	values := [][]byte{v1, v2}

	t.Run("not exist", func(t *testing.T) {
		if _, err := ds.MultiGet(keys); err == nil {
			t.Error("missing error")
		} else if err != data.ErrNotFound {
			t.Error("unexpected error:", err)
		}
	})

	for i, k := range keys {
		if _, err := ds.Set(k, values[i]); err != nil {
			t.Error(err)
			return
		}
	}

	t.Run("exists", func(t *testing.T) {
		if vals, err := ds.MultiGet(keys); err != nil {
			t.Error(err)
		} else {
			if want, got := len(keys), len(vals); want != got {
				t.Errorf("wrong number of values returned: want %d, got %d",
					want, got)
				return
			}
			for i, val := range vals {
				if want, got := string(values[i]), string(val); want != got {
					t.Errorf("wrong %d value: want %q, got %q", i, want, got)
				}
			}
		}
	})

	mix := []cipher.SHA256{
		k1,
		cipher.SumSHA256([]byte("mi")),
		k2,
		cipher.SumSHA256([]byte("ix")),
	}

	t.Run("mix", func(t *testing.T) {
		if _, err := ds.MultiGet(mix); err == nil {
			t.Error("missing error")
		} else if err != data.ErrNotFound {
			t.Error(err)
		}
	})

}

// CXDSMultiAdd tests MultiAdd method of CXDS
func CXDSMultiAdd(t *testing.T, ds data.CXDS) {

	k1, v1 := testKeyValue("something")
	k2, v2 := testKeyValue("another one")

	keys := []cipher.SHA256{k1, k2}
	values := [][]byte{v1, v2}

	t.Run("new", func(t *testing.T) {
		if err := ds.MultiAdd(values); err != nil {
			t.Error(err)
			return
		}

		for i, k := range keys {
			if val, rc, err := ds.Get(k); err != nil {
				t.Error(err)
			} else if rc != 1 {
				t.Error("wrong rc", rc)
			} else if want, got := string(values[i]), string(val); want != got {
				t.Errorf("wrong value: want %q, got %q", want, got)
			}
		}
	})

	t.Run("overwrite", func(t *testing.T) {
		if err := ds.MultiAdd(values); err != nil {
			t.Error(err)
			return
		}

		for i, k := range keys {
			if val, rc, err := ds.Get(k); err != nil {
				t.Error(err)
			} else if rc != 2 {
				t.Error("wrong rc", rc)
			} else if want, got := string(values[i]), string(val); want != got {
				t.Errorf("wrong value: want %q, got %q", want, got)
			}
		}
	})

}

// CXDSMultiInc tests MultiInc method of CXDS
func CXDSMultiInc(t *testing.T, ds data.CXDS) {

	k1, v1 := testKeyValue("something")
	k2, v2 := testKeyValue("another one")

	keys := []cipher.SHA256{k1, k2}
	values := [][]byte{v1, v2}

	if err := ds.MultiAdd(values); err != nil {
		t.Error(err)
		return
	}

	t.Run("first", func(t *testing.T) {
		if err := ds.MultiInc(keys); err != nil {
			t.Error(err)
			return
		}

		// value must not be changes by MultiInc
		for i, k := range keys {
			if val, rc, err := ds.Get(k); err != nil {
				t.Error(err)
			} else if rc != 2 {
				t.Error("wrong rc", rc)
			} else if want, got := string(values[i]), string(val); want != got {
				t.Errorf("wrong value: want %q, got %q", want, got)
			}
		}
	})

	t.Run("second", func(t *testing.T) {
		if err := ds.MultiInc(keys); err != nil {
			t.Error(err)
			return
		}

		// value must not be changes by MultiInc
		for i, k := range keys {
			if val, rc, err := ds.Get(k); err != nil {
				t.Error(err)
			} else if rc != 3 {
				t.Error("wrong rc", rc)
			} else if want, got := string(values[i]), string(val); want != got {
				t.Errorf("wrong value: want %q, got %q", want, got)
			}
		}
	})

	mix := []cipher.SHA256{
		k1,
		cipher.SumSHA256([]byte("mi")),
		k2,
		cipher.SumSHA256([]byte("ix")),
	}

	t.Run("mix", func(t *testing.T) {
		// MultiInc shoud silently ignore unexisting values
		if err := ds.MultiInc(mix); err == nil {
			t.Error("misisng error")
		} else if err != data.ErrNotFound {
			t.Error("unexpected error:", err)
		}

		// value must not be changes by MultiInc
		for i, k := range keys {
			if val, rc, err := ds.Get(k); err != nil {
				t.Error(err)
			} else if false == (rc == 4 || rc == 3) {
				t.Error("wrong rc", rc)
			} else if want, got := string(values[i]), string(val); want != got {
				t.Errorf("wrong value: want %q, got %q", want, got)
			}
		}
	})

}

// CXDSMultiDec tests MultiDec method of CXDS
func CXDSMultiDec(t *testing.T, ds data.CXDS) {

	k1, v1 := testKeyValue("something")
	k2, v2 := testKeyValue("another one")

	keys := []cipher.SHA256{k1, k2}
	values := [][]byte{v1, v2}

	if err := ds.MultiAdd(values); err != nil {
		t.Error(err)
		return
	}

	for i := 0; i < 2; i++ {
		// otherwise, the MultiDec below removes values
		if err := ds.MultiInc(keys); err != nil {
			t.Error(err)
			return
		}
	}

	t.Run("first", func(t *testing.T) {
		if err := ds.MultiDec(keys); err != nil {
			t.Error(err)
			return
		}

		// value must not be changes by MultiInc
		for i, k := range keys {
			if val, rc, err := ds.Get(k); err != nil {
				t.Error(err)
			} else if rc != 2 {
				t.Error("wrong rc", rc)
			} else if want, got := string(values[i]), string(val); want != got {
				t.Errorf("wrong value: want %q, got %q", want, got)
			}
		}
	})

	t.Run("second", func(t *testing.T) {
		if err := ds.MultiDec(keys); err != nil {
			t.Error(err)
			return
		}

		// value must not be changes by MultiInc
		for i, k := range keys {
			if val, rc, err := ds.Get(k); err != nil {
				t.Error(err)
			} else if rc != 1 {
				t.Error("wrong rc", rc)
			} else if want, got := string(values[i]), string(val); want != got {
				t.Errorf("wrong value: want %q, got %q", want, got)
			}
		}
	})

	mix := []cipher.SHA256{
		cipher.SumSHA256([]byte("mi")),
		k1,
		cipher.SumSHA256([]byte("ix")),
		k2,
	}

	t.Run("mix", func(t *testing.T) {
		if err := ds.MultiDec(mix); err == nil {
			t.Error("missing error")
		} else if err != data.ErrNotFound {
			t.Error("unexpected error:", err)
		}
	})

	t.Run("delete", func(t *testing.T) {
		if err := ds.MultiDec(keys); err != nil {
			t.Error(err)
			return
		}

		// value must not be changes by MultiInc
		for _, k := range keys {
			if _, _, err := ds.Get(k); err == nil {
				t.Error("missing error")
			} else if err != data.ErrNotFound {
				t.Error("unexpected error:", err)
			}
		}
	})

}

// CXDSClose tests Close method of CXDS
func CXDSClose(t *testing.T, ds data.CXDS) {
	if err := ds.Close(); err != nil {
		t.Error(err)
	}
	if err := ds.Close(); err != nil {
		t.Error(err)
	}
}
