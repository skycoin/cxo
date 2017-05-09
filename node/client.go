package node

import (
	"errors"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
)

// An Event represents client event
type Event func(*gnet.Conn) (terminate error)

// A Client represnets CXO client
type Client struct {
	log.Logger

	so *skyobject.Container

	fmx   sync.Mutex
	feeds []cipher.PubKey // subscriptions

	smx   sync.Mutex
	srvfs []cipher.PubKey // server feeds

	conf ClientConfig
	pool *gnet.Pool

	cn *gnet.Conn

	quito sync.Once
	quit  chan struct{}
	await sync.WaitGroup
}

// NewClient cretes Client with given Container
func NewClient(cc ClientConfig, so *skyobject.Container) (c *Client,
	err error) {

	if so == nil {
		panic("nil so")
	}

	c = new(Client)
	c.so = so
	c.feeds = nil
	c.srvfs = nil
	c.Logger = log.NewLogger(cc.Log.Prefix, cc.Log.Debug)
	cc.Config.Logger = c.Logger // use the same logger

	cc.Config.ConnectionHandler = c.connectHandler
	cc.Config.DisconnectHandler = c.disconnectHandler

	c.conf = cc

	if c.pool, err = gnet.NewPool(cc.Config); err != nil {
		c = nil
		return
	}

	c.quit = make(chan struct{})

	return
}

func (c *Client) Start(address string) (err error) {
	c.Debug("starting client of ", address)

	var cn *gnet.Conn
	if cn, err = c.pool.Dial(address); err != nil {
		return
	}
	c.cn = cn // keep connection
	c.await.Add(1)
	go c.handle(cn)
	return
}

// Close client
func (c *Client) Close() (err error) {
	c.Debug("closing client")

	c.quito.Do(func() {
		close(c.quit)
	})
	err = c.pool.Close()
	c.await.Wait()
	return
}

func (c *Client) connectHandler(cn *gnet.Conn) {
	c.Debug("connected to ", cn.Address())
}

func (c *Client) disconnectHandler(cn *gnet.Conn) {
	c.Debug("disconnected from ", cn.Address())
}

func (c *Client) handle(cn *gnet.Conn) {
	defer c.await.Done()

	var (
		receive <-chan []byte   = cn.ReceiveQueue()
		closed  <-chan struct{} = cn.Closed()

		data []byte
		msg  Msg

		err error
	)

	// events loop
	for {
		select {
		case <-closed:
			return
		case data = <-receive:
			if msg, err = Decode(data); err != nil {
				c.Print("[ERR] error decoding message: ", err)
				cn.Close()
				return
			}
			c.handleMessage(cn, msg)
		}
	}

}

func (c *Client) addServerFeed(feed cipher.PubKey) {
	c.smx.Lock()
	defer c.smx.Unlock()

	for _, f := range c.srvfs {
		if f == feed {
			return // already have the feed
		}
	}
	c.srvfs = append(c.srvfs, feed)
}

func (c *Client) delServerFeed(feed cipher.PubKey) {
	c.smx.Lock()
	defer c.smx.Unlock()

	for i, f := range c.srvfs {
		if f == feed {
			c.srvfs = append(c.srvfs[:i], c.srvfs[i+1:]...)
			return
		}
	}
}

func (c *Client) addRoot(feed cipher.PubKey, p []byte,
	sig cipher.Sig) (r *skyobject.Root, err error) {

	c.fmx.Lock()
	defer c.fmx.Unlock()

	for _, f := range c.feeds {
		if f == feed {
			r, err = c.so.AddEncodedRoot(p, sig)
			return
		}
	}
	return // does't have the feed
}

func (c *Client) handleMessage(cn *gnet.Conn, msg Msg) {
	c.Debugf("handle message %T", msg)

	switch x := msg.(type) {
	case *AddFeedMsg:
		c.addServerFeed(x.Feed)
	case *DelFeedMsg:
		c.delServerFeed(x.Feed)
	case *RootMsg:
		r, err := c.addRoot(x.Feed, x.Root, x.Sig)
		if err != nil {
			c.Print("[ERR] error decoding root: ", err)
			return
		} else if r == nil {
			return // too old, or already have
		}
		// request Registry if need
		if rr := r.RegistryReference(); c.so.WantRegistry(rr) {
			c.sendMessage(&RequestRegistryMsg{Ref: rr})
		}
	case *RequestRegistryMsg:
		if reg, _ := c.so.Registry(x.Ref); reg != nil {
			c.sendMessage(&RegistryMsg{
				Ref: x.Ref,
				Reg: reg.Encode(),
			})
		}
	case *RegistryMsg:
		if c.so.WantRegistry(x.Ref) {
			if reg, err := skyobject.DecodeRegistry(x.Reg); err != nil {
				c.Print("[ERR] error decoding registry: ", err)
				return
			} else if reg.Reference() != x.Ref {
				// reference not match registry
				c.Print("[ERR] received registry key-body missmatch")
				return
			} else {
				c.so.AddRegistry(reg)
			}
		}
	case *DataMsg:
		c.fmx.Lock()
		defer c.fmx.Unlock()

		for _, f := range c.feeds {
			if f == x.Feed {
				hash := skyobject.Reference(cipher.SumSHA256(x.Data))
				err := c.so.WantFeed(x.Feed,
					func(ref skyobject.Reference) (_ error) {
						if ref == hash {
							c.so.Set(hash, x.Data)
							return skyobject.ErrStopRange
						}
						return
					})
				if err != nil {
					c.Print("[ERR] error ranging feed: ", err)
				}
				return
			}
		}

	}
}

func (c *Client) sendMessage(msg Msg) (ok bool) {
	c.Debugf("send message %T", msg)

	select {
	case c.cn.SendQueue() <- Encode(msg):
		ok = true
	case <-c.cn.Closed():
	default:
		c.Print("[ERR] write queue full")
		c.cn.Close() // fatality
	}
	return
}

func (c *Client) hasServerFeed(feed cipher.PubKey) (has bool) {
	for _, sf := range c.srvfs {
		if sf == feed {
			has = true
			break
		}
	}
	return
}

func (c *Client) hasFeed(feed cipher.PubKey) (has bool) {
	for _, f := range c.feeds {
		if f == feed {
			has = true
			break
		}
	}
	return
}

func (c *Client) Subscribe(feed cipher.PubKey) (ok bool) {
	c.smx.Lock()
	defer c.smx.Unlock()

	if !c.hasServerFeed(feed) {
		return // false
		// can't subscribe if connected server doesn't has the
		// feed because it doesn't make sence
	}

	c.fmx.Lock()
	defer c.fmx.Unlock()

	if c.hasFeed(feed) {
		return // false (already subscribed)
	}
	c.feeds = append(c.feeds, feed)
	ok = c.sendMessage(&AddFeedMsg{feed})
	return
}

func (c *Client) Unsubscribe(feed cipher.PubKey) (ok bool) {
	c.fmx.Lock()
	defer c.fmx.Unlock()

	for i, f := range c.feeds {
		if f == feed {
			c.feeds = append(c.feeds[:i], c.feeds[i+1:]...)
			ok = c.sendMessage(&DelFeedMsg{feed})
			return
		}
	}
	return // false (not subscribed)
}

// Subscribed feeds
func (c *Client) Feeds() (feeds []cipher.PubKey) {
	c.fmx.Lock()
	defer c.fmx.Unlock()
	if len(c.feeds) == 0 {
		return
	}
	feeds = make([]cipher.PubKey, 0, len(c.feeds))
	copy(feeds, c.feeds)
	return
}

var ErrClosed = errors.New("closed")

// Container returns wraper around skyobject.Container.
// The wrapper sends all changes to server
func (c *Client) Container() *Container {
	return &Container{c.so, c}
}

type Container struct {
	*skyobject.Container
	client *Client
}

func (c *Container) NewRoot(pk cipher.PubKey, sk cipher.SecKey) (r *Root) {
	sr := c.Container.NewRoot(pk, sk)
	r = &Root{sr, c} // TODO
	return
}

func (c *Container) AddEncodedRoot(b []byte, sig cipher.Sig) (r *Root,
	err error) {

	var sr *skyobject.Root
	if sr, err = c.Container.AddEncodedRoot(b, sig); err != nil {
		return
	} else if sr != nil {
		r = &Root{sr, c}
	}
	return

}

func (c *Container) LastRoot(pk cipher.PubKey) (r *Root) {
	if sr := c.Container.LastRoot(pk); sr != nil {
		r = &Root{sr, c}
	}
	return
}

func (c *Container) LastFullRoot(pk cipher.PubKey) (r *Root) {
	if sr := c.Container.LastFullRoot(pk); sr != nil {
		r = &Root{sr, c}
	}
	return
}

func (c *Container) RootBySeq(pk cipher.PubKey, seq uint64) (r *Root) {
	if sr := c.Container.RootBySeq(pk, seq); sr != nil {
		r = &Root{sr, c}
	}
	return
}

type Root struct {
	*skyobject.Root
	c *Container
}

func (r *Root) Touch() {
	// just send the root
	// TODO: lock-unlock-...
	// TODO: sending never see the (*Client).feeds
	r.Root.Touch()
	// lock-unlock (+possible changes between Touch and Encode)
	b, sig := r.Encode()
	r.c.client.sendMessage(&RootMsg{
		Feed: r.Pub(),
		Sig:  sig,
		Root: b,
	})
}

func (r *Root) Inject(i interface{}) (inj skyobject.Dynamic) {
	// send root and objects related to injected object
	// TODO: lock-unlock-...
	// TODO: sending never see the (*Client).feeds
	inj = r.Root.Inject(i)
	// lock-unlock (+possible changes between Touch and Encode)
	b, sig := r.Encode()
	r.c.client.sendMessage(&RootMsg{
		Feed: r.Pub(),
		Sig:  sig,
		Root: b,
	})
	sent := make(map[skyobject.Reference]struct{})
	r.GotOfFunc(inj, func(ref skyobject.Reference) (_ error) {
		if _, ok := sent[ref]; !ok {
			data, ok := r.Get(ref)
			if !ok {
				panic("misisng object: " + ref.String())
			}
			ok = r.c.client.sendMessage(&DataMsg{
				Feed: r.Pub(),
				Data: data,
			})
			if !ok { // error sending
				return skyobject.ErrStopRange
			}
			sent[ref] = struct{}{}
		}
		return
	})
	return
}

func (r *Root) InjectMany(i ...interface{}) (injs []skyobject.Dynamic) {
	// send the root and objects related to all injected objects
	// TODO: lock-unlock-...
	// TODO: sending never see the (*Client).feeds
	injs = r.Root.InjectMany(i...)
	// lock-unlock (+possible changes between Touch and Encode)
	b, sig := r.Encode()
	r.c.client.sendMessage(&RootMsg{
		Feed: r.Pub(),
		Sig:  sig,
		Root: b,
	})
	sent := make(map[skyobject.Reference]struct{}) // already sent
	for _, inj := range injs {
		r.GotOfFunc(inj, func(ref skyobject.Reference) (_ error) {
			if _, ok := sent[ref]; !ok {
				data, ok := r.Get(ref)
				if !ok {
					panic("misisng object: " + ref.String())
				}
				ok = r.c.client.sendMessage(&DataMsg{
					Feed: r.Pub(),
					Data: data,
				})
				if !ok { // error sending
					return skyobject.ErrStopRange
				}
				sent[ref] = struct{}{}
			}
			return
		})
	}
	return
}

func (r *Root) Replace(refs []skyobject.Dynamic) (prev []skyobject.Dynamic) {
	// send root and all its got
	// TODO: lock-unlock-...
	// TODO: sending never see the (*Client).feeds
	prev = r.Root.Replace(refs)
	// lock-unlock (+possible changes between Touch and Encode)
	b, sig := r.Encode()
	r.c.client.sendMessage(&RootMsg{
		Feed: r.Pub(),
		Sig:  sig,
		Root: b,
	})
	sent := make(map[skyobject.Reference]struct{}) // already sent
	r.GotFunc(func(ref skyobject.Reference) (_ error) {
		if _, ok := sent[ref]; !ok {
			data, ok := r.Get(ref)
			if !ok {
				panic("misisng object: " + ref.String())
			}
			ok = r.c.client.sendMessage(&DataMsg{
				Feed: r.Pub(),
				Data: data,
			})
			if !ok { // error sending
				return skyobject.ErrStopRange
			}
			sent[ref] = struct{}{}
		}
		return
	})
	return
}
