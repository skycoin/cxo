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
	feeds map[cipher.PubKey]struct{} // subscriptions

	smx   sync.Mutex
	srvfs map[cipher.PubKey]struct{} // server feeds

	rmx   sync.RWMutex
	roots []*fillRoot // filling up received root objects

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
	c.feeds = make(map[cipher.PubKey]struct{})
	c.srvfs = make(map[cipher.PubKey]struct{})
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

//
// ============================================================================

func (s *Client) addServerFeed(feed cipher.PubKey) (added bool) {
	s.smx.Lock()
	defer s.smx.Unlock()
	if _, ok := s.srvfs[feed]; ok {
		return // already
	}
	s.srvfs[feed], added = struct{}{}, true
	return
}

func (s *Client) handleAddFeedMsg(msg *AddFeedMsg) {
	if !s.addServerFeed(msg.Feed) {
		return
	}
	full := s.so.LastFullRoot(msg.Feed)
	if full == nil {
		return
	}
	p, sig := full.Encode()
	s.sendMessage(&RootMsg{msg.Feed, sig, p})
}

func (s *Client) handleDelFeedMsg(msg *DelFeedMsg) {
	s.smx.Lock()
	defer s.smx.Unlock()
	delete(s.srvfs, msg.Feed)
	return
}

func (s *Client) hasFeed(pk cipher.PubKey) (yep bool) {
	s.fmx.Lock()
	defer s.fmx.Unlock()
	_, yep = s.feeds[pk]
	return
}

func (s *Client) addNonFullRoot(root *skyobject.Root,
	c *gnet.Conn) (fl *fillRoot) {

	fl = &fillRoot{root, c, skyobject.Reference{}}
	s.roots = append(s.roots, fl)
	return
}

func (s *Client) delNonFullRoot(root *skyobject.Root) {
	for i, fl := range s.roots {
		if fl.root == root {
			copy(s.roots[i:], s.roots[i+1:])
			s.roots[len(s.roots)-1] = nil // set to nil for golang GC
			s.roots = s.roots[:len(s.roots)-1]
			return
		}
	}
	return
}

func (s *Client) handleRootMsg(msg *RootMsg) {
	if !s.hasFeed(msg.Feed) {
		return
	}
	root, err := s.so.AddEncodedRoot(msg.Root, msg.Sig)
	if err != nil {
		s.Print("[ERR] error decoding root: ", err)
		return
	}
	if root.IsFull() {
		return
	}

	s.rmx.Lock()
	defer s.rmx.Unlock()

	fl := s.addNonFullRoot(root, nil)
	if !root.HasRegistry() {
		if !s.sendMessage(&RequestRegistryMsg{root.RegistryReference()}) {
			s.delNonFullRoot(root) // sending error (connection closed)
		}
		return
	}
	err = root.WantFunc(func(ref skyobject.Reference) error {
		if !s.sendMessage(&RequestDataMsg{ref}) {
			s.delNonFullRoot(root) // sending error (connection closed)
		} else {
			fl.await = ref // keep last requested reference
		}
		return skyobject.ErrStopRange
	})
	if err != nil {
		s.Print("[ERR] unexpected error: ", err)
	}
}

func (s *Client) handleRequestRegistryMsg(msg *RequestRegistryMsg) {

	if reg, _ := s.so.Registry(msg.Ref); reg != nil {
		s.sendMessage(&RegistryMsg{reg.Encode()})
	}
}

func (s *Client) handleRegistryMsg(msg *RegistryMsg) {
	reg, err := skyobject.DecodeRegistry(msg.Reg)
	if err != nil {
		s.Print("[ERR] error decoding received registry:", err)
		return
	}

	if !s.so.WantRegistry(reg.Reference()) {
		return // don't want the registry
	}

	s.so.AddRegistry(reg)

	s.rmx.Lock()
	defer s.rmx.Unlock()
	var i int = 0 // index for deleting
	for _, fl := range s.roots {
		if fl.root.RegistryReference() == reg.Reference() {
			if fl.root.IsFull() {
				continue // delete
			}
			var sent bool
			err = fl.root.WantFunc(func(ref skyobject.Reference) error {
				if sent = s.sendMessage(&RequestDataMsg{ref}); sent {
					fl.await = ref
				}
				return skyobject.ErrStopRange
			})
			if err != nil {
				s.Print("[ERR] unexpected error: ", err)
				continue // delete
			}
			if !sent {
				continue // delete
			}
		}
		s.roots[i] = fl
		i++
	}
	s.roots = s.roots[:i]
}

func (s *Client) handleRequestDataMsg(msg *RequestDataMsg) {
	if data, ok := s.so.Get(msg.Ref); ok {
		s.sendMessage(&DataMsg{data})
	}
}

func (s *Client) handleDataMsg(msg *DataMsg) {
	hash := skyobject.Reference(cipher.SumSHA256(msg.Data))

	s.rmx.Lock()
	defer s.rmx.Unlock()

	// does the Server really want the data
	var want bool
	for _, fl := range s.roots {
		if fl.await == hash {
			want = true
			break
		}
	}
	if !want {
		return // doesn't want the data
	}
	s.so.Set(hash, msg.Data) // save

	// check filling
	var i int = 0 // index for deleting
	for _, fl := range s.roots {
		if fl.await == hash {
			if fl.root.IsFull() {
				continue // delete
			}
			var sent bool
			err := fl.root.WantFunc(func(ref skyobject.Reference) error {
				if sent = s.sendMessage(&RequestDataMsg{ref}); sent {
					fl.await = ref
				}
				return skyobject.ErrStopRange
			})
			if err != nil {
				s.Print("[ERR] unexpected error: ", err)
				continue // delete
			}
			if !sent {
				continue // delete
			}
		}
		s.roots[i] = fl
		i++
	}
	s.roots = s.roots[:i]
}

// ============================================================================
//

func (s *Client) handleMessage(c *gnet.Conn, msg Msg) {
	s.Debugf("handle message %T from %s", msg, c.Address())

	switch x := msg.(type) {
	case *AddFeedMsg:
		s.handleAddFeedMsg(x)
	case *DelFeedMsg:
		s.handleDelFeedMsg(x)
	case *RootMsg:
		s.handleRootMsg(x)
	case *RequestRegistryMsg:
		s.handleRequestRegistryMsg(x)
	case *RegistryMsg:
		s.handleRegistryMsg(x)
	case *RequestDataMsg:
		s.handleRequestDataMsg(x)
	case *DataMsg:
		s.handleDataMsg(x)
	default:
		s.Printf("[CRIT] unhandled message type %T", msg)
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

func (c *Client) Subscribe(feed cipher.PubKey) (ok bool) {
	c.fmx.Lock()
	defer c.fmx.Unlock()

	if c.hasFeed(feed) {
		return // false (already subscribed)
	}
	c.feeds[feed] = struct{}{}
	if ok = c.sendMessage(&AddFeedMsg{feed}); ok {
		if full := c.so.LastFullRoot(feed); full != nil {
			p, sig := full.Encode()
			ok = c.sendMessage(&RootMsg{full.Pub(), sig, p})
		}
	}
	return
}

// Unsubscribe from a feed
func (c *Client) Unsubscribe(feed cipher.PubKey) (ok bool) {
	c.fmx.Lock()
	defer c.fmx.Unlock()

	if _, sub := c.feeds[feed]; !sub {
		return // not subscribed
	}
	delete(c.feeds, feed)
	ok = c.sendMessage(&DelFeedMsg{feed})
	c.so.DelFeed(feed) // remove from skyobject
	return
}

// Subscribed feeds
func (c *Client) Feeds() (feeds []cipher.PubKey) {
	c.fmx.Lock()
	defer c.fmx.Unlock()
	if len(c.feeds) == 0 {
		return
	}
	feeds = make([]cipher.PubKey, 0, len(c.feeds))
	for f := range c.feeds {
		feeds = append(feeds, f)
	}
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

func (r *Root) send(p []byte, sig cipher.Sig) {
	if !r.c.client.hasFeed(r.Pub()) {
		return // don't send
	}
	r.c.client.sendMessage(&RootMsg{
		Feed: r.Pub(),
		Sig:  sig,
		Root: p,
	})
}

func (r *Root) Touch() (sig cipher.Sig, p []byte) {
	// TODO: sending never see the (*Client).feeds
	sig, p = r.Root.Touch()
	r.send(p, sig)
	return
}

func (r *Root) Inject(i interface{}) (inj skyobject.Dynamic, sig cipher.Sig,
	p []byte) {

	// TODO: sending never see the (*Client).feeds
	inj, sig, p = r.Root.Inject(i)
	r.send(p, sig)
	return
}

func (r *Root) InjectMany(i ...interface{}) (injs []skyobject.Dynamic,
	sig cipher.Sig, p []byte) {

	// send the root and objects related to all injected objects
	// TODO: sending never see the (*Client).feeds
	injs, sig, p = r.Root.InjectMany(i...)
	r.send(p, sig)
	return
}

func (r *Root) Replace(refs []skyobject.Dynamic) (prev []skyobject.Dynamic,
	sig cipher.Sig, p []byte) {

	// send root and all its got
	// TODO: sending never see the (*Client).feeds
	prev, sig, p = r.Root.Replace(refs)
	r.send(p, sig)
	return
}
