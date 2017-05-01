package gnet

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestConnState_String(t *testing.T) {
	for _, x := range []struct {
		cs ConnState
		st string
	}{
		{ConnStateConnected, "CONNECTED"},
		{ConnStateDialing, "DIALING"},
		{ConnStateClosed, "CLOSED"},
		{ConnStateClosed + 1, fmt.Sprintf("ConnState<%d>", ConnStateClosed+1)},
		{-1, fmt.Sprintf("ConnState<%d>", -1)},
	} {
		if got := x.cs.String(); got != x.st {
			t.Errorf("wrong string: want %q, got %q", x.st, got)
		}
	}
}

func pair() (l, d *Pool, dc, lc *Conn, err error) {
	if d, err = NewPool(newConfig("dialer")); err != nil {
		return
	}
	var (
		lconf    Config     = newConfig("listener")
		accepted chan *Conn = make(chan *Conn, 1)
	)
	lconf.ConnectionHandler = func(c *Conn) { accepted <- c }
	if l, err = NewPool(lconf); err != nil {
		d.Close()
		return
	}
	if err = l.Listen(""); err != nil {
		d.Close()
		l.Close()
		return
	}
	if dc, err = d.Dial(l.Address()); err != nil {
		d.Close()
		l.Close()
	}
	select {
	case lc = <-accepted:
	case <-time.After(TM):
		d.Close()
		l.Close()
		err = errors.New("slow accepting (connection handler)")
	}
	return
}

func TestConn_Value(t *testing.T) {
	l, d, dc, _, err := pair()
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	defer d.Close()

	dc.SetValue(50)
	x := d.Connection(dc.Address())
	if x == nil {
		t.Error("missing connection")
		return
	}
	if val := x.Value(); val == nil {
		t.Error("missing value")
	} else if v, ok := val.(int); !ok {
		t.Errorf("wrong type of value %T", val)
	} else if v != 50 {
		t.Error("wrong value", v)
	}
}

func TestConn_SetValue(t *testing.T) {
	// see Value
}

func TestConn_LastRead(t *testing.T) {
	//
}

func TestConn_LastWrite(t *testing.T) {
	//
}

func TestConn_SendQueue(t *testing.T) {
	//
}

func TestConn_ReceiveQueue(t *testing.T) {
	//
}

func TestConn_Address(t *testing.T) {
	//
}

func TestConn_IsIncoming(t *testing.T) {
	//
}

func TestConn_State(t *testing.T) {
	//
}

func TestConn_Close(t *testing.T) {
	//
}

func TestConn_Closed(t *testing.T) {
	//
}
