package main

import (
	"net/rpc"

	"github.com/evanlinjin/cxo/encoder"
)

// Client represents a RPC client.
type Client struct {
	Client *rpc.Client
}

// RunClient runs the RPC client.
func RunClient(addr string) *Client {
	rpcClient := &Client{}
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		panic(err)
	}
	rpcClient.Client = client
	return rpcClient
}

// SendToRPC sends data to RPC.
func (rpcClient *Client) SendToRPC(command string, args []string) (Response, error) {
	msg := Message{command, args}
	var result []byte
	err := rpcClient.Client.Call("Server."+msg.Command, msg.Arguments, &result)
	return result, err
}

// Message represents a message sent through via RPC.
type Message struct {
	Command   string
	Arguments []string
}

// Response represents a response recieved from RPC server.
type Response []byte

// Deserialize deserializes the response data.
func (r Response) Deserialize(data interface{}) error {
	return encoder.DeserializeRaw(r, data)
}
