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

	boards := prepareTestData(DB, newNode)
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

func prepareTestData(ds data.IDataSource, sec nodeManager.INodeSecurity) *bbs.Bbs {
	bSystem := bbs.CreateBbs(ds, sec)
	boards := []bbs.Board{}
	for b := 0; b < 1; b++ {
		threads := []bbs.Thread{}
		for t := 0; t < 1; t++ {
			posts := []bbs.Post{}
			//for p := 0; p < 1; p++ {
			//	posts = append(posts, bSystem.CreatePost("Post_" + generateString(15), "Some text"))
			//}
			threads = append(threads, bSystem.CreateThread("Thread_" + generateString(15), posts...))
		}
		boards = append(boards, bSystem.AddBoard("Board_" + generateString(15), threads...))
	}
	return bSystem
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

