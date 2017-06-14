package gnet

import (
	"crypto/tls"
	"io/ioutil"
	stdlog "log"
	"net"
	"testing"
	"time"

	"github.com/skycoin/cxo/node/log"
)

const TM time.Duration = 100 * time.Millisecond

// helper variables
var (
	tlsc = &tls.Config{InsecureSkipVerify: true}
)

func newConfig(name string) (c Config) {
	if name == "" {
		name = "test"
	}
	c = NewConfig()
	c.ReadTimeout = 0
	c.WriteTimeout = 0
	c.DialTimeout = TM // make it shorter
	c.RedialTimeout = 0
	if testing.Verbose() {
		c.Logger = log.NewLogger("["+name+"] ", true)
		c.Logger.SetFlags(stdlog.Lshortfile | stdlog.Lmicroseconds)
	} else {
		c.Logger = log.NewLogger("["+name+"] ", false)
		c.Logger.SetOutput(ioutil.Discard)
	}
	return
}

// helper functions
func dial(t *testing.T, address string) (c net.Conn) {
	var err error
	if c, err = net.DialTimeout("tcp", address, TM); err != nil {
		t.Fatal(err)
	}
	return
}

func readChan(t *testing.T, c chan struct{}, fatal string) {
	select {
	case <-c:
	case <-time.After(TM):
		t.Fatal(fatal)
	}
}

func TestNewPool(t *testing.T) {
	t.Run("invalid config", func(t *testing.T) {
		if _, err := NewPool(Config{MaxConnections: -1}); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("logger", func(t *testing.T) {
		p, err := NewPool(NewConfig())
		if err != nil {
			t.Fatal(err)
		}
		if p.Logger == nil {
			t.Error("logger doen't created")
		}
		c := NewConfig()
		c.Logger = log.NewLogger("[asdf]", false)
		if p, err = NewPool(c); err != nil {
			t.Fatal(err)
		}
		if p.Logger != c.Logger {
			t.Error("another logger used")
		}
	})
}

func TestPool_Listen(t *testing.T) {
	t.Run("connect", func(t *testing.T) {
		connect := make(chan struct{})
		conf := newConfig("")
		conf.OnCreateConnection = func(*Conn) { connect <- struct{}{} }
		p, err := NewPool(conf)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()
		if err := p.Listen(""); err != nil {
			t.Error(err)
			return
		}
		if p.l == nil {
			t.Error("missing listener in Pool struct")
			return
		}
		c, err := net.DialTimeout("tcp", p.l.Addr().String(), TM)
		if err != nil {
			t.Error("error dialtig to the Pool")
			return
		}
		defer c.Close()
		readChan(t, connect, "slow connectiing")
	})
	t.Run("limit", func(t *testing.T) {
		//
	})
}

func TestPool_Address(t *testing.T) {
	//
}

func TestPool_Connections(t *testing.T) {
	//
}

func TestPool_Connection(t *testing.T) {
	//
}

func TestPool_Dial(t *testing.T) {
	//
}

func TestPool_Close(t *testing.T) {
	//
}

func TestPool_sendReceive(t *testing.T) {
	l, d, lc, dc, err := pair()
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	defer d.Close()

	want := []byte("yo-ho-ho!")

	// twice
	for i := 0; i < 2; i++ {

		select {
		case lc.SendQueue() <- want:
		case <-time.After(TM):
			t.Error("slow")
			return
		}
		select {
		case got := <-dc.ReceiveQueue():
			if string(got) != string(want) {
				t.Error("wrong message received:", string(got))
				return
			}
		case <-time.After(TM):
			t.Error("slow")
			return
		}

	}
}

func TestPool_receiveSend(t *testing.T) {
	l, d, lc, dc, err := pair()
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	defer d.Close()

	want := []byte("yo-ho-ho!")

	// twice
	for i := 0; i < 2; i++ {

		select {
		case dc.SendQueue() <- want:
		case <-time.After(TM):
			t.Error("slow")
			return
		}

		select {
		case got := <-lc.ReceiveQueue():
			if string(got) != string(want) {
				t.Error("wrong message received:", string(got))
				return
			}
		case <-time.After(TM):
			t.Error("slow")
			return
		}

	}

}

func closeRead(c *Conn) {
	c.cmx.Lock()
	defer c.cmx.Unlock()

	c.conn.(*net.TCPConn).CloseRead()
}

func closeWrite(c *Conn) {
	c.cmx.Lock()
	defer c.cmx.Unlock()

	c.conn.(*net.TCPConn).CloseWrite()
}

func sendTM(t *testing.T, c *Conn, msg string) {
	select {
	case c.SendQueue() <- []byte(msg):
	case <-time.After(TM):
		t.Error("slow")
	}
}

func receiveTM(t *testing.T, c *Conn, want string) {
	select {
	case msg := <-c.ReceiveQueue():
		if string(msg) != want {
			t.Error("wring message received")
		}
	case <-time.After(TM * 20):
		t.Error("slow")
	}
}

func sendReceiveDuplex(t *testing.T, cc, sc *Conn) {
	sendTM(t, cc, "hey")
	receiveTM(t, sc, "hey")
	sendTM(t, sc, "ho")
	receiveTM(t, cc, "ho")
}

func TestPool_redialRead(t *testing.T) {
	cconf := newConfig("CLIENT")
	// cconf.OnDial = func(*Conn, error) (_ error) {
	// 	t.Log("OnDial")
	// 	return
	// }

	client, err := NewPool(cconf)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	accept := make(chan *Conn, 1)

	sconf := newConfig("SERVER")
	sconf.OnCreateConnection = func(c *Conn) {
		accept <- c
	}

	server, err := NewPool(sconf)
	if err != nil {
		t.Error(err)
		return
	}
	defer server.Close()

	if err := server.Listen("127.0.0.1:0"); err != nil {
		t.Error(err)
		return
	}

	cc, err := client.Dial(server.Address())
	if err != nil {
		t.Error(err)
		return
	}

	select {
	case sc := <-accept:
		// test conenction
		sendReceiveDuplex(t, cc, sc)
		if t.Failed() {
			return
		}
		// trigger redialing
		closeWrite(sc) // break reading loop other side
	case <-time.After(TM):
		t.Error("slow")
		return
	}

	select {
	case sc := <-accept:
		t.Log("accepted again")
		// test conenction
		sendReceiveDuplex(t, cc, sc)
	case <-time.After(TM):
		t.Error("slow")
	}

}

func TestPool_redialWrite(t *testing.T) {
	cconf := newConfig("CLIENT")
	// cconf.OnDial = func(*Conn, error) (_ error) {
	// 	t.Log("OnDial")
	// 	return
	// }

	client, err := NewPool(cconf)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	accept := make(chan *Conn, 1)

	sconf := newConfig("SERVER")
	sconf.OnCreateConnection = func(c *Conn) {
		accept <- c
	}

	server, err := NewPool(sconf)
	if err != nil {
		t.Error(err)
		return
	}
	defer server.Close()

	if err := server.Listen("127.0.0.1:0"); err != nil {
		t.Error(err)
		return
	}

	cc, err := client.Dial(server.Address())
	if err != nil {
		t.Error(err)
		return
	}

	select {
	case sc := <-accept:
		// test conenction
		sendReceiveDuplex(t, cc, sc)
		if t.Failed() {
			return
		}
		// trigger redialing
		closeRead(sc) // break reading loop other side
	case <-time.After(TM * 100):
		t.Error("slow")
		return
	}

	select {
	case sc := <-accept:
		// test conenction
		sendReceiveDuplex(t, cc, sc)
	case <-time.After(TM * 100):
		t.Error("slow")
		return
	}

}
