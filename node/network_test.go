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

func TestNode_send_receive(t *testing.T) {
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
		x.node.Register(String{})
		x.hook.Set(x.node)
		if err := x.node.Start(); err != nil {
			t.Error(err)
			return
		}
		defer x.node.Close()
	}
	// subscribe to feed
	if addr, err := feed.Incoming().Address(); err != nil {
		t.Error("can't get listening address of node: ", err)
		return
	} else {
		feed.Debug("[TEST] listening address: ", addr)
		if err := subscr.Outgoing().Connect(addr, cipher.PubKey{}); err != nil {
			t.Error("error conection to remote node: ", err)
			return
		}
	}

	if !tf.estIncTimeout(1, time.Second, t) {
		return
	}
	if !ts.estOutTimeout(1, time.Second, t) {
		return
	}

	gPingPong.reset() // reset global ping-pong counter

	if err = feed.Incoming().Broadcast(&String{"PING"}); err != nil {
		t.Error("broadcasting error: ", err)
	}

	if !ts.recvOutTimeout(1, time.Second, t) {
		return
	}

	if !tf.recvIncTimeout(1, time.Second, t) {
		return
	}

	ping, pong := gPingPong.Get()

	if ping != 1 {
		t.Error("wrong ping counter: want 1, got ", ping)
	}

	if pong != 1 {
		t.Error("wrong pong counter: want 1, got ", ping)
	}

}

func TestNode_cross_send_receive(t *testing.T) {
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
		x.node.Register(String{})
		x.hook.Set(x.node)
		if err := x.node.Start(); err != nil {
			t.Error(err)
			return
		}
		defer x.node.Close()
	}

	// subscribe to feed
	if addr, err := feed.Incoming().Address(); err != nil {
		t.Fatal("can't get listening address of node: ", err)
	} else {
		feed.Debug("[TEST] listening address: ", addr)
		if err := subscr.Outgoing().Connect(addr, cipher.PubKey{}); err != nil {
			t.Fatal("error conection to remote node: ", err)
		}
	}

	// subscribe to subscr
	if addr, err := subscr.Incoming().Address(); err != nil {
		t.Fatal("can't get listening address of node: ", err)
	} else {
		subscr.Debug("[TEST] listening address: ", addr)
		if err := feed.Outgoing().Connect(addr, cipher.PubKey{}); err != nil {
			t.Fatal("error conection to remote node: ", err)
		}
	}

	if !tf.estIncTimeout(1, time.Second, t) ||
		!tf.estOutTimeout(1, time.Second, t) ||
		!ts.estIncTimeout(1, time.Second, t) ||
		!ts.estOutTimeout(1, time.Second, t) {
		return
	}

	gPingPong.reset() // reset global ping-pong counter

	if err = feed.Incoming().Broadcast(&String{"PING"}); err != nil {
		t.Error("broadcasting error: ", err)
	}
	if err = subscr.Incoming().Broadcast(&String{"PING"}); err != nil {
		t.Error("broadcasting error: ", err)
	}

	if !(tf.recvIncTimeout(1, time.Second, t) &&
		tf.recvOutTimeout(1, time.Second, t) &&
		ts.recvIncTimeout(1, time.Second, t) &&
		ts.recvOutTimeout(1, time.Second, t)) {
		return
	}

	if gPingPong.ping != 2 {
		t.Error("wrong ping counter: ", gPingPong.ping)
	}

	if gPingPong.pong != 2 {
		t.Error("wrong pong counter: ", gPingPong.ping)
	}

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
