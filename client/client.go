package client

import (
	"flag"
	"fmt"
	"time"
	"github.com/skycoin/cxo/nodeManager"
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/bbs"
	"github.com/skycoin/cxo/gui"
	"math/rand"
	"strconv"
	"net"
	"github.com/skycoin/skycoin/src/cipher"
)

//TODO: Refactor - avoid global var. The problem now in HandleFromUpstream/HandleFromDownstream. No way to provide Dataprovider into the handler
var DB *data.DataBase
var Sync *syncContext

type client struct {
	//db          *DataBase
	imTheVertex  bool
	subscribeTo  string
	config       *Config
	manager      *nodeManager.Manager
	messanger    *gui.Messenger
	dataProvider skyobject.ISkyObjects
}

func Client() *client {

	c := &client{}
	//1. Create Hash Database
	DB = data.NewDB()
	//2. Create schema provider. Integrate Hash Database to schema provider(schema store)

	//3. Pass SchemaProvider into LaunchWebInterfaceAPI and route handdler
	flag.StringVar(&c.subscribeTo, "subscribe-to", "", "Address of the node to subscribe to")
	c.config = defaultConfig()
	c.config.Parse()

	nodeManager.Debugging = false

	managerConfig := nodeManager.NewManagerConfig()
	manager, err := nodeManager.NewManager(managerConfig)
	c.manager = manager

	if err != nil {
		fmt.Println("error while configuring node manager", "error", err)
	}

	newNode := manager.NewNode()
	err = manager.AddNode(newNode)
	if err != nil {
		panic("Can't create node")
	}

	//// this callback will be executed each time DB.Add(data) is called
	//DB.NewDataCallback(func(key cipher.SHA256, value interface{}) error {
	//	newDataIHave := AnnounceMessage{
	//		Hash: key,
	//	}
	//	fmt.Println("broadcasting new data to all connected nodes")
	//
	//	return newNode.BroadcastToSubscribers(newDataIHave)
	//})


	// register messages that this node can receive from downstream
	newNode.RegisterDownstreamMessage(RequestMessage{})

	// register messages that this node can receive from upstream
	newNode.RegisterUpstreamMessage(AnnounceMessage{})
	newNode.RegisterUpstreamMessage(DataMessage{})

	err = newNode.Start()
	if err != nil {
		panic("Can't create node")
	}

	c.messanger = NodeMessanger(newNode)

	Sync = SyncContext(c.messanger)
	c.imTheVertex = c.subscribeTo == ""
	boards := bbs.CreateBbs(DB, newNode)

	if !c.imTheVertex {
		prepareTestData(boards)
		ip, portString, err := net.SplitHostPort(c.subscribeTo)
		if err != nil {
			fmt.Println("err: ", err)
			panic("Can't create node")
		}

		port, err := strconv.ParseUint(portString, 10, 16)

		// If the pubKey parameter is an empty cipher.PubKey{}, we will connect to that node
		// for any PubKey it communicates us it has.
		// For a specific match, you have to provide a specific pubKey.
		pubKeyOfNodeToSubscribeTo := &cipher.PubKey{}
		err = newNode.Subscribe(ip, uint16(port), pubKeyOfNodeToSubscribeTo)
		if err != nil {
			fmt.Println("err: ", err)
			panic("Can't create node")
		}

	} else {
		go func() {
			// give time for nodes to subscribe to this node before broadcasting
			// that it has new data
			time.Sleep(time.Second * 20)

			newDataIHave := AnnounceMessage{
				Hash: boards.Board,
			}
			fmt.Println("Broadcast hash", boards.Board)
			newNode.BroadcastToSubscribers(newDataIHave)
		}()
	}



	//
	//if c.imTheVertex {

	//}
	//fmt.Println("boards.Container", boards.Container)
	c.dataProvider = boards.Container
	return c
}

func (c *client) Run() {
	if c.imTheVertex {
		synchronizer := skyobject.Synchronizer(c.dataProvider, Sync)
		api := SkyObjectsAPI(c.dataProvider, synchronizer)
		RunAPI(c.config, c.manager, api)
	} else {
		time.Sleep(time.Minute * 120)
	}
}

func prepareTestData(bs *bbs.Bbs) {
	boards := []bbs.Board{}
	for b := 0; b < 1; b++ {
		threads := []bbs.Thread{}
		for t := 0; t < 10; t++ {
			posts := []bbs.Post{}
			for p := 0; p < 10; p++ {
				posts = append(posts, bs.CreatePost("Post_" + generateString(15), "Some text"))
			}
			threads = append(threads, bs.CreateThread("Thread_" + generateString(15), posts...))
		}
		boards = append(boards, bs.AddBoard("Board_" + generateString(15), threads...))
	}
}

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1 << letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func generateString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n - 1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

