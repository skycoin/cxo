package node

import (
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

func init() {
	gnet.DebugPrint = testing.Verbose()
}

// Send message from feed to single subscriber using broadcsting.
// The subscriber sends response back. That's all
func TestNode_sendReceive(t *testing.T) {
	srs := NewSendReciveStore()
	var err error
	tf, ts := newTestHook(), newTestHook()
	feed, subscr := mustCreateNode("feed", srs), mustCreateNode("subscr", srs)
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
	sr := srs.New()
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

	srs.Lookup(sr.Id, []ExpectedPoint{
		{Outgoing: true, Remote: feed.PubKey(), Local: subscr.PubKey()},
		{Outgoing: false, Remote: subscr.PubKey(), Local: feed.PubKey()},
	}, t)

}

// There are two nodes: n1 an n2. The n1 is subscribed to n2.
// The n2 is subscriberd to n1. Both sends SendRecive message
// using broadcasting and get response back. That's all
func TestNode_crossSendReceive(t *testing.T) {
	srs := NewSendReciveStore()
	var err error
	t1, t2 := newTestHook(),
		newTestHook()
	n1, n2 := mustCreateNode("n1", srs),
		mustCreateNode("n2", srs)
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
	sr1, sr2 := srs.New(), srs.New()

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

	// lookup sr1 (n1->n2->n1)
	srs.Lookup(sr1.Id, []ExpectedPoint{
		{Outgoing: true, Remote: n1.PubKey(), Local: n2.PubKey()},
		{Outgoing: false, Remote: n2.PubKey(), Local: n1.PubKey()},
	}, t)
	// lookup sr2 (n2->n1->n1)
	srs.Lookup(sr2.Id, []ExpectedPoint{
		{Outgoing: true, Remote: n2.PubKey(), Local: n1.PubKey()},
		{Outgoing: false, Remote: n1.PubKey(), Local: n2.PubKey()},
	}, t)

}
