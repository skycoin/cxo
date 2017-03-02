package main

import (
	"net/rpc"
)

// RPCClient represents an RPC client.
type RPCClient struct {
	Client *rpc.Client
}

// RPCMessage represents an RPC message.
type RPCMessage struct {
	Command   string
	Arguments []string
}

// RunClient runs the client on provided address.
func RunClient(addr string) *RPCClient {
	rpcClient := &RPCClient{}
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		panic(err)
	}
	rpcClient.Client = client
	return rpcClient
}

// SendToRPC sends message through RPC connection.
func (rpcClient *RPCClient) SendToRPC(command string, args []string) ([]byte, error) {
	msg := RPCMessage{
		command,
		args,
	}
	var result []byte
	err := rpcClient.Client.Call("RPCReceiver."+msg.Command, msg.Arguments, &result)
	return result, err
}
