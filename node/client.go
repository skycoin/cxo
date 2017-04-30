package node

import (
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

	conf  ClientConfig
	pool  *gnet.Pool
	await sync.WaitGroup
}

// NewClient
func NewClient(cc ClientConfig) (c *Client, err error) {

	var db *data.DB = data.NewDB()
	return NewClientSoDB(cc, db, skyobject.NewContainer(db))
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

	cc.Config.ConnectionHandler = s.connectHandler
	cc.Config.DisconnectHandler = s.disconnectHandler

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
}

// Close client
func (c *Client) Close() (err error) {
	err = c.pool.Close()
	// TODO: drain events
	c.await.Wait()
	return
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
		//
	case *DelFeedMsg:
		//
	case *RootMsg:
		//
	case *DataMsg:
		//
	}
}
