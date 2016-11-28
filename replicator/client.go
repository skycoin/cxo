package replicator

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/nodeManager"
)


type replicator struct {
	db          *DataBase
	imTheVertex bool
	subscribeTo string
	config      *Config
	manager     *nodeManager.Manager
}

func Client() *replicator {

	client := &replicator{db:NewDB()}
	//1. Create Hash Database
	//2. Create schema provider
	//3. Integrate Hash Database to schema provider(schema store)
	//4. Pass SchemaProvider into LaunchWebInterfaceAPI and route handdler
	//schema := schema.NewStore()

	flag.StringVar(&client.subscribeTo, "subscribe-to", "", "Address of the node to subscribe to")
	client.config = defaultConfig()
	client.config.Parse()

	nodeManager.Debugging = false

	managerConfig := nodeManager.NewManagerConfig()
	manager, err := nodeManager.NewManager(managerConfig)
	client.manager = manager

	if err != nil {
		fmt.Println("error while configuring node manager", "error", err)
	}

	newNode := manager.NewNode()
	err = manager.AddNode(newNode)
	if err != nil {
		panic("Can't create node")
	}

	// this callback will be executed each time DB.Add(data) is called
	client.db.NewDataCallback(func(key cipher.SHA256, value interface{}) error {
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
		panic("Can't create node")
	}

	client.imTheVertex = client.subscribeTo == ""

	// for the demo, there is one vertex that after a specified period of time
	// announces to its subscribers that it has new data.

	// if this node is not a vertex, just subscribe to the specified node
	// and wait for incoming AnnounceMessage
	if !client.imTheVertex {

		ip, portString, err := net.SplitHostPort(client.subscribeTo)
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
		// If I'm the vertex, I pubblish data

		// for this simple example, we use a colored piece of text
		// as data

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
			panic("Can't create node")
		}

		go func() {
			// give time for nodes to subscribe to this node before broadcasting
			// that it has new data
			time.Sleep(time.Second * 40)

			// when adding the data, a callback is called, which announces the new data to
			// all subscribers
			err = client.db.Add(hashOfTheData, theData)

			if err != nil {
				fmt.Println("error while adding data to db", err)
			}
		}()
	}
	return client
}

func (r *replicator) Run() {

	if r.imTheVertex {
		RunAPI(r.config, r.manager)
	} else {
		time.Sleep(time.Minute * 120)
	}
}
