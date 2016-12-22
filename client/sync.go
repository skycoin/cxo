package client

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/nodeManager"
	"fmt"
)

type syncContext struct {
	container  skyobject.ISkyObjects
}

func SyncContext(container skyobject.ISkyObjects) *syncContext {
	return &syncContext{container:container}
}

func (c *syncContext) OnRequest(r *nodeManager.Subscription, hash cipher.SHA256) {
	toSync := c.container.MissingDependencies(hash)
	if (len(toSync) >0){
		for _, item := range toSync {
			r.Send(RequestMessage{Hash:item})
		}
		return
	}
	fmt.Println("Nothing to sync", c.container.Statistic())
}
