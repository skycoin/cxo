package node

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"
)

// unsubscribe
func (c *Conn) delFillingRootsOfFeed(pk cipher.PubKey) {
	for _, fl := range c.fillers {
		if r := fl.Root(); r.Pub == pk {
			fl.Terminate(ErrUnsubscribed) // close skyobject.Filler
		}
	}
}

// full/drop
func (c *Conn) delFillingRoot(r *skyobject.Root) {
	delete(c.fillers, r.Hash) // delete
}

func (c *Conn) addRequestedObjectToWaitingList(wcxo skyobject.WCXO) {
	c.requests[wcxo.Key] = append(c.requests[wcxo.Key], wcxo.GotQ)
}

// fill a *skyobject.Root
func (c *Conn) fillRoot(r *skyobject.Root) {
	if _, ok := c.fillers[r.Hash]; !ok {
		c.fillers[r.Hash] = c.s.so.NewFiller(r, skyobject.FillingBus{
			WantQ: c.wantq,
			FullQ: c.full,
			DropQ: c.drop,
		})
	}
}

func (c *Conn) closeFillers() {
	for _, fl := range c.fillers {
		fl.Terminate(ErrConnClsoed)
	}
}
