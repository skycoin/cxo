package client

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/bbs"
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/gui"
	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/skyobject"
)

type Client struct {
	db *data.DB

	config    *Config
	node      node.Node
	skyobject skyobject.ISkyObjects

	// list of known feeds to subscribe to
	known []node.Connection
}

// type client struct {
// 	//db          *DataBase
// 	imTheVertex  bool
// 	subscribeTo  string
// 	config       *Config
// 	node         node.Node
// 	messenger    *gui.Messenger
// 	dataProvider skyobject.ISkyObjects
// }

func NewClient() (c *Client) {
	var err error

	c = new(Client)

	// configs
	c.config = defaultConfig()
	c.config.Parse()
	// DEVELOPMENT:
	// =========================================================================
	// generate secret key for node if it is not set
	{
		if c.config.SecretKey == "" {
			_, sec := cipher.GenerateKeyPair()
			c.config.SecretKey = sec.Hex()
		}
	}
	// =========================================================================

	// data base
	c.db = data.NewDB()

	// node
	c.node, err = node.NewNode(
		mustParseSecretKey(c.config.SecretKey), // secret key
		c.config.NodeConfig,                    // node configurations
		c)                                      // pass this to message handlers
	if err != nil {
		logger.Fatal("can't create node: ", err)
	}

	c.node.Register(&RequestMessage{})
	c.node.Register(&AnnounceMessage{})
	c.node.Register(&DataMessage{})

	if err = c.node.Start(); err != nil {
		logger.Fatal("can't start node: ", err)
	}

	// boards (TOTH: what the hell is that?)
	c.skyobject = bbs.CreateBbs(c.db, c.node)

	logger.Info("stat: %s", c.skyobject.Statistic())

	//
	// TODO: we need a way to fill up known addresses
	//
	c.known = known // temprary solution is useage of hardcoded global variable

	// subscribe to knonw
	c.subscribeToKnown()

	// geberate test data
	if c.config.Testing {
		go c.generateTestData()
	}
}

func (c *Client) subscribeToKnown() {
	var err error
	for _, c := range c.known {
		err = c.node.Outgoing().Connect(c.Addr, c.Pub)
		if err != nil {
			logger.Error("can't connect to known node [%s]: %v", c.Addr, err)
		}
	}
}

// send announce to subscribers
func (c *Client) Announce(hash cipher.SHA256) (err error) {
	err = c.node.Incoming().
		Broadcast(Announce{
			Hash: hash,
		})
	return
}

func mustParseSecretKey(str string) cipher.SecKey {
	sec, err := cipher.SecKeyFromHex(str)
	if err != nil {
		logger.Fatal(err)
	}
	return sec
}

func (c *client) Run() {
	if c.imTheVertex {
		api := SkyObjectsAPI(c.dataProvider)
		RunAPI(c.config, c.node, api)
	} else {
		time.Sleep(time.Minute * 120)
	}
}

//
// TODO: refector
//

func (c *Client) generateTestData() {
	time.Sleep(20 * time.Second)
	prepareTestData(boards)
	refs := skyobject.Href{Ref: boards.Board}
	refs.References(boards.Container)
	time.Sleep(time.Second * 30)
	c.Announce(boards.Board)
	r := skyobject.Href{Ref: boards.Board}
	rs := r.References(boards.Container)
	logger.Info("Total refs: %d %v",
		len(rs),
		boards.Container.Statistic())
	time.Sleep(time.Minute * 120)
}

func prepareTestData(bs *bbs.Bbs) {
	boards := []bbs.Board{}
	for b := 0; b < 1; b++ {
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
		boards = append(boards, bs.AddBoard(
			fmt.Sprintf("Board (board: %d)", b),
			threads...,
		))
	}
}
