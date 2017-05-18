package data

import (
	"fmt"
	"strings"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

type Stat struct {
	Total  int `json:"total"`
	Memory int `json:"memory"`
}

// String implemets fmt.Stringer interface and returns
// human readable string
func (s Stat) String() string {
	return fmt.Sprintf("{total: %d, memory: %s}",
		s.Total,
		HumanMemory(s.Memory))
}

// HumanMemory returns human readable memory string
func HumanMemory(bytes int) string {
	var fb float64 = float64(bytes)
	var ms string = "B"
	for _, m := range []string{"KiB", "MiB", "GiB"} {
		if fb > 1024.0 {
			fb = fb / 1024.0
			ms = m
			continue
		}
		break
	}
	if ms == "B" {
		return fmt.Sprintf("%.0fB", fb)
	}
	// 2.00 => 2
	// 2.10 => 2.1
	// 2.53 => 2.53
	return strings.TrimRight(
		strings.TrimRight(fmt.Sprintf("%.2f", fb), "0"),
		".") + ms
}

// A DB is common database interface
type DB interface {

	//
	// Data related methods
	//

	// Has the DB an object with the key
	Has(key cipher.SHA256) (ok bool)
	// Get an object by its key
	Get(key cipher.SHA256) (v []byte, ok bool)
	// Set object using it pre-calculated key
	Set(key cipher.SHA256, data []byte)
	// Range over data objects (read-only)
	Range(fn func(key cipher.SHA256))
	// AddAutoKey add an object ot DB and get its key
	AddAutoKey(data []byte) (key cipher.SHA256)
	// Del deletes and object by its key
	Del(key cipher.SHA256)

	//
	// Root objects related methods
	//

	// Root by hash
	Root(hash cipher.SHA256) (rp RootPack, ok bool)
	// Roots returns key of all roots of a feed
	Roots(pk cipher.PubKey) (keys []cipher.SHA256)
	// AddRoot to the feed
	AddRoot(pk cipher.PubKey, rp RootPack)
	// RangeFeed
	RangeFeed(pk cipher.PubKey, fn func(hash cipher.SHA256, rp RootPack))

	//
	// Other methods
	//

	// Stat of the DB
	Stat() (s Stat)
	// Close the DB
	Close() (err error)
}

// A RootPack represents encoded root object with signature,
// seq number, and next/prev/this hashes
type RootPack struct {
	Root []byte
	Sig  cipher.Sig
	Seq  uint64

	Hash RootReference
	Prev RootReference
	Next RootReference
}
