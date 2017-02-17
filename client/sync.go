package client

import (
	"fmt"
	"github.com/skycoin/cxo/nodeManager"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/skycoin/src/cipher"
)

type syncContext struct {
	container skyobject.ISkyObjects
}

func SyncContext(container skyobject.ISkyObjects) *syncContext {
	return &syncContext{container: container}
}

func (c *syncContext) OnRequest(r *nodeManager.Subscription, hash cipher.SHA256) {
	for _, item := range c.container.MissingDependencies(hash) {
		fmt.Println("RequestMessage", item)
		r.Send(RequestMessage{Hash: item})
	}
}
