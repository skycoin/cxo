package replicator

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/nodeManager"
	"github.com/skycoin/cxo/schema"
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/bbs"
	"math/rand"
)

//TODO: Refactor - avoid global var. The problem now in HandleFromUpstream/HandleFromDownstream. No way to provide Dataprovider into the handler
var DB *data.DataBase

type replicator struct {
	//db          *DataBase
	imTheVertex  bool
	subscribeTo  string
	config       *Config
	manager      *nodeManager.Manager
	dataProvider *schema.Container
}

func Client() *replicator {

	client := &replicator{}
	//1. Create Hash Database
	DB = data.NewDB()
	//2. Create schema provider. Integrate Hash Database to schema provider(schema store)

	//boards := bbs.CreateBbs(DB)
	boards:= prepareTestData(DB)
	client.dataProvider = boards.Container



	//3. Pass SchemaProvider into LaunchWebInterfaceAPI and route handdler
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
			err = DB.Add(hashOfTheData, theData)

			if err != nil {
				fmt.Println("error while adding data to db", err)
			}
		}()
	}
	return client
}

func (r *replicator) Run() {

	if r.imTheVertex {
		RunAPI(r.config, r.manager, r.dataProvider)
	} else {
		time.Sleep(time.Minute * 120)
	}
}


func prepareTestData(d *data.DataBase) *bbs.Bbs {
	bSystem := bbs.CreateBbs(d)
	boards := []bbs.Board{}
	for b := 0; b < 3; b++ {
		threads := []bbs.Thread{}
		for t := 0; t < 10; t++ {
			posts := []bbs.Post{}
			for p := 0; p < 10; p++ {
				posts = append(posts, bbs.Post{Text: "Post_" + generateString(15)})
			}
			threads = append(threads, bSystem.CreateThread("Thread_" + generateString(15), posts...))
		}
		boards = append(boards, bSystem.CreateBoard("Board_" + generateString(15), threads...))
	}
	bSystem.AddBoards(boards)
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
