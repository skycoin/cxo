package client

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"
)

type syncContext struct {
	container skyobject.ISkyObjects
}

func SyncContext(container skyobject.ISkyObjects) *syncContext {
	return &syncContext{container: container}
}

func (c *syncContext) OnRequest(
	r Replier,
	hash cipher.SHA256) {

	for _, item := range c.container.MissingDependencies(hash) {
		fmt.Println("RequestMessage", item)
		if err := r.Reply(RequestMessage{Hash: item}); err != nil {
			logger.Error("error sending reply: ", err)
		}
	}
}
