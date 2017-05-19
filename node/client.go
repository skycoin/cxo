package node

import (
	"errors"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
)

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

	icmx        sync.Mutex
	isConnected bool

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

// IsConnected reports true if the Client
// connected to server
func (c *Client) IsConnected() bool {
	c.icmx.Lock()
	defer c.icmx.Unlock()
	return c.isConnected
}

func (c *Client) setIsConnected(t bool) {
	c.icmx.Lock()
	defer c.icmx.Unlock()
	c.isConnected = t
}

func (c *Client) connectHandler(cn *gnet.Conn) {
	c.Debug("connected to ", cn.Address())
	c.setIsConnected(true)
	if c.conf.OnConnect != nil {
		c.conf.OnConnect()
	}
}

func (c *Client) disconnectHandler(cn *gnet.Conn) {
	c.Debug("disconnected from ", cn.Address())
	c.setIsConnected(false)
	if c.conf.OnDisconenct != nil {
		c.conf.OnDisconenct()
	}
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
	s.sendMessage(&RootMsg{msg.Feed, full.Encode()})
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
	root, err := s.so.AddRootPack(msg.RootPack)
	if err != nil {
		if err == skyobject.ErrAlreadyHaveThisRoot {
			s.Debug("reject root: alredy have this root")
			return
		}
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

func (c *Client) handlePingMsg() {
	c.sendMessage(&PongMsg{})
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
	case *PingMsg:
		s.handlePingMsg()
	case *PongMsg:
		s.Print("[WRN] unexpected Pong message received")
	default:
		s.Printf("[CRIT] unhandled message type %T", msg)
	}

	s.Debugf("the message %T was handled", msg)
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

	if _, has := c.feeds[feed]; has {
		return // false (already subscribed)
	}
	c.feeds[feed] = struct{}{}
	if c.cn == nil {
		return
	}
	if ok = c.sendMessage(&AddFeedMsg{feed}); ok {
		if full := c.so.LastFullRoot(feed); full != nil {
			ok = c.sendMessage(&RootMsg{full.Pub(), full.Encode()})
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
	if c.cn != nil {
		ok = c.sendMessage(&DelFeedMsg{feed})
	}
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

func (c *Container) NewRoot(pk cipher.PubKey, sk cipher.SecKey) (r *Root,
	err error) {

	var sr *skyobject.Root
	if sr, err = c.Container.NewRoot(pk, sk); err != nil {
		return
	}
	r = &Root{sr, c}
	return
}

func (c *Container) AddRootPack(rp data.RootPack) (r *Root,
	err error) {

	var sr *skyobject.Root
	if sr, err = c.Container.AddRootPack(rp); err != nil {
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

type Root struct {
	*skyobject.Root
	c *Container
}

func (r *Root) send(rp data.RootPack) {
	r.c.client.Debug("sending root")
	// if !r.c.client.hasFeed(r.Pub()) {
	// 	return // don't send
	// }
	r.c.client.sendMessage(&RootMsg{
		Feed:     r.Pub(),
		RootPack: rp,
	})
}

func (r *Root) Touch() (rp data.RootPack, err error) {
	// TODO: sending never see the (*Client).feeds
	if rp, err = r.Root.Touch(); err == nil {
		r.send(rp)
	}
	return
}

func (r *Root) Inject(schemName string, i interface{}) (inj skyobject.Dynamic,
	rp data.RootPack, err error) {

	r.c.client.Debug("[*node.Root] start Inject")

	// TODO: sending never see the (*Client).feeds
	if inj, rp, err = r.Root.Inject(schemName, i); err == nil {
		r.c.client.Debug("[*node.Root] Inject: sending")
		r.send(rp)
	} else {
		r.c.client.Debug("[*node.Root] Inject-ing error: ", err)
	}
	r.c.client.Debug("[*node.Root] Inject-ing done")
	return
}

func (r *Root) InjectMany(schemaName string,
	i ...interface{}) (injs []skyobject.Dynamic, rp data.RootPack,
	err error) {

	// TODO: sending never see the (*Client).feeds
	if injs, rp, err = r.Root.InjectMany(schemaName, i...); err == nil {
		r.send(rp)
	}
	return
}

func (r *Root) Replace(refs []skyobject.Dynamic) (prev []skyobject.Dynamic,
	rp data.RootPack, err error) {

	// TODO: sending never see the (*Client).feeds
	if prev, rp, err = r.Root.Replace(refs); err == nil {
		r.send(rp)
	}
	return
}

func (r *Root) Walker() (w *RootWalker) {
	return NewRootWalker(r)
}
