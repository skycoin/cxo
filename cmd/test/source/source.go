package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/rpc/server"
	"github.com/skycoin/cxo/skyobject"
)

const (
	CLI = "../../cli/cli"
)

// types to share

type Board struct {
	Head    string
	Threads skyobject.References `skyobject:"schema=Thread"`
	Owner   skyobject.Dynamic
}

type Thread struct {
	Head  string
	Posts skyobject.References `skyobject:"schema=Post"`
}

type Post struct {
	Head string
	Body string
}

type User struct {
	Name   string
	Age    int32
	Hidden string `enc:"-"`
}

type Man struct {
	Name string
	Age  int32
}

func main() {
	var (
		err error

		db  *data.DB
		so  *skyobject.Container
		n   *node.Node
		rpc *server.Server

		nc node.Config
		rc server.Config

		pub cipher.PubKey
		sec cipher.SecKey

		// flags
		pubf    string
		secf    string
		addr    string
		port    int
		rpcPort int

		rpcAddress string
	)

	log.SetFlags(log.Lshortfile | log.Ltime)
	log.SetPrefix("[SOURCE] ")

	// parse flags
	flag.StringVar(&pubf, "pub", "", "public key (feed)")
	flag.StringVar(&secf, "sec", "", "secret key (owner)")
	flag.StringVar(&addr, "a", "[::]", "address")
	flag.IntVar(&port, "p", 44000, "port")
	flag.IntVar(&rpcPort, "r", 55000, "rpc port")

	flag.Parse()

	// parse public and secret keys
	if pub, err = cipher.PubKeyFromHex(pubf); err != nil {
		log.Fatal(err)
	}
	if sec, err = cipher.SecKeyFromHex(secf); err != nil {
		log.Fatal(err)
	}

	rpcAddress = addr + ":" + strconv.Itoa(rpcPort)

	// container
	db = data.NewDB()
	so = skyobject.NewContainer(db)

	// node
	nc, rc = node.NewConfig(), server.NewConfig()
	nc.Name = "SOURCE"
	nc.Address = addr
	nc.Port = uint16(port)
	nc.RemoteClose = true
	rc.Address = rpcAddress
	n = node.NewNode(nc, db, so)
	n.Start()
	defer n.Close()

	// subscribe to the feed
	n.Subscribe(pub)

	// rpc
	rpc = server.NewServer(rc, n) // , so)
	if err = rpc.Start(); err != nil {
		log.Fatal("error starting RPC:", err)
	}
	defer rpc.Close()

	// register used schemas
	so.Register(
		"Board", Board{},
		"Thread", Thread{},
		"Post", Post{},
		"User", User{},
		"Man", Man{},
	)

	// create and fill down the feed
	root := so.NewRoot(pub)
	root.Inject(Board{
		Head: "Board #1",
		Threads: so.SaveArray(
			Thread{
				Head: "Thread #1.1",
				Posts: so.SaveArray(
					Post{
						Head: "Post #1.1.1",
						Body: "Body #1.1.1",
					},
					Post{
						Head: "Post #1.1.2",
						Body: "Body #1.1.2",
					},
				),
			},
			Thread{
				Head: "Thread #1.2",
				Posts: so.SaveArray(
					Post{
						Head: "Post #1.2.1",
						Body: "Body #1.2.1",
					},
					Post{
						Head: "Post #1.2.2",
						Body: "Body #1.2.2",
					},
				),
			},
			Thread{
				Head: "Thread #1.3",
				Posts: so.SaveArray(
					Post{
						Head: "Post #1.3.1",
						Body: "Body #1.3.1",
					},
					Post{
						Head: "Post #1.3.2",
						Body: "Body #1.3.2",
					},
				),
			},
			Thread{
				Head: "Thread #1.4",
				Posts: so.SaveArray(
					Post{
						Head: "Post #1.4.1",
						Body: "Body #1.4.1",
					},
					Post{
						Head: "Post #1.4.2",
						Body: "Body #1.4.2",
					},
				),
			},
		),
		Owner: so.Dynamic(User{
			Name:   "Billy Kid",
			Age:    16,
			Hidden: "secret",
		}),
	})
	so.AddRoot(root, sec)
	n.Share(pub)

	// wait some time to be sure that the rpc was started
	time.Sleep(1 * time.Second)
	// and inject another one using CLI
	hash := so.Save(so.Dynamic(
		Board{
			"Board #2",
			so.SaveArray(
				Thread{
					"Thread #2.1",
					so.SaveArray(Post{"Post #2.1.1", "Body #2.1.1"}),
				},
			),
			so.Dynamic(Man{
				Name: "Tom Cobley",
				Age:  89,
			}),
		},
	))

	// ../cli/cli -a "[::]:4000" -e "inject HASH PUB SEC"
	inject := exec.Command(CLI,
		"-a", rpcAddress,
		"-e", "inject "+hash.String()+" "+pub.Hex()+" "+sec.Hex())
	if err = inject.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			log.Print("CLI stderr: ", string(ee.Stderr))
		}
		log.Print("injecting error: ", err)
	}

	waitInterrupt(n.Quiting())
}

// service functions

func waitInterrupt(q <-chan struct{}) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	select {
	case <-sig:
	case <-q:
	}
}
