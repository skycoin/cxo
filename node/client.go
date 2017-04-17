package node

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/skyobject"
)

type Config struct {
	gnet.Config // pool configurations

	Connect string          // server to connect
	Feeds   []cipher.PubKey // feeds to subscribe to
}

type Client struct {
	log.Logger

	// skyobjects
	db *data.DB
	so *skyobject.Container

	// network
	pool *gnet.Pool

	// feeds
	feeds []cipher.PubKey
}
