package node

import (
	"net/rpc"
)

type RPCClient struct {
	c *rpc.Client
}
