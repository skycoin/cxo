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
		c.NewRoot(pk)
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
		for i := 0; true; i++ {
			<-time.After(time.Second)
			root := c.Root(pk)
			root.Inject(Board{
				Header: fmt.Sprintf("Board #%d", i),
				Threads: c.SaveArray(
					Thread{
						Header: fmt.Sprintf("Thread #%d.1", i),
						Posts: c.SaveArray(
							Post{
								Header: fmt.Sprintf("Post #%d.1.1", i),
								Body:   fmt.Sprintf("Body #%d.1.1", i),
							},
							Post{
								Header: fmt.Sprintf("Post #%d.1.2", i),
								Body:   fmt.Sprintf("Body #%d.1.2", i),
							},
							Post{
								Header: fmt.Sprintf("Post #%d.1.3", i),
								Body:   fmt.Sprintf("Body #%d.1.3", i),
							},
						),
					},
					Thread{
						Header: fmt.Sprintf("Thread #%d.2", i),
						Posts: c.SaveArray(
							Post{
								Header: fmt.Sprintf("Post #%d.2.1", i),
								Body:   fmt.Sprintf("Body #%d.2.1", i),
							},
							Post{
								Header: fmt.Sprintf("Post #%d.2.2", i),
								Body:   fmt.Sprintf("Body #%d.2.2", i),
							},
							Post{
								Header: fmt.Sprintf("Post #%d.2.3", i),
								Body:   fmt.Sprintf("Body #%d.2.3", i),
							},
						),
					},
					Thread{
						Header: fmt.Sprintf("Thread #%d.3", i),
						Posts: c.SaveArray(
							Post{
								Header: fmt.Sprintf("Post #%d.3.1", i),
								Body:   fmt.Sprintf("Body #%d.3.1", i),
							},
							Post{
								Header: fmt.Sprintf("Post #%d.3.2", i),
								Body:   fmt.Sprintf("Body #%d.3.2", i),
							},
							Post{
								Header: fmt.Sprintf("Post #%d.3.3", i),
								Body:   fmt.Sprintf("Body #%d.3.3", i),
							},
						),
					},
				),
			}, sk)
		}
		return
	})
}
