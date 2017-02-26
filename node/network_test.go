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

func TestNode_cross_send_receive(t *testing.T) {
	// var err error
	// tf, ts := newTestHook(), newTestHook()
	// feed, subscr := mustCreateNode("feed"), mustCreateNode("subscr")
	// for _, x := range []struct {
	// 	node Node
	// 	hook *testHook
	// }{
	// 	{feed, tf},
	// 	{subscr, ts},
	// } {
	// 	x.node.Register(String{})
	// 	x.hook.Set(x.node)
	// 	if err := x.node.Start(); err != nil {
	// 		t.Error(err)
	// 		return
	// 	}
	// 	defer x.node.Close()
	// }

	// // subscribe to feed
	// if addr, err := feed.Incoming().Address(); err != nil {
	// 	t.Fatal("can't get listening address of node: ", err)
	// } else {
	// 	feed.Debug("[TEST] listening address: ", addr)
	// 	if err := subscr.Outgoing().Connect(addr, cipher.PubKey{}); err != nil {
	// 		t.Fatal("error conection to remote node: ", err)
	// 	}
	// }

	// // subscribe to subscr
	// if addr, err := subscr.Incoming().Address(); err != nil {
	// 	t.Fatal("can't get listening address of node: ", err)
	// } else {
	// 	subscr.Debug("[TEST] listening address: ", addr)
	// 	if err := feed.Outgoing().Connect(addr, cipher.PubKey{}); err != nil {
	// 		t.Fatal("error conection to remote node: ", err)
	// 	}
	// }

	// if !tf.estIncTimeout(1, time.Second, t) ||
	// 	!tf.estOutTimeout(1, time.Second, t) ||
	// 	!ts.estIncTimeout(1, time.Second, t) ||
	// 	!ts.estOutTimeout(1, time.Second, t) {
	// 	return
	// }

	// gPingPong.reset() // reset global ping-pong counter

	// if err = feed.Incoming().Broadcast(&String{"PING"}); err != nil {
	// 	t.Error("broadcasting error: ", err)
	// }
	// if err = subscr.Incoming().Broadcast(&String{"PING"}); err != nil {
	// 	t.Error("broadcasting error: ", err)
	// }

	// if !(tf.recvIncTimeout(1, time.Second, t) &&
	// 	tf.recvOutTimeout(1, time.Second, t) &&
	// 	ts.recvIncTimeout(1, time.Second, t) &&
	// 	ts.recvOutTimeout(1, time.Second, t)) {
	// 	return
	// }

	// if gPingPong.ping != 2 {
	// 	t.Error("wrong ping counter: ", gPingPong.ping)
	// }

	// if gPingPong.pong != 2 {
	// 	t.Error("wrong pong counter: ", gPingPong.ping)
	// }

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
