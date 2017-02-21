package node

import (
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/enc"
)

//
// utils
//

var (
	encoder enc.Encoder = enc.NewEncoder()
)

func init() {
	encoder.Register(Board{})
	encoder.Register(Thread{})
	encoder.Register(Post{})
}

//
// Example messages
//

type Board struct {
	Name    string
	Threads []cipher.SHA256
}

func (b *Board) References() []cipher.SHA256 {
	return b.Threads
}

type Thread struct {
	Name  string
	Posts []cipher.SHA256
}

func (t *Thread) References() []cipher.SHA256 {
	return t.Posts
}

type Post struct {
	Header string
	Body   string
}

//
// helper functions
//

func createBoard(name string, threads ...Thread) Board {
	var (
		references []cipher.SHA256

		data []byte
		err  error
	)
	// encode threads and calculate hash of each encoded thread
	for _, th := range threads {
		if data, err = encoder.Encode(th); err != nil {
			panic(err)
		}
		references = append(references, cipher.SumSHA256(data))
	}
	//
	return Board{
		Name:    name,
		Threads: references,
	}
}

func createThread(name string, posts ...Post) Thread {
	var (
		references []cipher.SHA256

		data []byte
		err  error
	)
	// encode posts and calculate hash of each encoded post
	for _, ps := range posts {
		if data, err = encoder.Encode(ps); err != nil {
			panic(err)
		}
		references = append(references, cipher.SumSHA256(data))
	}
	//
	return Thread{
		Name:  name,
		Posts: references,
	}
}

func createPost(head, body string) Post {
	return Post{
		Header: head,
		Body:   body,
	}
}

func addToDB(n Node, obj interface{}) cipher.SHA256 {
	var (
		data []byte
		err  error

		hash cipher.SHA256
	)
	if data, err = encoder.Encode(obj); err != nil {
		panic(err)
	}
	hash = cipher.SumSHA256(data)
	n.DB().Set(hash, data)
	return hash
}

// n1 <- -> n2
func Test_2_node_duplex(t *testing.T) {
	var (
		conf1 *Config = NewConfig()
		conf2 *Config = NewConfig()

		sec cipher.SecKey

		n1 Node
		n2 Node
	)
	// generate secret key
	_, sec = cipher.GenerateKeyPair()
	// Unfortunately we can't use OS specified addr-port, because of
	// gnet linitations
	//
	conf1.Address = "127.0.0.1"
	conf1.Port = 7788
	//
	conf2.Address = "127.0.0.1"
	conf2.Port = 7789
	//
	//
	t.Log("Create nodes")
	n1 = NewNode(sec, conf1, nil, nil)
	n2 = NewNode(sec, conf2, nil, nil)
	//
	// set appropriate log prefixes
	n1.SetPrefix("[node 1] ")
	n2.SetPrefix("[node 2] ")
	//
	// we need to register messages
	t.Log("Register types")
	n1.Encoder().Register(Board{})
	n1.Encoder().Register(Thread{})
	n1.Encoder().Register(Post{})
	//
	n2.Encoder().Register(Board{})
	n2.Encoder().Register(Thread{})
	n2.Encoder().Register(Post{})
	//
	// (Node).Start doesn't block
	t.Log("start nodes")
	if err := n1.Start(); err != nil {
		t.Error("error starting node: ", err)
		return
	}
	defer n1.Close()
	if err := n2.Start(); err != nil {
		t.Error("error starting node: ", err)
		return
	}
	defer n2.Close()
	//
	// see DB stat
	t.Log("see DB statistic")
	t.Log("[n1] ", n1.DB().Stat())
	t.Log("[n2] ", n2.DB().Stat())
	//
	// cross-subscribe
	t.Log("cross-subscribe")
	if err := n1.Subscribe(n2.Address().String(), cipher.PubKey{}); err != nil {
		t.Error("subscribing error: ", err)
		return
	}
	if err := n2.Subscribe(n1.Address().String(), cipher.PubKey{}); err != nil {
		t.Error("subscribing error: ", err)
		return
	}
	//
	// ok, now, we creating Board->Threads->Posts, store objects inside
	// database of n1 and send the Board from n1 to n2
	//
	t.Log("create example objects")
	p11, p12, p21, p22 := createPost(
		"Post 1.1",
		"Some text 1.1"),
		createPost(
			"Post 1.2",
			"Some text 1.2"),
		createPost(
			"Post 2.1",
			"Some text 2.1"),
		createPost(
			"Post 2.2",
			"Some text 2.2")
	t1, t2 := createThread(
		"Thread 1",
		p11,
		p12),
		createThread(
			"Thread 2",
			p21,
			p22)
	board := createBoard("Board", t1, t2)
	//
	// stroe
	t.Log("stroe example objects inside n1")
	addToDB(n1, p11)
	addToDB(n1, p12)
	addToDB(n1, p21)
	addToDB(n1, p22)
	addToDB(n1, t1)
	addToDB(n1, t2)
	//
	// add to DB of n1 and broadcast hash of the Board to all subscribers
	t.Log("broadcast example board n1->n2")
	n1.Broadcast(addToDB(n1, board))
	//
	// wait communication
	t.Log("wait comminucation")
	time.Sleep(200 * time.Millisecond)
	//
	// ok take alook at n1
	t.Log("inspect n1")
	t.Log("[n1]", n1.DB().Stat())
	//
	// ok take alook at n2
	t.Log("inspect n2")
	t.Log("[n2]", n2.DB().Stat())

	return
}
