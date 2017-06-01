package data

import (
	"fmt"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
)

// A Space represents memory space
type Space int

// String implements fmt.Stringer interface
func (s Space) String() string {
	return HumanMemory(int(s))
}

// A Stat represents database statistic
type Stat struct {
	// Objects is amount of all objects
	// and schemas except root objects
	Objects int `json:"objects"`
	// Space taken by the Objects. The
	// Space is total size of all
	// encoded objects, but the Space
	// doesn't include space taken by
	// key and other
	Space Space `json:"space"`

	// Feeds represents statistic
	// of root objects by feed
	Feeds map[cipher.PubKey]FeedStat `json:"feeds"` // feeds
}

// A FeedStat represents statistic
// of a feed. The statistic contains
// informations about root objects only
type FeedStat struct {
	// Roots is amount of root objects of feed
	Roots int `json:"roots"`
	// Space taken by root objects of feed
	Space Space `json:"space"`
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
			r.Space.String())
	}
	x = fmt.Sprintf("{objects: %d, space: %s, feeds: [%s]}",
		s.Objects,
		s.Space.String(),
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
