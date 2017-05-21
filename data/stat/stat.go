// Package stat represents statistic of a database.
// The package introduced to use it inside packages
// (such as 'cli') that doesn't need full functionality
// of the cxo/data package. Thus the package avoids
// unnessesary import dependencies (like boltdb)
package stat

import (
	"fmt"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
)

// A Space represents memory space
type Space int

// String implements fmt.Stringer interface
func (s Space) Strign() string {
	return HumanMemory(int(s))
}

// A Stat represents database statistic
type Stat struct {
	Objects int   `json:"objects"` // amount of objects and schemas
	Space   Space `json:"space"`   // space taken by objects

	Feeds map[cipher.PubKey]FeedStat `json:"feeds"` // feeds
}

type FeedStat struct {
	Roots int   `json:"roots"` // amount of root objects of this feed
	Space Space `json:"space"` // space taken by root objects of this feed
}

func shortHex(a string) string {
	return string([]byte(a)[:7])
}

// String implemets fmt.Stringer interface and returns
// human readable string
func (s Stat) String() (x string) {
	feeds := ""
	for k, r := range s.Feeds {
		feeds += fmt.Sprintf("<%s>{roots %d, space: %s}",
			shortHex(k.Hex()),
			r.Roots,
			r.Space.Strign())
	}
	x = fmt.Sprintf("{objects: %d, space: %s, feeds: [%s]}",
		s.Objects,
		s.Space.Strign(),
		feeds)
	return
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
