package data

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	// "github.com/skycoin/skycoin/src/cipher/encoder"
)

//
// helper functions
//

func testPath(t *testing.T) string {
	fl, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer fl.Close()
	return fl.Name()
}

//
// Tests
//

//
// DB
//

func TestDB_View(t *testing.T) {
	// View(func(t Tx) error) (err error)

	//
}

func TestDB_Update(t *testing.T) {
	// Update(func(t Tx) error) (err error)

	//
}

func TestDB_Stat(t *testing.T) {
	// Stat() (s Stat)

	//
}

func TestDB_Close(t *testing.T) {
	// Close() (err error)

	//
}

//
// Tx
//

func TestTx_Objects(t *testing.T) {
	// Objects() Objects

	//
}

func TestTx_Feeds(t *testing.T) {
	// Feeds() Feeds

	//
}

//
// Objects
//

func TestObjects_Del(t *testing.T) {
	// Del(key cipher.SHA256)

	//
}

func TestObjects_Get(t *testing.T) {
	// Get(key cipher.SHA256) (value []byte, ok bool)

	//
}

func TestObjects_Set(t *testing.T) {
	// Set(key cipher.SHA256, value []byte)

	//
}

func TestObjects_Add(t *testing.T) {
	// Add(value []byte) cipher.SHA256

	//
}

func TestObjects_IsExists(t *testing.T) {
	// IsExists(key cipher.SHA256)

	//
}

func TestObjects_Range(t *testing.T) {
	// Range(func(key cipher.SHA256, value []byte) (stop bool))

	//
}

func TestObjects_RangeDel(t *testing.T) {
	// RangeDel(func(key cipher.SHA256, value []byte) (del, stop bool))

	//
}

//
// Feeds
//

func TestFeeds_Add(t *testing.T) {
	// Add(pk cipher.PubKey)

	//
}

func TestFeeds_Del(t *testing.T) {
	// Del(pk cipher.PubKey)

	//
}

func TestFeeds_IsExists(t *testing.T) {
	// IsExists(pk cipher.PubKey)

	//
}

func TestFeeds_List(t *testing.T) {
	// List() []cipher.PubKey

	//
}

func TestFeeds_Range(t *testing.T) {
	// Range(func(pk cipher.PubKey) (stop bool))

	//
}

func TestFeeds_RangeDel(t *testing.T) {
	// RangeDel(func(pk cipher.PubKey) (del, stop bool))

	//
}

func TestFeeds_Roots(t *testing.T) {
	// Roots(pk cipher.PubKey) Roots

	//
}

//
// Roots
//

func TestRoots_Feed(t *testing.T) {
	// Feed() cipher.PubKey

	//
}

func TestRoots_Add(t *testing.T) {
	// Add(rp *RootPack) (err error)

	//
}

func TestRoots_Last(t *testing.T) {
	// Last() (rp *RootPack, ok bool)

	//
}

func TestRoots_Get(t *testing.T) {
	// Get(seq uint64) (rp *RootPack, ok bool)

	//
}

func TestRoots_Del(t *testing.T) {
	// Del(seq uint64)

	//
}

func TestRoots_Range(t *testing.T) {
	// Range(func(rp *RootPack) (stop bool))

	//
}

func TestRoots_Reverse(t *testing.T) {
	// Reverse(fn func(rp *RootPack) (stop bool))

	//
}

func TestRoots_RangeDelete(t *testing.T) {
	// RangeDelete(fn func(rp *RootPack) (del, stop bool))

	//
}

func TestRoots_DelBefore(t *testing.T) {
	// DelBefore(seq uint64)

	//
}

//
// Old
//

func testDataDel(t *testing.T, db DB) {
	key := db.Add([]byte("hey ho"))
	if got, ok := db.Get(key); !ok {
		t.Error("not added")
	} else if string(got) != "hey ho" {
		t.Error("wrong value returned", string(got))
	} else {
		db.Del(key)
		if _, ok := db.Get(key); ok {
			t.Error("not deleted")
		}
	}
}

func TestData_Del(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testDataDel(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testDataDel(t, db)
	})

}

func testDataGet(t *testing.T, db DB) {
	key := db.Add([]byte("hey ho"))
	if got, ok := db.Get(key); !ok {
		t.Error("not added")
	} else if string(got) != "hey ho" {
		t.Error("wrong value returned", string(got))
	}
	if _, ok := db.Get(cipher.SumSHA256([]byte("ho hey"))); ok {
		t.Error("got unexisting value")
	}
}

func TestData_Get(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testDataGet(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testDataGet(t, db)
	})

}

func testDataSet(t *testing.T, db DB) {
	val := []byte("hey ho")
	key := cipher.SumSHA256(val)
	db.Set(key, val)
	if got, ok := db.Get(key); !ok {
		t.Error("not added")
	} else if string(got) != string(val) {
		t.Error("wrong value returned", string(got))
	}
}

func TestData_Set(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testDataSet(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testDataSet(t, db)
	})

}

func testDataAdd(t *testing.T, db DB) {
	val := []byte("hey ho")
	key := db.Add(val)
	if key != cipher.SumSHA256(val) {
		t.Error("wrong key calculated")
	}
	if got, ok := db.Get(key); !ok {
		t.Error("not added")
	} else if string(got) != string(val) {
		t.Error("wrong value returned", string(got))
	}
}

func TestData_Add(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testDataAdd(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testDataAdd(t, db)
	})

}

func testDataIsExists(t *testing.T, db DB) {
	val := []byte("hey ho")
	key := db.Add(val)
	if ok := db.IsExist(key); !ok {
		t.Error("not added")
	}
	if ok := db.IsExist(cipher.SumSHA256([]byte("ho hey"))); ok {
		t.Error("have unexisting value")
	}
}

func TestData_IsExist(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testDataIsExists(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testDataIsExists(t, db)
	})

}

func testDataRange(t *testing.T, db DB) {
	var vals = [][]byte{
		[]byte("one"),
		[]byte("two"),
		[]byte("othree"),
		[]byte("four"),
	}
	for _, val := range vals {
		db.Add(val)
	}
	var collect = make(map[cipher.SHA256][]byte)
	db.Range(func(key cipher.SHA256, value []byte) (stop bool) {
		collect[key] = value
		return
	})
	if len(collect) != len(vals) {
		t.Error("wong amount of values given")
		return
	}
	for _, v := range vals {
		if string(collect[cipher.SumSHA256(v)]) != string(v) {
			t.Error("wrong value")
		}
	}
	var i int
	db.Range(func(key cipher.SHA256, value []byte) (stop bool) {
		i++
		return true
	})
	if i != 1 {
		t.Error("can't stop")
	}
}

func TestData_Range(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testDataRange(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testDataRange(t, db)
	})

}

//
// feeds
//

func testDataFeeds(t *testing.T, db DB) {
	// empty
	if len(db.Feeds()) != 0 {
		t.Error("wrong feeds length")
	}
	// prepare
	var rp RootPack
	rp.Hash = cipher.SumSHA256(rp.Root)
	// one
	pk1, _ := cipher.GenerateKeyPair()
	if err := db.AddRoot(pk1, &rp); err != nil {
		t.Error(err)
		return
	}
	if fs := db.Feeds(); len(fs) != 1 {
		t.Error("wrong feeds length")
	} else if fs[0] != pk1 {
		t.Error("wrong feed content")
	}
	// two
	pk2, _ := cipher.GenerateKeyPair()
	if err := db.AddRoot(pk2, &rp); err != nil {
		t.Error(err)
		return
	}
	if fs := db.Feeds(); len(fs) != 2 {
		t.Error("wrong feeds length")
	} else {
		pks := map[cipher.PubKey]struct{}{
			pk1: {},
			pk2: {},
		}
		for _, pk := range fs {
			if _, ok := pks[pk]; !ok {
				t.Error("missing feed")
			}
		}
	}
}

func TestData_Feeds(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testDataFeeds(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testDataFeeds(t, db)
	})

}

func testDataDelFeed(t *testing.T, db DB) {
	// prepare
	var rp RootPack
	rp.Hash = cipher.SumSHA256(rp.Root) // nil
	pk, _ := cipher.GenerateKeyPair()
	if err := db.AddRoot(pk, &rp); err != nil {
		t.Error(err) // fatal
		return
	}
	// go
	db.DelFeed(pk)
	if len(db.Feeds()) != 0 {
		t.Error("feed is not deleted")
	}
	if _, ok := db.GetRoot(rp.Hash); ok {
		t.Error("root is not deleted")
	}
}

func TestData_DelFeed(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testDataDelFeed(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testDataDelFeed(t, db)
	})

}

func testRootError(t *testing.T, re *RootError, rp *RootPack,
	pk cipher.PubKey) {

	if re.Hash() != rp.Hash {
		t.Error("wrong root hash of RootError")
	} else if re.Feed() != pk {
		t.Error("wrong feed of RootError")
	} else if re.Seq() != rp.Seq {
		t.Error("wrong seq of RootError")
	}
}

func testDataAddRoot(t *testing.T, db DB) {
	// prepare
	var rp RootPack
	rp.Hash = cipher.SumSHA256(rp.Root) // nil
	pk, _ := cipher.GenerateKeyPair()

	t.Run("seq 0 prev", func(t *testing.T) {
		rp.Prev = cipher.SumSHA256([]byte("any")) // unexpected prev reference
		if err := db.AddRoot(pk, &rp); err == nil {
			t.Error("misisng error")
		} else if re, ok := err.(*RootError); !ok {
			t.Error("wroing error type")
		} else {
			testRootError(t, re, &rp, pk)
		}
	})

	t.Run("seq 1 no prev", func(t *testing.T) {
		rp.Seq = 1
		rp.Prev = cipher.SHA256{} // missing prev. reference
		if err := db.AddRoot(pk, &rp); err == nil {
			t.Error("misisng error")
		} else if re, ok := err.(*RootError); !ok {
			t.Error("wroing error type")
		} else {
			testRootError(t, re, &rp, pk)
		}
	})

	t.Run("wrong hash", func(t *testing.T) {
		// reset
		rp.Seq = 0
		rp.Prev = cipher.SHA256{}
		// reset hash
		rp.Hash = cipher.SumSHA256([]byte("any")) // unexpected hash
		if err := db.AddRoot(pk, &rp); err == nil {
			t.Error("misisng error")
		} else if re, ok := err.(*RootError); !ok {
			t.Error("wroing error type")
		} else {
			testRootError(t, re, &rp, pk)
		}
	})

	t.Run("add", func(t *testing.T) {
		// actual hash
		rp.Hash = cipher.SumSHA256(rp.Root)
		if err := db.AddRoot(pk, &rp); err != nil {
			t.Error(err)
		}
		if gr, ok := db.GetRoot(rp.Hash); !ok {
			t.Error("misisng root in roots bucket")
		} else if gr.Hash != rp.Hash {
			t.Error("wrong root saved by hash")
		}
	})

	if t.Failed() {
		return
	}

	t.Run("already exists", func(t *testing.T) {
		if err := db.AddRoot(pk, &rp); err == nil {
			t.Error("misisng error")
		} else if err != ErrRootAlreadyExists {
			t.Error("wrong error")
		}
	})

}

func TestData_AddRoot(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testDataAddRoot(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testDataAddRoot(t, db)
	})

}

func testRootPack(t *testing.T, rp1, rp2 *RootPack) {
	if bytes.Compare(rp1.Root, rp2.Root) != 0 {
		t.Error("wrong Root filed")
	}
	if rp1.Hash != rp2.Hash {
		t.Error("wrong Hash field")
	}
	if rp1.Sig != rp2.Sig {
		t.Error("wrong Sig field")
	}
	if rp1.Seq != rp2.Seq {
		t.Error("wrong Seq filed")
	}
	if rp1.Prev != rp2.Prev {
		t.Error("wrong Prev field")
	}
}

func testLastRoot(t *testing.T, db DB) {
	pk, _ := cipher.GenerateKeyPair()
	// no feed
	if _, ok := db.LastRoot(pk); ok {
		t.Error("unexpected LastRoot")
	}
	// add
	var rp RootPack
	rp.Hash = cipher.SumSHA256(rp.Root)
	if err := db.AddRoot(pk, &rp); err != nil {
		t.Error(err)
		return
	}
	lr, ok := db.LastRoot(pk)
	if !ok {
		t.Error("missing last root")
		return
	}
	testRootPack(t, lr, &rp)
	// second
	rp.Seq = 1
	rp.Prev = rp.Hash // previous
	if err := db.AddRoot(pk, &rp); err != nil {
		t.Error(err)
		return
	}
	lr, ok = db.LastRoot(pk)
	if !ok {
		t.Error("missing last root")
		return
	}
	testRootPack(t, lr, &rp)
}

func TestData_LastRoot(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testLastRoot(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testLastRoot(t, db)
	})

}

// returns RootPack that contains dummy Root filed,
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

func testRangeFeed(t *testing.T, db DB) {
	pk, _ := cipher.GenerateKeyPair()
	// no feed
	var i int
	db.RangeFeed(pk, func(*RootPack) (stop bool) {
		i++
		return
	})
	if i != 0 {
		t.Error("range over feed that doesn't not exist")
	}
	// one
	rp := getRootPack(0, "one")
	if err := db.AddRoot(pk, &rp); err != nil {
		t.Error(err)
		return
	}
	// two
	rp = getRootPack(1, "two")
	if err := db.AddRoot(pk, &rp); err != nil {
		t.Error(err)
		return
	}
	// three
	rp = getRootPack(2, "three")
	if err := db.AddRoot(pk, &rp); err != nil {
		t.Error(err)
		return
	}
	// range
	db.RangeFeed(pk, func(rp *RootPack) (stop bool) {
		if rp.Seq != uint64(i) {
			t.Error("wrong range order")
		}
		i++
		return // continue
	})
	if i != 3 {
		t.Error("wrong range rounds")
	}
	// stop
	i = 0 // reset
	db.RangeFeed(pk, func(rp *RootPack) (stop bool) {
		i++
		return true
	})
	if i != 1 {
		t.Error("can't stop")
	}
}

func TestData_RangeFeed(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testRangeFeed(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testRangeFeed(t, db)
	})

}

func testRangeFeedReverese(t *testing.T, db DB) {
	pk, _ := cipher.GenerateKeyPair()
	// no feed
	var i int
	db.RangeFeedReverse(pk, func(*RootPack) (stop bool) {
		i++
		return
	})
	if i != 0 {
		t.Error("range over feed that doesn't not exist")
	}
	// one
	rp := getRootPack(0, "one")
	if err := db.AddRoot(pk, &rp); err != nil {
		t.Error(err)
		return
	}
	// two
	rp = getRootPack(1, "two")
	if err := db.AddRoot(pk, &rp); err != nil {
		t.Error(err)
		return
	}
	// three
	rp = getRootPack(2, "three")
	if err := db.AddRoot(pk, &rp); err != nil {
		t.Error(err)
		return
	}
	// range
	i = 2
	db.RangeFeedReverse(pk, func(rp *RootPack) (stop bool) {
		t.Log(rp.Seq, i)
		if rp.Seq != uint64(i) {
			t.Error("wrong range order", rp.Seq, i)
		}
		i--
		return // continue
	})
	if i != -1 {
		t.Error("wrong range rounds")
	}
	// stop
	i = 0 // reset
	db.RangeFeedReverse(pk, func(rp *RootPack) (stop bool) {
		i++
		return true
	})
	if i != 1 {
		t.Error("can't stop")
	}
}

func TestData_RangeFeedReverse(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testRangeFeedReverese(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testRangeFeedReverese(t, db)
	})

}

//
// roots
//

func testGetRoot(t *testing.T, db DB) {
	if _, ok := db.GetRoot(cipher.SumSHA256([]byte("any"))); ok {
		t.Error("got root that doesn't exist")
	}
	rp := getRootPack(0, "content")
	pk, _ := cipher.GenerateKeyPair()
	if err := db.AddRoot(pk, &rp); err != nil {
		t.Error(err)
		return
	}
	if gr, ok := db.GetRoot(rp.Hash); !ok {
		t.Error("missing root")
	} else {
		testRootPack(t, gr, &rp)
	}
}

func TestData_GetRoot(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testGetRoot(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testGetRoot(t, db)
	})

}

func testDelRootsBefore(t *testing.T, db DB) {

	// fill
	pk, _ := cipher.GenerateKeyPair()
	rps := []RootPack{
		getRootPack(0, "one"),
		getRootPack(1, "two"),
		getRootPack(2, "three"),
	}

	for _, rp := range rps {
		if err := db.AddRoot(pk, &rp); err != nil {
			t.Error(err)
			return
		}
	}

	// del before 2
	db.DelRootsBefore(pk, 2)
	if s := db.Stat().Feeds; len(s) != 1 {
		t.Error("wrong len of feeds")
	} else if fs, ok := s[pk]; !ok {
		t.Error("missing feed")
	} else if fs.Roots != 1 {
		t.Error("wrong roots count")
	} else if _, ok := db.GetRoot(rps[2].Hash); !ok {
		t.Error("missing expected root")
	}

	// del all
	db.DelRootsBefore(pk, 9000)
	if len(db.Feeds()) == 0 {
		t.Error("feed was deleted")
	}

}

func TestData_DelRootsBefore(t *testing.T) {

	t.Run("mem", func(t *testing.T) {
		testDelRootsBefore(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testDelRootsBefore(t, db)
	})

}

//
// stat and close
//

func TestData_Stat(t *testing.T) {

	t.Skip("not implemented")

	t.Run("mem", func(t *testing.T) {
		// TODO: implement
	})

	t.Run("drive", func(t *testing.T) {
		// TODO: implement
	})

}

func TestData_Close(t *testing.T) {

	t.Skip("not implemented")

	t.Run("mem", func(t *testing.T) {
		// TODO: implement
	})

	t.Run("drive", func(t *testing.T) {
		// TODO: implement
	})

}

func testRangeFeedDelete(t *testing.T, db DB) {

	pk, _ := cipher.GenerateKeyPair()

	t.Run("empty", func(t *testing.T) {
		var called int
		db.RangeFeedDelete(pk, func(rp *RootPack) (_ bool) {
			called++
			return
		})
		if called != 0 {
			t.Error("has been called for empty feed")
		}
	})

	// fill
	rps := []RootPack{
		getRootPack(0, "one"),
		getRootPack(1, "two"),
		getRootPack(2, "three"),
	}

	for _, rp := range rps {
		if err := db.AddRoot(pk, &rp); err != nil {
			t.Error(err)
			return
		}
	}

	t.Run("order", func(t *testing.T) {
		var i uint64
		db.RangeFeedDelete(pk, func(rp *RootPack) (_ bool) {
			if rp.Seq != i {
				t.Error("wrong order")
			}
			i++
			return
		})
	})

	t.Run("full", func(t *testing.T) {

		var called int
		db.RangeFeedDelete(pk, func(rp *RootPack) (del bool) {
			called++
			return rp.Seq == 1 // delete
		})
		if called != len(rps) {
			t.Errorf("has been called wrong times, expected %d, called %d",
				len(rps), called)
		}
		if feeds := db.Stat().Feeds; len(feeds) != 0 {
			if feeds[pk].Roots != 2 {
				t.Error("hasn't been deleted")
			}
		} else {
			t.Error("missing feed")
		}
	})

}

func TestData_RangeFeedDelete(t *testing.T) {

	t.Run("memory", func(t *testing.T) {
		testRangeFeedDelete(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testRangeFeedDelete(t, db)
	})

}

func testRangeDelete(t *testing.T, db DB) {
	// generate objects
	content := []string{
		"one",
		"two",
		"three",
		"four",
	}
	objects := make(map[cipher.SHA256][]byte, len(content))
	for _, c := range content {
		b := []byte(c)
		objects[cipher.SumSHA256(b)] = b
	}
	// fill database
	for k, v := range objects {
		db.Set(k, v)
	}

	// range count
	t.Run("count", func(t *testing.T) {
		var called int
		db.RangeDelete(func(cipher.SHA256) (_ bool) {
			called++
			return
		})
		if called != len(content) {
			t.Errorf("has been called wrong times, expected %d, got %d",
				len(content), called)
		}
	})

	// delete one
	t.Run("delete one", func(t *testing.T) {
		var called int
		var one cipher.SHA256

		for k, v := range objects {
			if string(v) == "one" {
				one = k
				break
			}
		}

		db.RangeDelete(func(key cipher.SHA256) (_ bool) {
			called++
			return key == one
		})

		if called != len(content) {
			t.Errorf("has been called wrong times, expected %d, got %d",
				len(content), called)
		}

		if db.Stat().Objects != len(objects)-1 {
			t.Error("undeleted")
		}

		// save the one back for next tests
		db.Set(one, objects[one])

	})

	// delete all
	t.Run("delete all", func(t *testing.T) {
		var called int
		db.RangeDelete(func(k cipher.SHA256) (del bool) {
			called++
			_, del = objects[k]
			return
		})
		if called != len(objects) {
			t.Errorf("has been called wrong times, expected %d, got %d",
				len(objects), called)
		}
		if db.Stat().Objects != 0 {
			t.Error("undeleted")
		}
	})

}

func TestData_RangeDelete(t *testing.T) {

	t.Run("memory", func(t *testing.T) {
		testRangeDelete(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testRangeDelete(t, db)
	})

}

func testAddFeed(t *testing.T, db DB) {
	pk, _ := cipher.GenerateKeyPair()
	db.AddFeed(pk)
	if !db.HasFeed(pk) {
		t.Error("missing feed")
	}
}

func TestData_AddFeed(t *testing.T) {

	t.Run("memory", func(t *testing.T) {
		testAddFeed(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testAddFeed(t, db)
	})

}

func testHasFeed(t *testing.T, db DB) {
	pk, _ := cipher.GenerateKeyPair()

	if db.HasFeed(pk) {
		t.Error("has unexisting feed")
	}

	rp := getRootPack(0, "ha-ha")

	if err := db.AddRoot(pk, &rp); err != nil {
		t.Error(err)
		return
	}

	if !db.HasFeed(pk) {
		t.Error("missing feed")
	}

}

func TestData_HasFeed(t *testing.T) {

	t.Run("memory", func(t *testing.T) {
		testHasFeed(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		testHasFeed(t, db)
	})

}
