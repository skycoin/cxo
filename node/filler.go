package node

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"
)

// unsubscribe
func (c *Conn) delFillingRootsOfFeed(pk cipher.PubKey) {
	for _, fl := range c.fillers {
		if r := fl.Root(); r.Pub == pk {
			fl.Close()                // close skyobject.Filler
			delete(c.fillers, r.Hash) // delete
		}
	}
}

// full/drop
func (c *Conn) delFillingRoot(r *skyobject.Root) {
	if fr, ok := f.fillers[r.Hash]; ok {
		fr.Close()                // close skyobejct.Filler
		delete(f.fillers, r.Hash) // delete
	}
}

func (c *Conn) addRequestedObjectToWaitingList(wcxo skyobject.WCXO) {
	c.requests[wcxo.Hash] = append(c.requests[wcxo.Hash], wcxo.GotQ)
}

// fill a *skyobject.Root
func (c *Conn) fillRoot(r *skyobject.Root) {
	if _, ok := c.fillers[r.Hash]; !ok {
		c.fillers[r.Hash] = c.s.so.NewFiller(r, skyobject.FillingBus{
			c.wantq,
			c.full,
			c.drop,
			&c.s.await,
		})
	}
}

func (c *Conn) closeFillers() {
	for _, fl := range c.fillers {
		fl.Close()
	}
}
