package node

import (
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

func mustGetAddress(n Node, t *testing.T) string {
	addr, err := n.Feed().Address()
	if err != nil {
		t.Fatal(err)
	}
	return addr.String()
}

func newNetworkNode(name string, rc ReceiveCallback) Node {
	c := newConfig()
	_, sec := cipher.GenerateKeyPair()
	c.SecretKey = sec.Hex()
	c.Name = name
	c.ReceiveCallback = rc
	return NewNode(c)
}

// send PING from feed to subscriber and receive PONG back
func Test_ping_pong_simplex(t *testing.T) {
	var (
		n1, n2 Node

		pong1, ping2 int

		err error
	)

	SetDebugLog(false)

	n1 = newNetworkNode("node 1", func(d []byte, n Node, r ReplyFunc) error {
		if s := string(d); s == "PONG" {
			pong1++
			t.Log("got PONG")
		} else {
			t.Error("unexpected message received")
		}
		return nil
	})
	n2 = newNetworkNode("node 2", func(d []byte, n Node, r ReplyFunc) error {
		if s := string(d); s == "PING" {
			ping2++
			t.Log("got PING")
			r([]byte("PONG"))
		} else {
			t.Error("unexpected message received")
		}
		return nil
	})

	t.Log("starting nodes")
	if err = n1.Start(); err != nil {
		t.Fatal(err)
	}
	defer n1.Close()
	if err = n2.Start(); err != nil {
		t.Fatal(err)
	}
	defer n2.Close()
	//
	time.Sleep(1 * time.Second)
	//
	t.Log("n2 to n1 subscribe")
	err = n2.Inflow().Subscribe(mustGetAddress(n1, t), cipher.PubKey{})
	if err != nil {
		t.Fatal(err)
	}
	//
	time.Sleep(1 * time.Second)
	n1.Debug("BACK 1:", n1.(*node).back)
	n2.Debug("BACK 2:", n2.(*node).back)
	//
	// ping
	t.Log("ping-ping")
	n1.Feed().Broadcast([]byte("PING"))
	n1.Feed().Broadcast([]byte("PING"))
	time.Sleep(1 * time.Second)
	//
	if pong1 != 2 {
		t.Error("wrong pong1: ", pong1)
	}
	if ping2 != 2 {
		t.Error("wrong ping2: ", ping2)
	}
}

// Send PING and receive PONG from n1 to n2 and vice versa.
// Then send PONG without reply (n1->n2, n2->n1)
func Test_ping_pong_duplex(t *testing.T) {
	var (
		n1, n2 Node

		ping1, ping2 int

		pong1, pong2 int

		err error
	)

	SetDebugLog(false)

	n1 = newNetworkNode("node 1", func(d []byte, n Node, r ReplyFunc) error {
		if s := string(d); s == "PING" {
			ping1++
			t.Log("got PING")
			r([]byte("PONG"))
		} else if s == "PONG" {
			pong1++
			t.Log("got PONG")
		} else {
			t.Error("unexpected message received")
		}
		return nil
	})
	n2 = newNetworkNode("node 2", func(d []byte, n Node, r ReplyFunc) error {
		if s := string(d); s == "PING" {
			ping2++
			t.Log("got PING")
			r([]byte("PONG"))
		} else if s == "PONG" {
			pong2++
			t.Log("got PONG")
		} else {
			t.Error("unexpected message received")
		}
		return nil
	})

	t.Log("starting nodes")
	if err = n1.Start(); err != nil {
		t.Fatal(err)
	}
	defer n1.Close()
	if err = n2.Start(); err != nil {
		t.Fatal(err)
	}
	defer n2.Close()
	//
	time.Sleep(1 * time.Second)
	//
	t.Log("cross-subscribe")
	err = n1.Inflow().Subscribe(mustGetAddress(n2, t), cipher.PubKey{})
	if err != nil {
		t.Fatal(err)
	}
	err = n2.Inflow().Subscribe(mustGetAddress(n1, t), cipher.PubKey{})
	if err != nil {
		t.Fatal(err)
	}
	//
	time.Sleep(1 * time.Second)
	n1.Debug("BACK 1:", n1.(*node).back)
	n2.Debug("BACK 2:", n2.(*node).back)
	//
	// ping
	t.Log("ping-ping")
	n1.Feed().Broadcast([]byte("PING"))
	n2.Feed().Broadcast([]byte("PING"))
	// pong
	t.Log("pong-pong")
	n1.Feed().Broadcast([]byte("PONG"))
	n2.Feed().Broadcast([]byte("PONG"))
	//
	time.Sleep(1 * time.Second)
	//
	if ping1 != 1 {
		t.Error("wrong ping1: ", ping1)
	}
	if ping2 != 1 {
		t.Error("wrong ping1: ", ping2)
	}
	if pong1 != 2 {
		t.Error("wrong pong1: ", pong1)
	}
	if pong2 != 2 {
		t.Error("wrong pong1: ", pong2)
	}
}

// Send PING from feed to many subscribers
// and receive PONG back from every subscriber
func Test_ping_pong_one_to_multiply(t *testing.T) {
	var (
		feed           Node
		s1, s2, s3, s4 Node
		ping, pong     int

		src ReceiveCallback

		err error
	)

	SetDebugLog(false)

	src = func(d []byte, n Node, r ReplyFunc) error {
		if s := string(d); s == "PING" {
			ping++
			r([]byte("PONG"))
		} else {
			t.Error("unexpected message received")
		}
		return nil
	}

	feed = newNetworkNode("feed", func(d []byte, n Node, r ReplyFunc) error {
		if s := string(d); s == "PONG" {
			pong++
		} else {
			t.Error("unexpected message received")
		}
		return nil
	})

	s1 = newNetworkNode("s1", src)
	s2 = newNetworkNode("s2", src)
	s3 = newNetworkNode("s3", src)
	s4 = newNetworkNode("s4", src)

	t.Log("starting nodes")
	for _, n := range []Node{feed, s1, s2, s3, s4} {
		if err = n.Start(); err != nil {
			t.Fatal(err)
		}
		defer n.Close()
	}
	time.Sleep(1 * time.Second)
	//
	t.Log("subscribe")
	for _, s := range []Node{s1, s2, s3, s4} {
		err = s.Inflow().Subscribe(mustGetAddress(feed, t), cipher.PubKey{})
		if err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(1 * time.Second)
	//
	for _, n := range []Node{feed, s1, s2, s3, s4} {
		n.Debugf("BACK %s: %v", n.(*node).conf.Name, n.(*node).back)
	}
	//
	// ping
	t.Log("ping")
	feed.Feed().Broadcast([]byte("PING"))
	time.Sleep(1 * time.Second)
	//
	if ping != 4 {
		t.Error("wrong ping: ", ping)
	}
	if pong != 4 {
		t.Error("wrong pong: ", pong)
	}
}
