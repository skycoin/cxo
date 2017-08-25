package cxds

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

const testFileName = "test.db"

func testKeyValue(s string) (key cipher.SHA256, val []byte) {
	val = []byte(s)
	key = cipher.SumSHA256(val)
	return
}

func testShouldNotPanic(t *testing.T) {
	if pc := recover(); pc != nil {
		t.Error("unexpected panic:", pc)
	}
}

func Test_one(t *testing.T) {
	if len(one) != 4 {
		t.Fatal("wrong length of the one")
	}
	if binary.LittleEndian.Uint32(one) != 1 {
		t.Error("the one is not a one")
	}
}

func testDriveDS(t *testing.T) (ds CXDS) {
	var err error
	if ds, err = NewDriveCXDS(testFileName); err != nil {
		t.Fatal(err)
	}
	return
}

func TestNewDriveCXDS(t *testing.T) {
	// NewDriveCXDS(filePath string) (ds CXDS, err error)

	ds := testDriveDS(t)
	defer ds.Close()
}

func TestNewMemoryCXDS(t *testing.T) {
	// NewMemoryCXDS() (ds CXDS, err error)

	ds := NewMemoryCXDS()
	defer ds.Close()
}

func testGet(t *testing.T, ds CXDS) {

	key, value := testKeyValue("something")

	t.Run("not exist", func(t *testing.T) {
		if val, rc, err := ds.Get(key); err != nil {
			t.Error(err)
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

func TestCXDS_Get(t *testing.T) {
	// Get(key cipher.SHA256) (val []byte, rc uint32, err error)

	t.Run("memory", func(t *testing.T) {
		testGet(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		testGet(t, ds)
	})
}

func testSet(t *testing.T, ds CXDS) {

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

func TestCXDS_Set(t *testing.T) {
	// Set(key cipher.SHA256, val []byte) (rc uint32, err error)

	t.Run("memory", func(t *testing.T) {
		testSet(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		testSet(t, ds)
	})
}

func testAdd(t *testing.T, ds CXDS) {

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

func TestCXDS_Add(t *testing.T) {
	// Add(val []byte) (key cipher.SHA256, rc uint32, err error)

	t.Run("memory", func(t *testing.T) {
		testAdd(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		testAdd(t, ds)
	})
}

func testInc(t *testing.T, ds CXDS) {

	key, value := testKeyValue("something")

	t.Run("not exist", func(t *testing.T) {
		if rc, err := ds.Inc(key); err != nil {
			t.Error(err)
		} else if rc != 0 {
			t.Error("wrong rc", rc)
		}

		if val, rc, err := ds.Get(key); err != nil {
			t.Error(err)
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

func TestCXDS_Inc(t *testing.T) {
	// Inc(key cipher.SHA256) (rc uint32, err error)

	t.Run("memory", func(t *testing.T) {
		testInc(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		testInc(t, ds)
	})
}

func testDec(t *testing.T, ds CXDS) {

	key, value := testKeyValue("something")

	t.Run("not exist", func(t *testing.T) {
		if rc, err := ds.Dec(key); err != nil {
			t.Error(err)
		} else if rc != 0 {
			t.Error("wrong rc", rc)
		}

		if val, rc, err := ds.Get(key); err != nil {
			t.Error(err)
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

		if val, rc, err := ds.Get(key); err != nil {
			t.Error(err)
		} else if rc != 0 {
			t.Error("wrong rc", rc)
		} else if got := string(val); got != "" {
			t.Errorf("unexpected value %q", got)
		}
	})

}

func TestCXDS_Dec(t *testing.T) {
	// Dec(key cipher.SHA256) (rc uint32, err error)

	t.Run("memory", func(t *testing.T) {
		testDec(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		testDec(t, ds)
	})
}

func testMultiGet(t *testing.T, ds CXDS) {

	k1, v1 := testKeyValue("something")
	k2, v2 := testKeyValue("another one")

	keys := []cipher.SHA256{k1, k2}
	values := [][]byte{v1, v2}

	t.Run("not exist", func(t *testing.T) {
		if vals, err := ds.MultiGet(keys); err != nil {
			t.Error(err)
		} else {
			if want, got := len(keys), len(vals); want != got {
				t.Error("wrong number of values returned: want %d, got %d",
					want, got)
				return
			}
			for _, val := range vals {
				if val != nil {
					t.Errorf("got unexpected value: %q", val)
				}
			}
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
		if vals, err := ds.MultiGet(mix); err != nil {
			t.Error(err)
		} else {
			if want, got := len(mix), len(vals); want != got {
				t.Errorf("wrong number of values returned: want %d, got %d",
					want, got)
				return
			}
			for i, val := range vals {
				if i == 1 || i == 3 {
					if val != nil {
						t.Error("unexpected value", string(val))
					}
					continue
				}
				if i != 0 {
					i = 1
				}
				if want, got := string(values[i]), string(val); want != got {
					t.Errorf("wrong %d value: want %q, got %q", i, want, got)
				}
			}
		}
	})

}

func TestCXDS_MultiGet(t *testing.T) {
	// MultiGet(keys []cipher.SHA256) (vals [][]byte, err error)

	t.Run("memory", func(t *testing.T) {
		testMultiGet(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		testMultiGet(t, ds)
	})
}

func testMultiAdd(t *testing.T, ds CXDS) {

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

func TestCXDS_MultiAdd(t *testing.T) {
	// MultiAdd(vals [][]byte) (err error)

	t.Run("memory", func(t *testing.T) {
		testMultiAdd(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		testMultiAdd(t, ds)
	})
}

func testMultiInc(t *testing.T, ds CXDS) {

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
		if err := ds.MultiInc(mix); err != nil {
			t.Error(err)
			return
		}

		// value must not be changes by MultiInc
		for i, k := range keys {
			if val, rc, err := ds.Get(k); err != nil {
				t.Error(err)
			} else if rc != 4 {
				t.Error("wrong rc", rc)
			} else if want, got := string(values[i]), string(val); want != got {
				t.Errorf("wrong value: want %q, got %q", want, got)
			}
		}
	})

}

func TestCXDS_MultiInc(t *testing.T) {
	// MultiGet(keys []cipher.SHA256) (vals [][]byte, err error)

	t.Run("memory", func(t *testing.T) {
		testMultiInc(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		testMultiInc(t, ds)
	})
}

func testMultiDec(t *testing.T, ds CXDS) {

	k1, v1 := testKeyValue("something")
	k2, v2 := testKeyValue("another one")

	keys := []cipher.SHA256{k1, k2}
	values := [][]byte{v1, v2}

	if err := ds.MultiAdd(values); err != nil {
		t.Error(err)
		return
	}

	for i := 0; i < 3; i++ {
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
			} else if rc != 3 {
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
			} else if rc != 2 {
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
		if err := ds.MultiDec(mix); err != nil {
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

	t.Run("delete", func(t *testing.T) {
		if err := ds.MultiDec(keys); err != nil {
			t.Error(err)
			return
		}

		// value must not be changes by MultiInc
		for _, k := range keys {
			if val, rc, err := ds.Get(k); err != nil {
				t.Error(err)
			} else if rc != 0 {
				t.Error("wrong rc", rc)
			} else if got := string(val); got != "" {
				t.Errorf("got unexpected value: %q", got)
			}
		}
	})

}

func TestCXDS_MultiDec(t *testing.T) {
	// MultiAdd(vals [][]byte) (err error)

	t.Run("memory", func(t *testing.T) {
		testMultiDec(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		testMultiDec(t, ds)
	})
}

func testClose(t *testing.T, ds CXDS) {
	if err := ds.Close(); err != nil {
		t.Error(err)
	}
	if err := ds.Close(); err != nil {
		t.Error(err)
	}
}

func TestCXDS_Close(t *testing.T) {
	// Close() (err error)

	t.Run("memory", func(t *testing.T) {
		testClose(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		testClose(t, ds)
	})
}
