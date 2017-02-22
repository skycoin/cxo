package node

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

// Subscriptions
type Inflow interface {
	// List subscriptions
	List(ListFunc)
	// Terminate subscription by its public key. It can return ErrNotFound
	Terminate(cipher.PubKey) error
	// Subscribe to given address
	Subscribe(string, cipher.PubKey) error
}

type inflow node

func (i *inflow) node() *node { return (*node)(i) }

func (i *inflow) List(fn ListFunc) {
	if fn == nil {
		i.Panic("call (Inflow).List with nil")
	}
	i.backmx.RLock()
	for gc, c := range i.back {
		if c.isPending() || c.isFeed() {
			continue
		}
		i.backmx.RUnlock()
		if fn(c.pub, gc.Conn.RemoteAddr()) {
			return
		}
		i.backmx.RLock()
	}
	i.backmx.RUnlock()
}

func (i *inflow) Terminate(pub cipher.PubKey) error {
	i.Debug("terminate ", pub.Hex())
	i.backmx.RLock()
	defer i.backmx.RUnlock()
	for gc, c := range i.back {
		if c.isPending() || c.isFeed() || c.pub != pub {
			continue
		}
		i.pool.Disconnect(gc, ErrManualDisconnect)
		return nil
	}
	return ErrNotFound
}

func (i *inflow) Subscribe(address string, dpub cipher.PubKey) (err error) {
	if dpub == (cipher.PubKey{}) {
		i.Debug("subscribe ", address)
	} else {
		i.Debugf("subscribe %s with desired public key: ",
			address,
			dpub.Hex())
	}
	var (
		gc *gnet.Connection
		c  *conn
	)
	if gc, err = i.pool.Connect(address); err != nil {
		i.Debug("subscribe error:, ", err)
		return err
	}
	c = &conn{
		pub:  dpub,
		done: make(chan struct{}),
	}
	i.backmx.Lock()
	defer i.backmx.Unlock()
	i.back[gc] = c
	if i.conf.HandshakeTimeout > 0 {
		go i.node().countDownHandshakeTimeout(gc, c.done)
	}
	i.pool.SendMessage(gc, c.Question(i.node()))
	return
}
