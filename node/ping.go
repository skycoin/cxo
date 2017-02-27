package node

import (
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

// A Ping is sent from feed to subscribers to keep conenction alive
// if any write timeout is set or we want to communicate long time
type Ping struct{}

// Handle implemetns gnet.Message interface
func (*Ping) Handle(ctx *gnet.MessageContext, ni interface{}) (err error) {
	return ni.(Node).handlePing(ctx.Conn)
}

// A Pong is sent as Ping reply from subscribers back to feed
type Pong struct{}

// Handle implemetns gnet.Message interface
func (*Pong) Handle(ctx *gnet.MessageContext, ni interface{}) (err error) {
	return ni.(Node).handlePong(ctx.Conn)
}
