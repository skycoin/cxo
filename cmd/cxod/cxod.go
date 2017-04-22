package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/rpc"
	"os"
	"os/signal"
	"reflect"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/skycoin/src/mesh/app"
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
	return
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
	defer meshnet.Shutdown()

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
			// TODO: terminate connection
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
					// TODO: send updates to related connections except this one
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
				delete(want, skyobject.Reference(hash))
			}
		default:
			fmt.Printf("unexpected message type: %T\n", msg)
		}
		return
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

func (r *RPCReceiver) Subscribe(feed cipher.PubKey,
	subscribed *bool) (err error) {

	r.lock.Lock()
	defer r.lock.Unlock()

	*subscribed = r.nd.Subscribe(feed)

	// TODO: send SyncMsg to peers

	return
}

func (r *RPCReceiver) Unsubscribe(feed cipher.PubKey,
	unsubscribed *bool) (err error) {

	r.lock.Lock()
	defer r.lock.Unlock()

	*unsubscribed = r.nd.Unsubscribe(feed)

	// TODO: send SyncMsg to peers

	return
}

func (r *RPCReceiver) Tree(feed cipher.PubKey, tree *[]byte) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	buf := new(bytes.Buffer)

	root := r.so.Root(feed)
	if root == nil {
		err = node.ErrNoRootObject
		return
	}

	var vs []*skyobject.Value
	if vs, err = root.Values(); err != nil {
		return
	}

	buf.WriteString("  Root object: " + feed.Hex() + "\n")

	for _, val := range vs {
		inspect(buf, val, nil, "")
	}

	*tree = buf.Bytes()

	return
}

// create function for inspecting
func inspect(w io.Writer, val *skyobject.Value, err error, prefix string) {
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	switch val.Kind() {
	case reflect.Invalid: // nil
		fmt.Fprintln(w, "nil")
	case reflect.Ptr: // reference
		fmt.Fprintln(w, "<reference>")
		fmt.Fprint(w, prefix+"  ")
		d, err := val.Dereference()
		inspect(w, d, err, prefix+"  ")
	case reflect.Bool:
		if b, err := val.Bool(); err != nil {
			fmt.Fprintln(w, err)
		} else {
			fmt.Fprintln(w, b)
		}
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, err := val.Int(); err != nil {
			fmt.Fprintln(w, err)
		} else {
			fmt.Fprintln(w, i)
		}
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if u, err := val.Uint(); err != nil {
			fmt.Fprintln(w, err)
		} else {
			fmt.Fprintln(w, u)
		}
	case reflect.Float32, reflect.Float64:
		if f, err := val.Float(); err != nil {
			fmt.Fprintln(w, err)
		} else {
			fmt.Fprintln(w, f)
		}
	case reflect.String:
		if s, err := val.String(); err != nil {
			fmt.Fprintln(w, err)
		} else {
			fmt.Fprintf(w, "%q\n", s)
		}
	case reflect.Array, reflect.Slice:
		if val.Kind() == reflect.Array {
			fmt.Fprintf(w, "<array %s>\n", val.Schema().String())
		} else {
			fmt.Fprintf(w, "<slice %s>\n", val.Schema().String())
		}
		el, err := val.Schema().Elem()
		if err != nil {
			fmt.Fprintln(w, err)
			break
		}
		if el.Kind() == reflect.Uint8 {
			fmt.Fprint(w, prefix)
			b, err := val.Bytes()
			if err != nil {
				fmt.Fprintln(w, err)
			} else {
				fmt.Fprintln(w, hex.EncodeToString(b))
			}
			break
		}
		ln, err := val.Len()
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		for i := 0; i < ln; i++ {
			iv, err := val.Index(i)
			fmt.Fprint(w, prefix)
			inspect(w, iv, err, prefix+"  ")
		}
	case reflect.Struct:
		fmt.Fprintf(w, "<struct %s>\n", val.Schema().String())
		err = val.RangeFields(func(name string, val *skyobject.Value) error {
			fmt.Fprint(w, prefix, name, ": ")
			inspect(w, val, nil, prefix+"  ")
			return nil
		})
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
	}
}

func (r *RPCReceiver) Want(feed cipher.PubKey,
	list *[]cipher.SHA256) (err error) {

	r.lock.Lock()
	defer r.lock.Unlock()

	var wn []cipher.SHA256
	if wn, err = r.nd.Want(feed); err == nil {
		*list = wn
	}

	return
}

func (r *RPCReceiver) Got(feed cipher.PubKey,
	list *[]cipher.SHA256) (err error) {

	r.lock.Lock()
	defer r.lock.Unlock()

	var gt []cipher.SHA256
	if gt, err = r.nd.Got(feed); err == nil {
		*list = gt
	}

	return
}

func (r *RPCReceiver) Feeds(_ struct{}, list *[]cipher.PubKey) (_ error) {

	r.lock.Lock()
	defer r.lock.Unlock()

	*list = r.nd.Feeds()

	return
}

func (r *RPCReceiver) Stat(_ struct{}, stat *data.Stat) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	var s data.Stat = r.db.Stat()
	stat = &s
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
