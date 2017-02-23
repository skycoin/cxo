package node

import (
	"fmt"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

func init() {
	gnet.DebugPrint = testing.Verbose()
}

func wait() {
	time.Sleep(1000 * time.Millisecond)
}

type pingPongCounter struct {
	pingc int
	pongc int
}

func (p *pingPongCounter) reset() {
	p.pingc = 0
	p.pongc = 0
}

func (p *pingPongCounter) ping() {
	p.pingc++
}

func (p *pingPongCounter) pong() {
	p.pongc++
}

var gPingPong pingPongCounter

type String struct {
	Value string
}

// feed side
func (s *String) HandleIncoming(ic IncomingContext) (terminate error) {
	gPingPong.pong()
	if testing.Verbose() {
		fmt.Println(s.Value)
	}
	return
}

// subscriber side
func (s *String) HandleOutgoing(oc OutgoingContext) (terminate error) {
	gPingPong.ping()
	if testing.Verbose() {
		fmt.Println(s.Value)
	}
	return oc.Reply(&String{"PONG"})
}

func TestNode_ping_pong(t *testing.T) {
	var (
		n1, n2 Node
		c      Config = newConfig()

		err error
	)

	c.Name = "feed"
	if n1, err = NewNode(secretKey(), c); err != nil {
		t.Fatal(err)
	}
	c.Name = "subsrciber"
	c.MaxIncomingConnections = 0 // suppress listening
	if n2, err = NewNode(secretKey(), c); err != nil {
		t.Fatal(err)
	}

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

	wait()

	// subscribe n2 to n1
	if addr, err := n1.Incoming().Address(); err != nil {
		t.Fatal("can't get listening address of node: ", err)
	} else {
		n1.Debug("[TEST] listening address: ", addr)
		if err := n2.Outgoing().Connect(addr, cipher.PubKey{}); err != nil {
			t.Fatal("error conection to remote node: ", err)
		}
	}

	wait()

	gPingPong.reset() // reset global ping-pong counter

	if err = n1.Incoming().Broadcast(&String{"PING"}); err != nil {
		t.Error("broadcasting error: ", err)
	}

	wait()

	if gPingPong.pingc != 1 {
		t.Error("wrong ping counter: ", gPingPong.pingc)
	}

	if gPingPong.pongc != 1 {
		t.Error("wrong pong counter: ", gPingPong.pingc)
	}

}
