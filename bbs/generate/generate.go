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

// A Board
type Board struct {
	Header  string
	Threads skyobject.References // []Thread
}

// A Thread
type Thread struct {
	Header string
	Posts  skyobject.References // []Post
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

		cc node.ClientConfig = node.NewClientConfig()
	)

	cc.FromFlags()

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

	c, err := node.NewClient(cc)
	if err != nil {
		log.Fatal(err)
	}
	if err = c.Start(serverAddress); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	time.Sleep(1 * time.Second) // sync with the server

	// subscribe to the feed
	if !c.Subscribe(pk) {
		log.Print("can't subscribe: server doesn't share the feed")
		return
	}

	c.Execute(func(c *node.Container) (_ error) {
		// register types to use
		c.Register(
			Board{},
			Thread{},
			Post{},
		)
		// create empty root
		c.NewRoot(pk, sk)
		return
	})

	go generate(c, pk, sk) // generate threads infinity

	waitInterrupt() // exit on SIGINT

	return
}

func waitInterrupt() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

func generate(c *node.Client, pk cipher.PubKey, sk cipher.SecKey) {
	c.Execute(func(c *node.Container) (_ error) {
		var i int = 0
		select {
		case <-time.Tick(5 * time.Second):
			generateBoards(c, pk, sk, i) // add new board every 5 seconds
			i++
		case <-time.Tick(time.Minute):
			c.NewRoot(pk, sk) // reset root every minute
			// don't reset the i variable keeping incrementing it
		}
		return
	})
}

func generateBoards(c *node.Container, pk cipher.PubKey, sk cipher.SecKey,
	i int) {

	root := c.Root(pk)
	root.Inject(Board{
		Header:  fmt.Sprintf("Board #%d", i),
		Threads: generateThreads(c, i),
	}, sk)
}

func generateThreads(c *node.Container, i int) (threads skyobject.References) {
	for t := 1; t < 4; t++ {
		ref := c.Save(Thread{
			Header: fmt.Sprintf("Thread #%d.%t", i, t),
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
