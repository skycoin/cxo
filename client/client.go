package client

import (
	"fmt"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/bbs"
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/skyobject"
)

// A Client repesents client that is node with web-interface, database and bss
type Client struct {
	db *data.DB

	config    *Config
	node      node.Node
	skyobject skyobject.ISkyObjects

	// hash of root object
	root cipher.SHA256

	// list of known feeds to subscribe to
	known []node.Connection

	quit chan struct{}
}

// NewClient creates new client or returns error
func NewClient() (c *Client, err error) {
	c = new(Client)

	// configs
	c.config = defaultConfig()
	c.config.Parse()
	// DEVELOPMENT:
	// =========================================================================
	// generate secret key for node if it is not set
	{
		if c.config.SecretKey == "" {
			logger.Info("secret key is not provided: generate")
			_, sec := cipher.GenerateKeyPair()
			c.config.SecretKey = sec.Hex()
		}
	}
	// =========================================================================

	// data base
	c.db = data.NewDB()

	// node
	c.config.NodeConfig.ConnectCallback = c.connectCallback
	c.node, err = node.NewNode(
		mustParseSecretKey(c.config.SecretKey), // secret key
		c.config.NodeConfig,                    // node configurations
		c)                                      // pass this to message handlers
	if err != nil {
		err = fmt.Errorf("can't create node: %v", err)
		return
	}

	c.node.Register(&Request{})
	c.node.Register(&Announce{})
	c.node.Register(&Data{})
	c.node.Register(&Root{})

	//
	// TODO: we need a way to fill up known addresses
	//
	c.known = known // temprary solution is useage of hardcoded global variable

	return
}

// Start launches client or return error
func (c *Client) Start() (err error) {
	// start node
	if err = c.node.Start(); err != nil {
		err = fmt.Errorf("can't start node: %v", err)
		return
	}

	// boards
	var boards *bbs.Bbs = bbs.CreateBbs(c.db, c.node)
	c.skyobject = boards.Container

	logger.Info("stat: %s", c.skyobject.Statistic())

	// subscribe to knonw
	c.subscribeToKnown()

	// generate test data
	if c.config.Testing {
		go c.generateTestData(boards)
	}

	// web interface
	c.launchWebInterface()

	if c.config.RemoteTermination == true {
		c.quit = make(chan struct{})
	}

	return
}

// Close is used to shutdown client. Unfortunately, there's no way to
// shutdown web-interface. Thus, we can't reuse closed client
func (c *Client) Close() {
	if c.config.Log.Level == "debug" {
		logger.Debug("\n\nINSPECT\n\n")
		c.skyobject.Inspect()
	}
	logger.Info("closing...")
	c.node.Close()
}

// Set the given hash as root object of the node and
// send announce for all subscribers
func (c *Client) Publish(root cipher.SHA256) {
	c.setRoot(root)
	c.announce(root)
}

//
func (c *Client) setRoot(root cipher.SHA256) {
	c.root = root
}

func (c *Client) connectCallback(conn node.Sender, outgoign bool) {
	if outgoign {
		// if we've not root
		if c.root == (cipher.SHA256{}) {
			// send empty request to get root object
			logger.Debug("send request for root")
			conn.Send(Request{})
		}
	}
}

// send announce to subscribers
func (c *Client) announce(hash cipher.SHA256) (err error) {
	logger.Debugf("broadcast announce %s", hash.Hex())
	err = c.node.Incoming().
		Broadcast(Announce{
			Hash: hash,
		})
	return
}

func (c *Client) requestMissing(r Replier, hash cipher.SHA256) {
	for _, item := range c.skyobject.MissingDependencies(hash) {
		logger.Debugf("send request %s", item.Hex())
		err := r.Reply(Request{
			Hash: item,
		})
		if err != nil {
			logger.Error("error sending request: %v", err)
		}
	}
}

// hash is valid, and fresly added to the db
func (c *Client) gotNewData(r Replier, hash cipher.SHA256) {
	c.requestMissing(r, hash)
	if err := c.announce(hash); err != nil && err != node.ErrNotListening {
		logger.Errorf("error sending announce: %v", err)
	}
}

func (c *Client) subscribeToKnown() {
	var err error
	for _, k := range c.known {
		err = c.node.Outgoing().Connect(k.Addr, k.Pub)
		if err != nil {
			logger.Error("can't connect to known node [%s]: %v", k.Addr, err)
		}
	}
}

func mustParseSecretKey(str string) cipher.SecKey {
	sec, err := cipher.SecKeyFromHex(str)
	if err != nil {
		logger.Fatal(err)
	}
	return sec
}

//
// testing
//

func (c *Client) generateTestData(boards *bbs.Bbs) {
	prepareTestData(boards)
	time.Sleep(10 * time.Second)
	// boards is root
	c.Publish(boards.Board)
	//
	r := skyobject.Href{
		Ref: boards.Board,
	}
	rs := r.References(boards.Container)
	logger.Info("Total refs: %d %v",
		len(rs),
		boards.Container.Statistic())
}

func prepareTestData(bs *bbs.Bbs) {
	// TOTH: many boards?
	// for b := 0; b < 1; b++ {
	b := 0
	threads := []bbs.Thread{}
	for t := 0; t < 10; t++ {
		posts := []bbs.Post{}
		for p := 0; p < 200; p++ {
			posts = append(posts, bs.CreatePost(
				fmt.Sprintf("Post (board: %d, thread: %d, post:%d)",
					b, t, p),
				"Some text",
			))
		}
		threads = append(threads, bs.CreateThread(
			fmt.Sprintf("Thread (board: %d, thread: %d)", b, t),
			posts...,
		))
	}
	bs.AddBoard(fmt.Sprintf("Board (board: %d)", b), threads...)
	// }
}
