package data

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	// "github.com/skycoin/skycoin/src/cipher/encoder"
)

func testPath(t *testing.T) string {
	fl, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer fl.Close()
	return fl.Name()
}

//
// objects
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
		db := NewMemoryDB()
		// Del
		testDataDel(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Del
		//
		testDataDel(t, db)
		//
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
		db := NewMemoryDB()
		// Get
		//
		testDataGet(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Get
		//
		testDataGet(t, db)
		//
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
		db := NewMemoryDB()
		// Set
		//
		testDataSet(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Set
		//
		testDataSet(t, db)
		//
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
		db := NewMemoryDB()
		// Add
		//
		testDataAdd(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Add
		//
		testDataAdd(t, db)
		//
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
		db := NewMemoryDB()
		// IsExist
		//
		testDataIsExists(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// IsExist
		//
		testDataIsExists(t, db)
		//
	})
}

func testDataRange(t *testing.T, db DB) {
	var vals [][]byte = [][]byte{
		[]byte("one"),
		[]byte("two"),
		[]byte("othree"),
		[]byte("four"),
	}
	for _, val := range vals {
		db.Add(val)
	}
	var collect map[cipher.SHA256][]byte = make(map[cipher.SHA256][]byte)
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
		db := NewMemoryDB()
		// Range
		//
		testDataRange(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Range
		//
		testDataRange(t, db)
		//
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
		db := NewMemoryDB()
		// Feeds
		//
		testDataFeeds(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Feeds
		//
		testDataFeeds(t, db)
		//
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
		db := NewMemoryDB()
		// DelFeed
		//
		testDataDelFeed(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// DelFeed
		//
		testDataDelFeed(t, db)
		//
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
		db := NewMemoryDB()
		// AddRoot
		//
		testDataAddRoot(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// AddRoot
		//
		testDataAddRoot(t, db)
		//
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
		db := NewMemoryDB()
		// LastRoot
		//
		testLastRoot(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// LastRoot
		//
		testLastRoot(t, db)
		//
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
	var i int = 0
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
		db := NewMemoryDB()
		// RangeFeed
		//
		testRangeFeed(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// RangeFeed
		//
		testRangeFeed(t, db)
		//
	})
}

func testRangeFeedReverese(t *testing.T, db DB) {
	pk, _ := cipher.GenerateKeyPair()
	// no feed
	var i int = 0
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
		db := NewMemoryDB()
		// RangeFeedReverse
		//
		testRangeFeedReverese(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// RangeFeedReverse
		//
		testRangeFeedReverese(t, db)
		//
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
		db := NewMemoryDB()
		// GetRoot
		//
		testGetRoot(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// GetRoot
		//
		testGetRoot(t, db)
		//
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
		db := NewMemoryDB()
		// DelRootsBefore
		//
		testDelRootsBefore(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// DelRootsBefore
		//
		testDelRootsBefore(t, db)
		//
	})
}

//
// stat and close
//

func TestData_Stat(t *testing.T) {

	t.Skip("not implemented")

	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Stat
		//
		_ = db
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Stat
		//
		_ = db
		//
	})
}

func TestData_Close(t *testing.T) {

	t.Skip("not implemented")

	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Close
		//
		_ = db
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Close
		//
		_ = db
		//
	})
}
