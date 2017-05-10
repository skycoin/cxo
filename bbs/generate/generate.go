package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/skyobject"
)

// keys for example:
// public: 03517b80b2889e4de80aae0fa2a4b2a408490f3178857df5b756e690b4524e1e61
// secret: 3cd98cc9385225f9af47e5ff0dfc073253aa410076cf5f426c19460a1d0de976

type User struct {
	Name string
	Age  uint32
}

// A Board
type Board struct {
	Header  string
	Threads skyobject.References `skyobject:"schema=bbs.Thread"` // []Thread
	Owner   skyobject.Dynamic
}

// A Thread
type Thread struct {
	Header string
	Posts  skyobject.References `skyobject:"schema=bbs.Post"` // []Post
}

// A Post
type Post struct {
	Header string
	Body   string
}

func main() {
	var (
		serverAddress        string = "[::]:8998"
		publicKey, secretKey string

		conf node.ClientConfig = node.NewClientConfig()
	)

	conf.FromFlags()

	flag.StringVar(&serverAddress,
		"a",
		serverAddress,
		"address of server to connect to")
	flag.StringVar(&publicKey,
		"pk",
		"",
		"public key (required)")
	flag.StringVar(&secretKey,
		"sk",
		"",
		"secret key (required)")

	flag.Parse()

	pk, err := cipher.PubKeyFromHex(publicKey)
	if err != nil {
		log.Fatal(err)
	}
	sk, err := cipher.SecKeyFromHex(secretKey)
	if err != nil {
		log.Fatal(err)
	}

	reg := skyobject.NewRegistry()

	reg.Regsiter("bbs.Board", Board{})
	reg.Regsiter("bbs.Thread", Thread{})
	reg.Regsiter("bbs.Post", Post{})
	reg.Regsiter("bbs.User", User{})

	c, err := node.NewClient(conf, skyobject.NewContainer(reg))
	if err != nil {
		log.Fatal(err)
	}

	if err = c.Start(serverAddress); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	time.Sleep(5 * time.Second) // sync with the server

	// subscribe to the feed
	if !c.Subscribe(pk) {
		log.Print("can't subscribe: server doesn't share the feed")
		return
	}

	go generate(c.Container(), pk, sk) // generate threads infinity

	waitInterrupt() // exit on SIGINT

	return
}

func waitInterrupt() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

func generate(c *node.Container, pk cipher.PubKey, sk cipher.SecKey) {
	var i int = 0
	fst, omt := time.Tick(5*time.Second), time.Tick(time.Minute)
	for {
		select {
		case <-fst:
			generateBoards(c, pk, sk, i) // add new board every 5 seconds
			i++
		case <-omt:
			root := c.NewRoot(pk, sk)
			root.Touch() // reset root every minute

			// don't reset the i variable keeping incrementing it
		}
	}
}

func shortHex(a string) string {
	return string([]byte(a)[:7])
}

func generateBoards(c *node.Container, pk cipher.PubKey, sk cipher.SecKey,
	i int) {

	root := c.NewRoot(pk, sk)
	root.Inject(Board{
		Header:  fmt.Sprintf("Board #%d", i),
		Threads: generateThreads(c, i),
		Owner:   root.Dynamic(User{"Alice Cooper", 16}),
	})
}

func generateThreads(c *node.Container, i int) (threads skyobject.References) {
	for t := 1; t < 4; t++ {
		ref := c.Save(Thread{
			Header: fmt.Sprintf("Thread #%d.%d", i, t),
			Posts:  generatePosts(c, i, t),
		})
		threads = append(threads, ref)
	}
	return
}

func generatePosts(c *node.Container, i, t int) skyobject.References {
	return c.SaveArray(
		Post{
			Header: fmt.Sprintf("Post #%d.%d.1", i, t),
			Body:   fmt.Sprintf("Body #%d.%d.1", i, t),
		},
		Post{
			Header: fmt.Sprintf("Post #%d.%d.2", i, t),
			Body:   fmt.Sprintf("Body #%d.%d.2", i, t),
		},
		Post{
			Header: fmt.Sprintf("Post #%d.%d.3", i, t),
			Body:   fmt.Sprintf("Body #%d.%d.3", i, t),
		},
	)
}

func hashTree(r *node.Root) {
	//
}
