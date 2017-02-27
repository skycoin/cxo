package client

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"
)

type SyncContext struct {
	skyobject.ISkyObjects
}

func NewSyncContext(container skyobject.ISkyObjects) SyncContext {
	return SyncContext{container}
}

func (c SyncContext) OnRequest(r Replier, hash cipher.SHA256) {
	for _, item := range c.MissingDependencies(hash) {
		logger.Debugf("Request message: %v", item.Hex())
		if err := r.Reply(Request{Hash: item}); err != nil {
			logger.Error("error sending Request message: %v", err)
		}
	}
}
