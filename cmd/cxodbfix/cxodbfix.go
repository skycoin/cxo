package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/boltdb/bolt"
)

var (
	metaBucket = []byte("m")       // bucket with meta information
	versionKey = []byte("version") // value with version in the meta bucket
)

func main() {

	// paths
	var idxdbPath, cxdsPath string

	flag.StringVar(&idxdbPath, "idxdb", "idxdb", "path to idxdb")
	flag.StringVar(&cxdsPath, "cxds", "cxds.db", "path to cxds")

	flag.Usage = usage

	flag.Parse()

	// let's go

	if err := fix(idxdbPath, cxdsPath); err != nil {
		log.Fatal(err) // exit code should be 1
	}

	fmt.Print("done")
}

func usage() {

	fmt.Printf(`Use %[1]s -idxdb=<path> -cxds=<path>

The %[1]s fixes CXO DB, and it also used to move from
prevous version to new one, keeping data if possible

Arguments are

    -idxdb  path to idxdb, use :memeory: to notify the utility that
            idxdb in memory used and should not be fixed; default value
            is 'idx.db'
    -cxds   path to cxds, use :memory: to notify the utility that
            cxds in memory used and should not be fixed; default value
            is 'cxds.db'

Before running the utility you have to be sure that:
    1) the latest version of the utility used
    2) you have made a backups of your DB files
    3) all applications that uses this files closed
    4) the utility has write access to directory with db files


For example
    %[1]s idxdb=idxdb.db -cxds=:memory:

    Fix given idxdb, skipping cxds

    %[1]s

    Perform all available fixes using idx.db and cxds.db files in
    current directory

No way to rollback fixes, use backups if you want.
Also, the utilit can't fix broken databases.

`, os.Args[1])

}

// returns exit code
func fix(idxdbPath, cxdsPath string) (err error) {
	if err = fcxds(cxdsPath); err != nil {
		return // stop on first error
	}
	return fidxdb(idxdbPath)
}

func tempFile(dbPath, prefix string) (tf string, err error) {
	var fl *os.File
	if fl, err = ioutil.TempFile(filepath.Dir(dbPath), prefix); err != nil {
		return
	}
	defer fl.Close()
	return fl.Name(), nil

}

func versionBytes(v int) (vb []byte) {
	vb = make([]byte, 4)
	binary.BigEndian.PutUint32(vb, uint32(v))
	return
}

func dup(val []byte) (cp []byte) {
	cp = make([]byte, len(val))
	copy(cp, val)
	return
}

// fix cxds
func fcxds(cxdsPath string) (err error) {

	if cxdsPath == ":memory:" {
		return // nothnig to fix
	}

	// available fixes

	var fixes = []func(sdb, ddb *bolt.DB) (err error){
		fcxds0_1, // 0 -> 1
	}

	var sdb, ddb *bolt.DB // src -> dst

	if sdb, err = bolt.Open(cxdsPath, 0644, nil); err != nil {
		return err
	}

	var version int // current version of the CXDS

	err = sdb.View(func(tx *bolt.Tx) (err error) {

		// check meta bucket presence

		meta := tx.Bucket(metaBucket)

		if meta == nil {
			return // version is 0
		}

		// get version

		vb := meta.Get(versionKey)

		if len(vb) == 0 {
			return errors.New("broken CXDS file, missing version, but has meta")
		}

		if len(vb) != 4 {
			return errors.New("brokne CXDS file, wrong version length")
		}

		version = int(binary.BigEndian.Uint32(vb))

		return // nil
	})

	if err != nil {
		return sdb.Close() // close the source
	}

	// if the CXDS already fixed
	if version == len(fixes) {
		return sdb.Close() // already fixed
	}

	// create and open temporary database
	var ddbPath string
	if ddbPath, err = tempFile(sdb.Path(), "cxds"); err != nil {
		sdb.Close() // close the source
		return
	}
	defer os.Remove(ddbPath) // remove the temp file after all

	if ddb, err = bolt.Open(ddbPath, 0644, nil); err != nil {
		sdb.Close()
		return
	}

	for i, fix := range fixes {
		// skip already fixed versions
		if i < version {
			continue
		}

		if err = fix(sdb, ddb); err != nil {
			sdb.Close()
			ddb.Close()
			return // fatality
		}

		version++ // don't skip next fix
	}

	sdb.Close() // clsoe, ignoring error
	ddb.Close() // clsoe, ignoring error

	return os.Rename(ddbPath, cxdsPath) // replace with fixed
}

// fix idxdb
func fidxdb(idxdbPath string) (code int) {
	//
}

// 0-> 1, fix CXDS the little-endian refs counts bug #128
func fcxds0_1(sdb, ddb *bolt.DB) (err error) {
	fmt.Println("[CXDS] fix 0 -> 1, #128")

	var objsBucket = []byte("o")

	// create necessary buckets

	err = ddb.Update(func(tx *bolt.Tx) (err error) {

		// objects
		if _, err = tx.CreateBucketIfNotExists(objsBucket); err != nil {
			return
		}

		var info *bolt.Bucket
		if info, err = tx.CreateBucketIfNotExists(metaBucket); err != nil {
			return
		}

		// 0 -> 1
		return info.Put(versionKey, versionBytes(1))
	})

	if err != nil {
		return
	}

	// service functions
	var getRefsCount = func(val []byte) (rc uint32) {
		rc = binary.LittleEndian.Uint32(val) // le
		return
	}

	// returns new uint32
	var setRefsCount = func(rc uint32, val []byte) {
		binary.BigEndian.PutUint32(val, rc) // be
		return
	}

	// range cover objects

	return sdb.View(func(stx *bolt.Tx) (err error) {

		sobjs := stx.Bucket(objsBucket)

		return ddb.Update(func(dtx *bolt.Tx) (err error) {
			dobjs := dtx.Bucket(objsBucket)

			return sobjs.ForEach(func(k, v []byte) (err error) {
				dv := dup(v) // we have to copy (boltdb #204)
				setRefsCount(getRefsCount(v), dv)
				return dobjs.Put(k, dv)
			})

		})
	})

}
