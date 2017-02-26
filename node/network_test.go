package node

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

func init() {
	gnet.DebugPrint = testing.Verbose()
}

func wait() {
	time.Sleep(500 * time.Millisecond)
}

type pingPongCounter struct {
	sync.Mutex
	ping int
	pong int
}

func (p *pingPongCounter) Ping() {
	p.Lock()
	defer p.Unlock()
	p.ping++
}

func (p *pingPongCounter) Pong() {
	p.Lock()
	defer p.Unlock()
	p.pong++
}

func (p *pingPongCounter) Get() (ping, pong int) {
	p.Lock()
	defer p.Unlock()
	ping, pong = p.ping, p.pong
	return
}

func (p *pingPongCounter) reset() {
	p.ping = 0
	p.pong = 0
}

var gPingPong pingPongCounter

type String struct {
	Value string
}

// feed side (feed sends PIND)
func (s *String) HandleIncoming(ic IncomingContext) (terminate error) {
	gPingPong.Ping()
	if testing.Verbose() {
		fmt.Println(s.Value)
	}
	return
}

// subscriber side (subscriber replies PONG)
func (s *String) HandleOutgoing(oc OutgoingContext) (terminate error) {
	gPingPong.Pong()
	if testing.Verbose() {
		fmt.Println(s.Value)
	}
	return oc.Reply(&String{"PONG"})
}

func TestNode_send_receive(t *testing.T) {
	var (
		th1, th2 *testHook = newTestHook(), newTestHook()
		n1, n2   Node
		c        Config = newConfig()

		err error
	)
	c.Name = "feed"
	if n1, err = NewNode(secretKey(), c); err != nil {
		t.Fatal(err)
	}
	th1.Set(n1) // set test hook to node
	c.Name = "subsrciber"
	c.MaxIncomingConnections = 0 // suppress listening
	if n2, err = NewNode(secretKey(), c); err != nil {
		t.Fatal(err)
	}
	th2.Set(n2) // set test hook to node
	//
	n1.Register(String{})
	if err := n1.Start(); err != nil {
		t.Error("error staring node: ", err)
		return
	}
	defer n1.Close()
	n2.Register(String{})
	if err := n2.Start(); err != nil {
		t.Error("error staring node: ", err)
		return
	}
	defer n2.Close()
	// subscribe n2 to n1
	if addr, err := n1.Incoming().Address(); err != nil {
		t.Fatal("can't get listening address of node: ", err)
	} else {
		n1.Debug("[TEST] listening address: ", addr)
		if err := n2.Outgoing().Connect(addr, cipher.PubKey{}); err != nil {
			t.Fatal("error conection to remote node: ", err)
		}
	}

	if !th1.estIncTimeout(1, time.Second, t) {
		return
	}
	if !th2.estOutTimeout(1, time.Second, t) {
		return
	}

	gPingPong.reset() // reset global ping-pong counter

	if err = n1.Incoming().Broadcast(&String{"PING"}); err != nil {
		t.Error("broadcasting error: ", err)
	}

	if !th2.recvOutTimeout(1, time.Second, t) {
		return
	}

	if !th1.recvIncTimeout(1, time.Second, t) {
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
	var (
		th1, th2 *testHook = newTestHook(), newTestHook()
		n1, n2   Node
		c        Config = newConfig()

		err error
	)

	c.Name = "feed"
	if n1, err = NewNode(secretKey(), c); err != nil {
		t.Fatal(err)
	}
	th1.Set(n1)
	c.Name = "subsrciber"
	if n2, err = NewNode(secretKey(), c); err != nil {
		t.Fatal(err)
	}
	th2.Set(n2)

	//for _, n := range []Node{n1, n2} { // <- stupid, doesn't work
	n1.Register(String{})
	if err := n1.Start(); err != nil {
		t.Error("error staring node: ", err)
		return
	}
	defer n1.Close()
	n2.Register(String{})
	if err := n2.Start(); err != nil {
		t.Error("error staring node: ", err)
		return
	}
	defer n2.Close()

	// subscribe n2 to n1
	if addr, err := n1.Incoming().Address(); err != nil {
		t.Fatal("can't get listening address of node: ", err)
	} else {
		n1.Debug("[TEST] listening address: ", addr)
		if err := n2.Outgoing().Connect(addr, cipher.PubKey{}); err != nil {
			t.Fatal("error conection to remote node: ", err)
		}
	}

	// subscribe n1 to n2
	if addr, err := n2.Incoming().Address(); err != nil {
		t.Fatal("can't get listening address of node: ", err)
	} else {
		n2.Debug("[TEST] listening address: ", addr)
		if err := n1.Outgoing().Connect(addr, cipher.PubKey{}); err != nil {
			t.Fatal("error conection to remote node: ", err)
		}
	}

	if !th1.estIncTimeout(1, time.Second, t) ||
		!th1.estOutTimeout(1, time.Second, t) ||
		!th2.estIncTimeout(1, time.Second, t) ||
		!th2.estOutTimeout(1, time.Second, t) {
		return
	}

	gPingPong.reset() // reset global ping-pong counter

	if err = n1.Incoming().Broadcast(&String{"PING"}); err != nil {
		t.Error("broadcasting error: ", err)
	}
	if err = n2.Incoming().Broadcast(&String{"PING"}); err != nil {
		t.Error("broadcasting error: ", err)
	}

	if !(th1.recvIncTimeout(1, time.Second, t) &&
		th1.recvOutTimeout(1, time.Second, t) &&
		th2.recvIncTimeout(1, time.Second, t) &&
		th2.recvOutTimeout(1, time.Second, t)) {
		return
	}

	if gPingPong.ping != 2 {
		t.Error("wrong ping counter: ", gPingPong.ping)
	}

	if gPingPong.pong != 2 {
		t.Error("wrong pong counter: ", gPingPong.ping)
	}

}
