package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/nodeManager"
)

// command line option flags
var (
	subscribeTo string
)

// DB holds all the data I have
var DB *DataBase

func main() {
	/*
		router := gui.NewRouter()

		h1 := func(ctx *gui.Context) error {
			fmt.Println("h1")
			return nil
		}
		h2 := func(ctx *gui.Context) error {
			fmt.Println("h2")
			return nil
		}
		h3 := func(ctx *gui.Context) error {
			fmt.Println("h3")
			return nil
		}

		router.GET("/manager/nodes/:node_id", h1)
		router.GET("/manager/nodes/:node_id/subscriptions/:subscription_id", h1)
		router.GET("/manager/nodes/:node_id/subscriptions/:subscripion_id/altro", h1)
		router.GET("/manager/nodes/:node_id/subscriptfions/:suiption_id", h2)
		router.GET("/manager", h3)
		router.GET("/manager/manager", h3)
		router.GET("/", h3)
		router.POST("/", h2)

		handler, err := router.TestHandle("POST", "/")
		if err != nil {
			fmt.Println(err)
			return
		}
		if handler != nil {
			handler(&gui.Context{})
		}
		return*/

	flag.StringVar(&subscribeTo, "subscribe-to", "", "Address of the node to subscribe to")
	cfg := defaultConfig()
	cfg.Parse()

	nodeManager.Debugging = false

	// delcare the map that will hold all the data
	DB = NewDB()

	managerConfig := nodeManager.NewManagerConfig()
	manager, err := nodeManager.NewManager(managerConfig)
	if err != nil {
		fmt.Println("error while configuring node manager", "error", err)
	}

	newNode := manager.NewNode()
	err = manager.AddNode(newNode)
	if err != nil {
		return
	}

	// this callback will be executed each time DB.Add(data) is called
	DB.NewDataCallback(func(key cipher.SHA256, value interface{}) error {
		newDataIHave := AnnounceMessage{
			HashOfNewAvailableData: key,
		}
		fmt.Println("broadcasting new data to all connected nodes")

		return newNode.BroadcastToSubscribers(newDataIHave)
	})

	// register messages that this node can receive from downstream
	newNode.RegisterDownstreamMessage(RequestMessage{})

	// register messages that this node can receive from upstream
	newNode.RegisterUpstreamMessage(AnnounceMessage{})
	newNode.RegisterUpstreamMessage(DataMessage{})

	err = newNode.Start()
	if err != nil {
		return
	}

	ImTheVertex := subscribeTo == ""

	// for the demo, there is one vertex that after a specified period of time
	// announces to its subscribers that it has new data.

	// if this node is not a vertex, just subscribe to the specified node
	// and wait for incoming AnnounceMessage
	if !ImTheVertex {

		ip, portString, err := net.SplitHostPort(subscribeTo)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}

		port, err := strconv.ParseUint(portString, 10, 16)

		// If the pubKey parameter is an empty cipher.PubKey{}, we will connect to that node
		// for any PubKey it communicates us it has.
		// For a specific match, you have to provide a specific pubKey.
		pubKeyOfNodeToSubscribeTo := &cipher.PubKey{}
		err = newNode.Subscribe(ip, uint16(port), pubKeyOfNodeToSubscribeTo)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}

	} else {
		// If I'm the vertex, I pubblish data

		// for this simple example, we use a colored piece of text
		// as data

		//lime := ansi.ColorCode("red+b:white")
		//reset := ansi.ColorCode("reset")
		colored :=
			"@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n" +
			"@@@@@ THIS IS A PIECE OF DATA @@@\n" +
			"@@@@@ SENT TO SUBSCRIBERS @@@@@@@\n" +
			"@@@@@ FROM A CENTRAL NODE @@@@@@@\n" +
			"@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n"

		theData := []byte(colored)
		// hash the question
		hashOfTheData := cipher.SumSHA256(theData)
		if err != nil {
			fmt.Println("err while signing data: ", err)
			return
		}

		go func() {
			// give time for nodes to subscribe to this node before broadcasting
			// that it has new data
			time.Sleep(time.Second * 40)

			// when adding the data, a callback is called, which announces the new data to
			// all subscribers
			err = DB.Add(hashOfTheData, theData)

			if err != nil {
				fmt.Println("error while adding data to db", err)
			}
		}()
	}

	if ImTheVertex {
		RunAPI(cfg, manager)
	} else {
		time.Sleep(time.Minute * 120)
	}
}
