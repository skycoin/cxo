package node

import (
	"github.com/skycoin/cxo/node/log"
)

// debug log pins
const (

	// connections

	NewInConnPin  log.Pin = 1 << iota // new incoming connection
	NewOutConnPin                     // new outgoing connection

	ConnEstPin   // connection established
	CloseConnPin // connection closed (except ERR logs)

	// messeges

	MsgSendPin    // send msg
	MsgReceivePin // receive msg

	// filling

	// ...

	// discovery

	DiscoveryPin // show discovery debug logs

	// joiners

	MsgPin  = MsgSendPin | MsgReceivePin // send/receive
	ConnPin = NewInConnPin | NewOutConnPin | ConnEstPin | HskErrPin |
		CloseConnPin // connections
)
