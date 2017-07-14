package data_test

import (
	"fmt"
	"log"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func Example() {
	var db data.DB

	db = data.NewMemoryDB()

	//
	// objects
	//

	fmt.Println("Objects:")

	objects := []string{
		"one",
		"two",
		"three",
		"c4",
	}

	keys := make([]cipher.SHA256, 0, len(objects))

	// write

	err := db.Update(func(tx data.Tu) (_ error) {
		objs := tx.Objects()
		for _, o := range objects {
			k, err := objs.Add([]byte(o))
			if err != nil {
				return err // rollback
			}
			keys = append(keys, k)
		}
		return
	})
	if err != nil {
		log.Fatal(err)
	}

	// read

	err = db.View(func(tx data.Tv) (_ error) {
		objs := tx.Objects()
		return objs.Range(func(_ cipher.SHA256, value []byte) (_ error) {
			fmt.Println(" ", string(value))
			return
		})
	})
	if err != nil {
		log.Fatal(err)
	}

	//
	// feeds
	//

	fmt.Println()
	fmt.Println("Feeds:")

	pk, _ := cipher.GenerateDeterministicKeyPair([]byte("x"))

	// write

	err = db.Update(func(tx data.Tu) error {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		log.Fatal(err)
	}

	// read

	err = db.View(func(tx data.Tv) (_ error) {
		if tx.Feeds().IsExist(pk) {
			fmt.Println(" ", pk.Hex())
		}
		return
	})

	//
	// roots
	//

	fmt.Println()
	fmt.Println("Roots:")

	// write

	err = db.Update(func(tx data.Tu) error {
		roots := tx.Feeds().Roots(pk)
		if roots == nil {
			log.Fatal("missing feed")
		}
		return roots.Add(&data.RootPack{
			Seq:  0,
			Hash: cipher.SumSHA256([]byte("encoded content")),
			Root: []byte("encoded content"),
		})
	})
	if err != nil {
		log.Fatal(err)
	}

	// read

	err = db.View(func(tx data.Tv) (_ error) {
		roots := tx.Feeds().Roots(pk)
		if roots == nil {
			log.Fatal("misisng feed")
		}
		rp := roots.Last()
		if rp == nil {
			log.Fatal("empty feed")
		}
		fmt.Println(" ", string(rp.Root))
		return
	})

	// Output:
	//
	// Objects:
	//   c4
	//   two
	//   one
	//   three
	//
	// Feeds:
	//   0268bc0885c2b9dc58199403bc5ac529e2962b326188cd09caabeb9d5e61f26c39
	//
	// Roots:
	//   encoded content

}
