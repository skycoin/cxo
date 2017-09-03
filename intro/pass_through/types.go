package passThrough

import (
	sky "github.com/skycoin/cxo/skyobject"
)

type Content struct {
	Thread sky.Refs `skyobject:"schema=pt.Vote"`
	Post   sky.Refs `skyobject:"schema=pt.Vote"`
}

type Vote struct {
	Up    bool
	Index uint32
}
