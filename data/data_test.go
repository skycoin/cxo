package data

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

//
// helper functions
//

const TM time.Duration = 50 * time.Millisecond

func shouldNotPanic(t *testing.T) {
	if err := recover(); err != nil {
		t.Error("unexpected panic:", err)
	}
}

func receiveTimeout(t *testing.T, c <-chan struct{}, tm time.Duration) {
	select {
	case <-c:
	case <-time.After(tm):
		t.Error("locked (timeout)")
	}
}

func sendTimeout(t *testing.T, c chan<- struct{}, tm time.Duration) {
	select {
	case c <- struct{}{}:
	case <-time.After(tm):
		t.Error("locked (timeout)")
	}
}

func testPath(t *testing.T) string {
	fl, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer fl.Close()
	return fl.Name()
}

func testDriveDB(t *testing.T) (db DB, cleanUp func()) {
	dbFile := testPath(t)
	db, err := NewDriveDB(dbFile)
	if err != nil {
		os.Remove(dbFile)
		t.Fatal(err)
	}
	cleanUp = func() {
		db.Close()
		os.Remove(dbFile)
	}
	return
}

// returns RootPack that contains dummy Root field,
// the field can't be used to encode/decode
func getRootPack(seq uint64, content string) (rp RootPack) {
	rp.Seq = seq
	if seq != 0 {
		rp.Prev = cipher.SumSHA256([]byte("any"))
	}
	rp.Root = []byte(content)
	rp.Hash = cipher.SumSHA256(rp.Root)
	return
}

type testObjectKeyValue struct {
	key   cipher.SHA256
	value []byte
}

type testObjectKeyValues []testObjectKeyValue

func (t testObjectKeyValues) Len() int {
	return len(t)
}

func (t testObjectKeyValues) Less(i, j int) bool {
	return bytes.Compare(t[i].key[:], t[j].key[:]) < 0
}

func (t testObjectKeyValues) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func testSortedObjects(input ...string) (to testObjectKeyValues) {
	to = make(testObjectKeyValues, 0, len(input))
	for _, s := range input {
		to = append(to, testObjectKeyValue{
			key:   cipher.SumSHA256([]byte(s)),
			value: []byte(s),
		})
	}
	sort.Sort(to)
	return
}

// testOrderedPublicKeys retursn slice with two generated
// public keys in ascending order
func testOrderedPublicKeys() []cipher.PubKey {
	// add feeds
	pk1, _ := cipher.GenerateKeyPair()
	pk2, _ := cipher.GenerateKeyPair()

	// be sure that keys are not equal
	for pk1 == pk2 {
		pk2, _ = cipher.GenerateKeyPair()
	}

	// oreder
	if bytes.Compare(pk2[:], pk1[:]) < 0 { // if pk2 < pk1
		pk1, pk2 = pk2, pk1 // swap
	}

	return []cipher.PubKey{pk1, pk2}
}

func testComparePublicKeyLists(t *testing.T, a, b []cipher.PubKey) {
	if len(a) != len(b) {
		t.Error("wrong list length")
		return
	}
	for i, ax := range a {
		if bx := b[i]; bx != ax {
			t.Errorf("wrong item %d: want %s, got %s",
				i,
				ax.Hex(), // shortHex(
				bx.Hex()) // shortHex(
		}
	}
}

//
// Tests
//

//
// DB
//

func testDBView(t *testing.T, db DB) {

	t.Run("concurent", func(t *testing.T) {
		wg := new(sync.WaitGroup)

		v1 := make(chan struct{}, 1)
		v2 := make(chan struct{}, 1)

		concurent := func(wg *sync.WaitGroup, sc, rc chan struct{}) {
			defer wg.Done()

			err := db.View(func(tx Tv) (_ error) {
				sc <- struct{}{}
				receiveTimeout(t, rc, TM)
				return
			})
			if err != nil {
				t.Error(err)
			}
		}

		wg.Add(2)

		go concurent(wg, v1, v2)
		go concurent(wg, v2, v1)

		wg.Wait()

	})

	t.Run("update lock", func(t *testing.T) {
		t.Skip("boltdb allows simultaneous Update and View")
	})

}

func TestDB_View(t *testing.T) {
	// View(func(t Tv) error) (err error)

	t.Run("memory", func(t *testing.T) {
		testDBView(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testDBView(t, db)
	})

}

func testDBUpdate(t *testing.T, db DB) {

	var key cipher.SHA256

	ae := errors.New("just an average error to rollback this transaction")

	err := db.Update(func(tx Tu) (err error) {
		key, err = tx.Objects().Add([]byte("a value"))
		if err != nil {
			return
		}
		return ae
	})
	if err != ae {
		t.Error(err)
	}

	err = db.View(func(tx Tv) (_ error) {
		if tx.Objects().Get(key) != nil {
			t.Error("doesn't rolled back")
		}
		return
	})
	if err != nil {
		t.Error(err)
	}

}

func TestDB_Update(t *testing.T) {
	// Update(func(t Tu) error) (err error)

	t.Run("memory", func(t *testing.T) {
		testDBUpdate(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testDBUpdate(t, db)
	})

}

func testDBStat(t *testing.T, db DB) {

	t.Run("empty", func(t *testing.T) {
		stat := db.Stat()
		if stat.Objects.Amount != 0 {
			t.Error("objects in empty db")
		}
		if stat.Objects.Volume != 0 {
			t.Error("non-zero volume of empty db")
		}
		if len(stat.Feeds) != 0 {
			t.Error("feeds in empty db")
		}
	})

	// fill with objects
	objects := []string{"one", "two", "three", "c4"}
	err := db.Update(func(tx Tu) (_ error) {
		objs := tx.Objects()
		for _, s := range objects {
			if _, err := objs.Add([]byte(s)); err != nil {
				return err
			}
		}
		return
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("objects", func(t *testing.T) {
		stat := db.Stat()
		if int(stat.Objects.Amount) != len(objects) {
			t.Error("wrong amount of objects")
		}
		if stat.Objects.Volume == 0 {
			t.Error("wrong volume")
		}
	})

	pk, _ := cipher.GenerateKeyPair()

	if testFillWithExampleFeed(t, pk, db); t.Failed() {
		return
	}

	t.Run("feeds", func(t *testing.T) {
		stat := db.Stat()
		if stat.Feeds == nil {
			t.Error("missing feed in stat")
			return
		}
		if fs, ok := stat.Feeds[pk]; !ok {
			t.Error("mising feed in stat")
		} else if len(fs.Roots) != 3 {
			t.Error("wrong root count")
		} else if fs.Volume == 0 {
			t.Error("wrong amount of space of roots")
		}

	})
}

func TestDB_Stat(t *testing.T) {
	// Stat() (s Stat)

	t.Run("memory", func(t *testing.T) {
		testDBStat(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testDBStat(t, db)
	})

}

func testDBClose(t *testing.T, db DB) {
	if err := db.Close(); err != nil {
		t.Error("closing error:", err)
	}
	// Close can be called many times
	defer shouldNotPanic(t)
	db.Close()
}

func TestDB_Close(t *testing.T) {
	// Close() (err error)

	t.Run("memory", func(t *testing.T) {
		testDBClose(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testDBClose(t, db)
	})

}

//
// Tv
//

func testTvObjects(t *testing.T, db DB) {
	err := db.View(func(tx Tv) (_ error) {
		if tx.Objects() == nil {
			t.Error("Tv.Objects returns nil")
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
	return
}

func TestTv_Objects(t *testing.T) {
	// Objects() ViewObjects

	t.Run("memory", func(t *testing.T) {
		testTvObjects(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testTvObjects(t, db)
	})

}

func testTvFeeds(t *testing.T, db DB) {
	err := db.View(func(tx Tv) (_ error) {
		if tx.Feeds() == nil {
			t.Error("Tv.Feeds returns nil")
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTv_Feeds(t *testing.T) {
	// Feeds() ViewFeeds

	t.Run("memory", func(t *testing.T) {
		testTvFeeds(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testTvFeeds(t, db)
	})

}

func testTvMisc(t *testing.T, db DB) {
	err := db.View(func(tx Tv) (_ error) {
		if tx.Misc() == nil {
			t.Error("Tv.Misc returns nil")
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTv_Misc(t *testing.T) {
	// Feeds() ViewFeeds

	t.Run("memory", func(t *testing.T) {
		testTvMisc(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testTvMisc(t, db)
	})

}

//
// Tu
//

func testTuObjects(t *testing.T, db DB) {
	err := db.Update(func(tx Tu) (_ error) {
		if tx.Objects() == nil {
			t.Error("Tu.Objects returns nil")
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
	return
}

func TestTu_Objects(t *testing.T) {
	// Objects() UpdateObjects

	t.Run("memory", func(t *testing.T) {
		testTuObjects(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testTuObjects(t, db)
	})

}

func testTuFeeds(t *testing.T, db DB) {
	err := db.Update(func(tx Tu) (_ error) {
		if tx.Feeds() == nil {
			t.Error("Tu.Feeds returns nil")
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTu_Feeds(t *testing.T) {
	// Feeds() UpdateFeeds

	t.Run("memory", func(t *testing.T) {
		testTuFeeds(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testTuFeeds(t, db)
	})

}

func testTuMisc(t *testing.T, db DB) {
	err := db.Update(func(tx Tu) (_ error) {
		if tx.Misc() == nil {
			t.Error("Tu.Misc returns nil")
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTu_Misc(t *testing.T) {
	// Feeds() ViewFeeds

	t.Run("memory", func(t *testing.T) {
		testTuMisc(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testTuMisc(t, db)
	})

}
