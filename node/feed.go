package node

import (
	"net"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

func init() {
	// gnet.Message interface
	var _ gnet.Message = &Msg{}
	// Register handshake related messages (must be non-pointer)
	gnet.RegisterMessage(
		gnet.MessagePrefixFromString("DATA"),
		Msg{})

	gnet.VerifyMessages()
}

// Subscribers
type Feed interface {
	// List subscribers
	List(ListFunc)
	// Terminate subscriber by its public key. It can return ErrNotFound
	// if requested pubkey is not in feed
	Terminate(cipher.PubKey) error
	// Bradcast encoded object to subscribers
	Broadcast([]byte)
	// Address returns listening address or ErrNotListening
	Address() (net.Addr, error)
}

type feed node

func (f *feed) List(fn ListFunc) {
	if fn == nil {
		f.Panic("call (Feed).List with nil")
	}
	f.RLock()
	for gc, c := range f.back {
		if c.isPending() || !c.isFeed() {
			continue
		}
		f.RUnlock()
		if fn(c.pub, gc.Conn.RemoteAddr()) {
			return
		}
		f.RLock()
	}
	f.RUnlock()
}

func (f *feed) Terminate(pub cipher.PubKey) error {
	f.Debug("terminate ", pub.Hex())
	f.RLock()
	defer f.RUnlock()
	for gc, c := range f.back {
		if c.isPending() || !c.isFeed() || c.pub != pub {
			continue
		}
		f.pool.Disconnect(gc, ErrManualDisconnect)
		return nil
	}
	return ErrNotFound
}

func (f *feed) Broadcast(data []byte) {
	f.Debug("broadcast")
	f.RLock()
	defer f.RUnlock()
	for gc, c := range f.back {
		if c.isPending() || !c.isFeed() {
			continue
		}
		f.pool.SendMessage(gc, &Msg{
			Data: data,
		})
	}
}

func (f *feed) Address() (addr net.Addr, err error) {
	// temporary (skycoin/skycoin PR 290)
	f.RLock()
	defer f.RUnlock()
	return f.pool.ListeningAddress()
}

type Msg struct {
	Data []byte
}

func (m *Msg) Handle(ctx *gnet.MessageContext, ni interface{}) error {
	n := ni.(*node)
	n.Debug("(*Msg).Handle: got message")
	return n.conf.ReceiveCallback(m.Data, n, func(r []byte) {
		n.Debug("(*Msg).Handle: send reply")
		ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, &Msg{r})
	})
}
