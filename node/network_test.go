package node

import (
	"testing"
	"time"

	// TORM
	"github.com/kr/pretty"
	//

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

func init() {
	gnet.DebugPrint = testing.Verbose()
}

//
// TODO: DRY
//

// Send message from feed to single subscriber using broadcsting.
// The subscriber sends response back. That's all
func TestNode_sendReceive(t *testing.T) {
	sendReceiveStore.Reset() // reset
	var err error
	tf, ts := newTestHook(), newTestHook()
	feed, subscr := mustCreateNode("feed"), mustCreateNode("subscr")
	for _, x := range []struct {
		node Node
		hook *testHook
	}{
		{feed, tf},
		{subscr, ts},
	} {
		x.node.Register(SendReceive{}) //
		x.hook.Set(x.node)
		if err := x.node.Start(); err != nil {
			t.Error(err)
			return
		}
		defer x.node.Close()
	}
	// subscribe to feed
	err = subscr.Outgoing().Connect(
		nodeMustGetAddress(feed, t),
		cipher.PubKey{})
	if err != nil {
		t.Error(err)
		return
	}

	// waiting connections
	if !tf.estIncTimeout(1, time.Second, t) || // handshake for incoming conn
		!ts.estOutTimeout(1, time.Second, t) { // handshake for outgoing conn
		return
	}

	// broadcast SendReceive message
	sr := sendReceiveStore.New()
	if err = feed.Incoming().Broadcast(sr); err != nil {
		t.Error("broadcasting error: ", err)
		return
	}

	// waiting receiving
	// receive from subscriber and receive from feed
	if !ts.recvOutTimeout(1, time.Second, t) ||
		!tf.recvIncTimeout(1, time.Second, t) {
		return
	}

	result, ok := sendReceiveStore.Get(sr.Id)
	if !ok {
		t.Error("message hasn't traveled")
		return
	}

	switch lr := len(result); lr {
	case 0, 1:
		t.Error("to few message handlings: ", lr)
		return
	case 2: // ok
	default:
		t.Error("too many message handlings:", lr)
		return
	}

	// position in the slice and point number
	// order must be 0->1, 1->2
	if result[0].Point != 1 || result[1].Point != 2 {
		t.Error("invalid handling order")
		return
	}

	// first: sent by feed and handled by subscriber
	if result[0].Outgoing != true {
		t.Error("firstly the message handled by feed")
	} else {
		// remote is feed
		// local is subscriber
		// check out public keys
		if result[0].RoutePoint.Remote.Pub != feed.PubKey() {
			t.Error("invalid public key of remote point")
		}
		if result[0].RoutePoint.Local.Pub != subscr.PubKey() {
			t.Error("invalid public key of local point")
		}
	}

	// second: sent by subscriber and handled by feed
	if result[1].Outgoing != false {
		t.Error("secondly the message handled by subscriber")
	} else {
		// remote is subscriber
		// local is feed
		// check out public keys
		if result[1].RoutePoint.Remote.Pub != subscr.PubKey() {
			t.Error("invalid public key of remote point")
		}
		if result[1].RoutePoint.Local.Pub != feed.PubKey() {
			t.Error("invalid public key of local point")
		}
	}
}

// There are two nodes: n1 an n2. The n1 is subscribed to n2.
// The n2 is subscriberd to n1. Both sends SendRecive message
// using broadcasting and get response back. That's all
func TestNode_crossSendReceive(t *testing.T) {
	sendReceiveStore.Reset() // reset
	var err error
	t1, t2 := newTestHook(),
		newTestHook()
	n1, n2 := mustCreateNode("feed"),
		mustCreateNode("subscr")
	for _, x := range []struct {
		node Node
		hook *testHook
	}{
		{n1, t1},
		{n2, t2},
	} {
		x.node.Register(SendReceive{}) //
		x.hook.Set(x.node)
		if err := x.node.Start(); err != nil {
			t.Error(err)
			return
		}
		defer x.node.Close()
	}
	// subscribe
	// n1 to n2
	err = n1.Outgoing().Connect(nodeMustGetAddress(n2, t), cipher.PubKey{})
	if err != nil {
		t.Error(err)
		return
	}
	// n2 to n1
	err = n2.Outgoing().Connect(nodeMustGetAddress(n1, t), cipher.PubKey{})
	if err != nil {
		t.Error(err)
		return
	}

	// waiting connections
	if !t1.estIncTimeout(1, time.Second, t) ||
		!t1.estOutTimeout(1, time.Second, t) ||
		!t2.estIncTimeout(1, time.Second, t) ||
		!t2.estOutTimeout(1, time.Second, t) {
		return
	}

	// create messages
	sr1, sr2 := sendReceiveStore.New(), sendReceiveStore.New()

	// broadcast SendReceive message
	// n1
	if err = n1.Incoming().Broadcast(sr1); err != nil {
		t.Error("broadcasting error: ", err)
		return
	}
	// n2
	if err = n2.Incoming().Broadcast(sr2); err != nil {
		t.Error("broadcasting error: ", err)
		return
	}

	// waiting receiving
	if !t1.recvOutTimeout(1, time.Second, t) ||
		!t1.recvIncTimeout(1, time.Second, t) ||
		!t2.recvOutTimeout(1, time.Second, t) ||
		!t2.recvIncTimeout(1, time.Second, t) {
		return
	}

	result1, ok := sendReceiveStore.Get(sr1.Id)
	if !ok {
		t.Error("message n1->n2 hasn't traveled")
		return
	}

	result2, ok := sendReceiveStore.Get(sr2.Id)
	if !ok {
		t.Error("message n2->n1 hasn't traveled")
		return
	}

	// =========================================================================

	//
	// TODO: DRY, motherfucker, DRY!
	//

	//
	// n1->n2->n1
	//

	switch lr := len(result1); lr {
	case 0, 1:
		t.Error("to few message handlings: ", lr)
		return
	case 2: // ok
	default:
		t.Error("too many message handlings:", lr)
		return
	}

	// position in the slice and point number
	// order must be 0->1, 1->2
	if result1[0].Point != 1 || result1[1].Point != 2 {
		t.Error("invalid handling order")
		return
	}

	// first: sent by n1 and handled by n2
	if result1[0].Outgoing != true {
		t.Error("firstly the message handled by n1")
	} else {
		// remote is n1
		// local is n2
		// check out public keys
		if result1[0].RoutePoint.Remote.Pub != n1.PubKey() {
			t.Error("invalid public key of remote point")
		}
		if result1[0].RoutePoint.Local.Pub != n2.PubKey() {
			t.Error("invalid public key of local point")
		}
	}

	// second: sent by n2 and handled by n1
	if result1[1].Outgoing != false {
		t.Error("secondly the message handled by n2")
	} else {
		// remote is n2
		// local is n1
		// check out public keys
		if result1[1].RoutePoint.Remote.Pub != n2.PubKey() {
			t.Error("invalid public key of remote point")
		}
		if result1[1].RoutePoint.Local.Pub != n1.PubKey() {
			t.Error("invalid public key of local point")
		}
	}

	//
	// n2->n1->n1
	//

	switch lr := len(result2); lr {
	case 0, 1:
		t.Error("to few message handlings: ", lr)
		return
	case 2: // ok
	default:
		t.Error("too many message handlings:", lr)
		return
	}

	// position in the slice and point number
	// order must be 0->1, 1->2
	if result2[0].Point != 1 || result2[1].Point != 2 {
		t.Error("invalid handling order")
		return
	}

	// first: sent by n2 and handled by n1
	if result2[0].Outgoing != true {
		t.Error("firstly the message handled by n2")
	} else {
		// remote is n2
		// local is n1
		// check out public keys
		if result2[0].RoutePoint.Remote.Pub != n2.PubKey() {
			t.Error("invalid public key of remote point")
		}
		if result2[0].RoutePoint.Local.Pub != n1.PubKey() {
			t.Error("invalid public key of local point")
		}
	}

	// second: sent by n1 and handled by n2
	if result2[1].Outgoing != false {
		t.Error("secondly the message handled by n1")
	} else {
		// remote is n1
		// local is n2
		// check out public keys
		if result2[1].RoutePoint.Remote.Pub != n1.PubKey() {
			t.Error("invalid public key of remote point")
		}
		if result2[1].RoutePoint.Local.Pub != n2.PubKey() {
			t.Error("invalid public key of local point")
		}
	}

	// =========================================================================

}

func TestNode_forwarding(t *testing.T) {
	var err error
	src, pipe, dst := mustCreateNode("src"),
		mustCreateNode("pipe"),
		mustCreateNode("dst")
	ts, tp, td := newTestHook(),
		newTestHook(),
		newTestHook()
	for _, x := range []struct {
		node Node
		hook *testHook
	}{
		{src, ts},
		{pipe, tp},
		{dst, td},
	} {
		x.node.Register(Forward{})
		x.hook.Set(x.node)
		if err := x.node.Start(); err != nil {
			t.Error(err)
			return
		}
		defer x.node.Close()
	}
	// src -> pipe -> dst
	// subscribe pipe to src
	err = pipe.Outgoing().Connect(
		nodeMustGetAddress(src, t),
		cipher.PubKey{})
	if err != nil {
		t.Error(err)
		return
	}
	// subscribe dst to pipe
	err = dst.Outgoing().Connect(
		nodeMustGetAddress(pipe, t),
		cipher.PubKey{})
	if err != nil {
		t.Error(err)
		return
	}
	// waiting for handshake
	if !ts.estIncTimeout(1, time.Second, t) { // incoming to src from pipe
		return
	}
	if !tp.estOutTimeout(1, time.Second, t) { // outgoing from pipe to src
		return
	}
	if !tp.estIncTimeout(1, time.Second, t) { // incoming to pipe from dst
		return
	}
	if !td.estOutTimeout(1, time.Second, t) { // outgoign to pipe from dst
		return
	}
	// forward
	forward := forwardStore.New()
	// send forward
	if err := src.Incoming().Broadcast(forward); err != nil {
		t.Error(err)
		return
	}
	// waiting for receive (src->pipe)
	if !tp.recvOutTimeout(1, time.Second, t) {
		t.Error("tp")
		//return
	}
	// waiting for receive (pipe->dst)
	if !td.recvOutTimeout(1, time.Second, t) {
		t.Error("td")
		//return
	}
	// check out routes
	pretty.Println(forwardStore.Get(forward.Id))
}
