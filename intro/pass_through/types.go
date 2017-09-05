package passThrough

import (
	sky "github.com/skycoin/cxo/skyobject"
)

// A Content represnts two arrays of Votes
type Content struct {
	Thread sky.Refs `skyobject:"schema=pt.Vote"`
	Post   sky.Refs `skyobject:"schema=pt.Vote"`
}

// A Vote represents Post or Thread vote
type Vote struct {
	Up    bool
	Index uint32
}
