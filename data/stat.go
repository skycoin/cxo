package data

import (
	"fmt"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
)

// A Volume represents memory space
type Volume uint32

// String implements fmt.Stringer interface
func (v Volume) String() string {
	return HumanMemory(int(v))
}

// A Stat represents database statistic
type Stat struct {
	// Objects contains statistic of all
	// necessary objects
	Objects ObjectsStat `json:"objects"`

	// Feeds represents statistic
	// of root objects by feed. This
	// map is nil if database
	// doesn't contains feeds
	Feeds map[cipher.PubKey]FeedStat `json:"feeds"` // feeds
}

// An ObjectsStat represents statistic of
// a group of objects
type ObjectsStat struct {
	Amount uint32 `json:"amount"` // amount of objects
	Volume Volume `json:"volume"` // volume without keys
}

// A FeedStat represents statistic
// of a feed
type FeedStat []RootStat

type RootStat struct {
	// Amount of all nested objects
	Amount uint32 `json:"amount"`
	// Volume taken by the Root
	// with all related obejcts
	Volume Volume `json:"volume"`
	// Seq number of the Root
	Seq uint64 `json:"seq"`
}

func shortHex(a string) string {
	return string([]byte(a)[:7])
}

// String implemets fmt.Stringer interface and returns
// human readable string (not implemented yet)
func (s Stat) String() string {
	return fmt.Sprintf("%#v", s)
}

// HumanMemory returns human readable memory string
func HumanMemory(bytes int) string {
	fb := float64(bytes)
	ms := "B"
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
