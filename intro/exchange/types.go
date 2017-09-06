package exchange

import (
	cxo "github.com/skycoin/cxo/skyobject"
)

// A Content represnts two arrays of Votes
type Content struct {
	Thread cxo.Refs `skyobject:"schema=exchg.Vote"`
	Post   cxo.Refs `skyobject:"schema=exchg.Vote"`
}

// A Vote represents Post or Thread vote
type Vote struct {
	Up    bool
	Index uint32
}
