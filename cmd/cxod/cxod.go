package main

import (
	"errors"
	"flag"
	"fmt"
	"net/rpc"
	"os"
	"os/signal"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/skycoin/src/mesh/app"
	"github.com/skycoin/skycoin/src/mesh/messages"
	"github.com/skycoin/skycoin/src/mesh/nodemanager"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/skyobject"
)

// defaults
const (
	Host        string = "[::]" // default host address of the server
	RemoteClose bool   = false
)

var ErrNotAllowed = errors.New("not allowed")

// Config represents configurations
type Config struct {
	Host        string
	RemoteClose bool
}

// NewConfig returns default configurations
func NewConfig() (c Config) {
	c.Host = Host
	c.RemoteClose = RemoteClose
}

func (c *Config) fromFlags() {
	flag.StringVar(&c.Host,
		"h",
		c.Host,
		"server host")
	flag.BoolVar(&c.RemoteClose,
		"rc",
		c.RemoteClose,
		"allow closing the server using RPC")
}

func waitInterrupt(quit <-chan struct{}) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	select {
	case <-sig:
	case <-quit:
	}
}

func main() {

	var (
		db   *data.DB             = data.NewDB()
		so   *skyobject.Container = skyobject.NewContainer(db)
		nd   *node.Node           = node.NewNode(db, so)
		c    Config               = NewConfig()
		quit chan struct{}        = make(chan struct{})
		rmx  sync.Mutex           // rpc lock
	)

	c.fromFlags()
	flag.Parse()

	meshnet := nodemanager.NewNetwork()
	defer nodemanager.Shutdown()

	address := meshnet.AddNewNode(c.Host)

	fmt.Println("host:           ", c.Host)
	fmt.Println("server address: ", address.Hex())

	_, err := app.NewServer(meshnet, address, Handler(db, so, nd, &rmx))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("started")

	// RPC

	rcvr := &RPCReceiver{
		lock:       &rmx,
		db:         db,
		so:         so,
		nd:         nd,
		allowClose: c.RemoteClose,
		quit:       quit,
	}

	// cxo related rpc procedures
	rpc.RegisterName("cxo", rcvr)

	rpcs := nodemanager.NewRPC()
	go rpcs.Serve()

	// waiting for SIGINT or termination using RPC

	waitInterrupt(quit)

}

func Handler(db *data.DB, so *skyobject.Container,
	nd *node.Node, rmx *sync.Mutex) (handler func([]byte) []byte) {

	want := make(skyobject.Set)

	handler = func(in []byte) (out []byte) {
		msg, err := node.Decode(in)
		if err != nil {
			fmt.Println("error decoding message:", err)
			return
		}

		// concurent access to databse, container and node
		rmx.Lock()
		defer rmx.Unlock()

		switch m := msg.(type) {
		case *node.SyncMsg:
			// add feeds of remote node to internal list
		case *node.RootMsg:
			for _, f := range nd.Feeds() {
				if f == m.Feed {
					ok, err := so.AddEncodedRoot(m.Root, m.Feed, m.Sig)
					if err != nil {
						fmt.Println("error adding root object: ", err)
						// TODO: close connection
						return
					}
					if !ok {
						return // older then existsing one
					}
					// todo: send updates to related connections except this one
					return
				}
			}
		case *node.RequestMsg:
			if data, ok := db.Get(m.Hash); ok {
				return node.Encode(&node.DataMsg{data})
			}
		case *node.DataMsg:
			hash := cipher.SumSHA256(m.Data)
			if _, ok := want[skyobject.Reference(hash)]; ok {
				db.Set(hash, m.Data)
				delete(want, hash)
			}
		default:
			fmt.Printf("unexpected message type: %T\n", msg)
		}
	}
	return
}

// RPC

type RPCReceiver struct {
	lock *sync.Mutex

	db *data.DB
	so *skyobject.Container
	nd *node.Node

	allowClose bool
	quit       chan struct{}
}

func (r *RPCReceiver) Tree(feed cipher.PubKey, reply *[]byte) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	//
	return
}

func (r *RPCReceiver) Want(feed cipher.PubKey, reply *struct{}) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	//
	return
}

func (r *RPCReceiver) Got(feed cipher.PubKey, reply *struct{}) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	//
	return
}

func (r *RPCReceiver) Info(_ struct{}, info *[]string) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	//
	return
}

func (r *RPCReceiver) Stat(_ struct{}, stat *data.Stat) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	stat = &r.db.Stat()
	return
}

func (r *RPCReceiver) Terminate(_ struct{}, _ *struct{}) (err error) {
	if !r.allowClose {
		err = ErrNotAllowed
		return
	}
	close(r.quit)
	return
}
