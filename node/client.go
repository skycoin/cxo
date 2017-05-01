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

// An Event represents client event
type Event func(*gnet.Conn) (terminate error)

// A Client represnets CXO client
type Client struct {
	log.Logger

	so *skyobject.Container
	db *data.DB

	feeds []cipher.PubKey
	srvfs []cipher.PubKey // server feeds

	eventq chan Event // events

	conf ClientConfig
	pool *gnet.Pool

	quito sync.Once
	quit  chan struct{}
	await sync.WaitGroup
}

// NewClient
func NewClient(cc ClientConfig) (c *Client, err error) {

	var db *data.DB = data.NewDB()
	return NewClientSoDB(cc, skyobject.NewContainer(db), db)
}

// NewClient with given database and container
func NewClientSoDB(cc ClientConfig, so *skyobject.Container,
	db *data.DB) (c *Client, err error) {

	if db == nil {
		panic("nil db")
	}
	if so == nil {
		panic("nil so")
	}

	c = new(Client)

	c.db = db
	c.so = so

	c.eventq = make(chan Event, 0)

	c.feeds = nil

	c.Logger = log.NewLogger(cc.Log.Prefix, cc.Log.Debug)
	cc.Config.Logger = c.Logger // use the same logger

	cc.Config.ConnectionHandler = c.connectHandler
	cc.Config.DisconnectHandler = c.disconnectHandler

	c.conf = cc

	if c.pool, err = gnet.NewPool(cc.Config); err != nil {
		c = nil
		return
	}

	return
}

func (c *Client) Start(address string) (err error) {
	var cn *gnet.Conn
	if cn, err = c.pool.Dial(address); err != nil {
		return
	}
	c.await.Add(1)
	go c.handle(cn)
	return
}

// Close client
func (c *Client) Close() (err error) {
	c.quito.Do(func() {
		close(c.quit)
	})
	err = c.pool.Close()
	// TODO: drain events
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

		events <-chan Event = c.eventq

		data []byte
		msg  Msg

		evt Event

		err error
	)

	// events loop
	for {
		select {
		case evt = <-events:
			if err = evt(cn); err != nil {
				c.Print("[ERR] ", err)
				cn.Close()
				return
			}
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

func (c *Client) handleMessage(cn *gnet.Conn, msg Msg) {
	switch x := msg.(type) {
	case *AddFeedMsg:
		for _, f := range c.srvfs {
			if f == x.Feed {
				return // already have the feed
			}
		}
		c.srvfs = append(c.srvfs, x.Feed)
	case *DelFeedMsg:
		for i, f := range c.srvfs {
			if f == x.Feed {
				c.srvfs = append(c.srvfs[:i], c.srvfs[i+1:]...)
				return
			}
		}
	case *RootMsg:
		for _, f := range c.feeds {
			if f == x.Feed {
				ok, err := c.so.AddEncodedRoot(x.Root, x.Feed, x.Sig)
				if err != nil {
					c.Print("[ERR] error decoding root: ", err)
					return
				}
				if !ok {
					return // older
				}
				return
			}
		}
	case *DataMsg:
		for _, f := range c.feeds {
			if x.Feed == f {
				hash := cipher.SumSHA256(x.Data)
				root := c.so.Root(x.Feed)
				if root == nil {
					return // doesn't have a root object of the feed
				}
				want, err := root.Want()
				if err != nil {
					c.Print("[ERR] malformed root: ", err)
					// TODO: reset roo object
					return
				}
				if _, ok := want[skyobject.Reference(hash)]; ok {
					c.db.Set(hash, x.Data)
					return
				}
				return // don't want the data
			}
		}
	}
}

func (c *Client) sendMessage(cn *gnet.Conn, msg Msg) (ok bool) {
	select {
	case cn.SendQueue() <- Encode(msg):
		ok = true
	case <-cn.Closed():
	default:
		c.Print("[ERR] write queue full")
		cn.Close() // fatality
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
	okChan := make(chan bool, 1)
	select {
	case c.eventq <- func(cn *gnet.Conn) (terminate error) {
		if !c.hasServerFeed(feed) {
			okChan <- false
			return
			// can't subscribe if connected server doesn't has the
			// feed because it doesn't make sence
		}
		if c.hasFeed(feed) {
			okChan <- false // already subscribed
			return
		}
		c.feeds = append(c.feeds, feed)
		okChan <- c.sendMessage(cn, &AddFeedMsg{feed})
		return
	}:
	case <-c.quit:
		return // false
	}
	ok = <-okChan
	return
}

func (c *Client) Unsubscribe(feed cipher.PubKey) (ok bool) {
	okChan := make(chan bool, 1)
	select {
	case c.eventq <- func(cn *gnet.Conn) (terminate error) {
		for i, f := range c.feeds {
			if f == feed {
				c.feeds = append(c.feeds[:i], c.feeds[i+1:]...)
				okChan <- c.sendMessage(cn, &DelFeedMsg{feed})
				return
			}
		}
		okChan <- false // don't subscribed
		return
	}:
	case <-c.quit:
		return // false
	}
	ok = <-okChan
	return
}

// Subscribed feeds
func (c *Client) Feeds() []cipher.PubKey {
	reply := make(chan []cipher.PubKey, 1)
	select {
	case c.eventq <- func(cn *gnet.Conn) (terminate error) {
		f := make([]cipher.PubKey, 0, len(c.feeds))
		copy(f, c.feeds)
		reply <- f
		return
	}:
	case <-c.quit:
		return nil
	}
	return <-reply
}

var ErrClosed = errors.New("closed")

func (c *Client) Execute(fn func(*Container) error) error {
	reply := make(chan error, 1)
	select {
	case c.eventq <- func(cn *gnet.Conn) (_ error) {
		reply <- fn(&Container{c.so, c, cn})
		return
	}:
	case <-c.quit:
		return ErrClosed
	}
	return <-reply
}

type Container struct {
	*skyobject.Container
	client *Client
	cn     *gnet.Conn
}

func (c *Container) NewRoot(pk cipher.PubKey, sk cipher.SecKey) (root *Root) {
	root = &Root{c.Container.NewRoot(pk, sk), c.client, c.cn}
	// send the root
	c.client.sendMessage(c.cn, &RootMsg{
		root.Pub,
		root.Sig,
		root.Encode(),
	})
	// a fresh root hasn't got any references
	return
}

func (c *Container) Root(pk cipher.PubKey) (r *Root) {
	if root := c.Container.Root(pk); root != nil {
		r = &Root{root, c.client, c.cn}
	}
	return
}

type Root struct {
	*skyobject.Root
	client *Client
	cn     *gnet.Conn
}

func (r *Root) Inject(i interface{},
	sk cipher.SecKey) (inj skyobject.Reference) {

	inj = r.Root.Inject(i, sk)
	r.client.sendMessage(r.cn, &RootMsg{
		r.Pub,
		r.Sig,
		r.Encode(),
	})
	got, _ := r.GotOf(inj)
	for k := range got {
		data, _ := r.client.db.Get(cipher.SHA256(k))
		r.client.sendMessage(r.cn, &DataMsg{r.Pub, data})
	}
	return
}
