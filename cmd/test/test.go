package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"time"

	au "github.com/logrusorgru/aurora"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/rpc/server"
	"github.com/skycoin/cxo/skyobject"
)

// note

// + start source
// + start nodes 1-5
// + start drain

// addresses of nodes

const (
	ADDRESS = "[::]"

	// communication

	SOURCE = 44000

	N1 = "44001"
	N2 = "44002"
	N3 = "44003"
	N4 = "44004"
	N5 = "44005"

	DRAIN = 44006

	// rpc

	RSOURCE = "55000"

	RN1 = "55001"
	RN2 = "55002"
	RN3 = "55003"
	RN4 = "55004"
	RN5 = "55005"

	RDRAIN = "55006"
)

// executables

const (
	CXOD = "../cxod/cxod"
	CLI  = "../cli/cli"
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

// srvice types

type pnode struct {
	name    string
	address string // communication address
	rpc     string // rpc addresss
	started bool
	cmd     *exec.Cmd
}

// globals

var (
	pub cipher.PubKey
	sec cipher.SecKey

	pipes []pnode = []pnode{
		{"N1", N1, RN1, false, nil},
		{"N2", N2, RN2, false, nil},
		{"N3", N3, RN3, false, nil},
		{"N4", N4, RN4, false, nil},
		{"N5", N5, RN5, false, nil},
	}
)

func init() {
	pub, sec = cipher.GenerateKeyPair()
}

// run tests

func main() {
	if len(os.Args) == 1 {
		fmt.Printf("Usage: %s <command> [<args>]\n", os.Args[0])
		fmt.Println("The commands are: ")
		fmt.Println(" source  - start source node")
		fmt.Println(" drain   - start drain node")
		fmt.Println(" pipe    - start pipe node")
		fmt.Println(" term    - terminate all nodes")
		return
	}

	switch os.Args[1] {
	case "source":
		source()
	case "drain":
		drain()
	case "pipe":
		pipe()
	case "term":
		term()
	default:
		fmt.Printf("%q is not valid command.\n", os.Args[1])
		os.Exit(2)
	}
}

// data source (cyan)
func source() {
	var code int // exit code
	defer func() { os.Exit(code) }()
	var err error
	var (
		db  *data.DB
		so  *skyobject.Container
		n   *node.Node
		rpc *server.Server

		nc node.Config
		rc server.Config
	)
	db = data.NewDB()
	so = skyobject.NewContainer(db)
	nc, rc = node.NewConfig(), server.NewConfig()
	nc.Name = au.Cyan("SOURCE").String()
	nc.Address = ADDRESS
	nc.Port = SOURCE
	nc.RemoteClose = true
	rc.Address = ADDRESS + ":" + RSOURCE
	n = node.NewNode(nc, db, so)
	n.Start()
	defer n.Close()
	n.Subscribe(pub)              // subscribe to the feed
	rpc = server.NewServer(rc, n) // , so)
	if err = rpc.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "error starting RPC:", err)
		code = 1
		return
	}
	defer rpc.Close()
	// generate data
	so.Register(
		"Board", Board{},
		"Thread", Thread{},
		"Post", Post{},
		"User", User{},
		"Man", Man{},
	)
	root := so.NewRoot()
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
	//
	time.Sleep(2 * time.Second)
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
	inject := exec.Cmd(CLI, "-a", ADDRESS+":"+RSOURCE, "-e", "inject "+
		hash.Hex()+" "+pub.Hex()+" "+sec.Hex())
	if err := inject.Run(); err != nil {
		log.Println(au.Cyan("[SOURCE TEST]"), "Inject CLI error:", err)
	}
	//
	waitInterrupt(n.Quiting())
}

// drain the root (magenta)
func drain() {
	var code int // exit code
	defer func() { os.Exit(code) }()
	var err error
	var (
		db  *data.DB
		so  *skyobject.Container
		n   *node.Node
		rpc *server.Server

		nc node.Config
		rc server.Config
	)
	db = data.NewDB()
	so = skyobject.NewContainer(db)
	nc, rc = node.NewConfig(), server.NewConfig()
	nc.Name = au.Magenta("DRAIN").String()
	nc.Address = ADDRESS
	nc.Port = DRAIN
	nc.RemoteClose = true
	rc.Address = ADDRESS + ":" + RDRAIN
	n = node.NewNode(nc, db, so)
	n.Start()
	defer n.Close()
	n.Subscribe(pub)              // subscribe to the feed
	rpc = server.NewServer(rc, n) // , so)
	if err = rpc.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "error starting RPC:", err)
		code = 1
		return
	}
	defer rpc.Close()
	//
	buf := new(bytes.Buffer)
	for {
		buf.Reset()

		time.Sleep(5 * time.Second)

		fmt.Fprintln(buf, "Inspect")
		fmt.Fprintln(buf, "=======")

		root := so.Root(pub)
		if root == nil {
			fmt.Println("  no root object")
			fmt.Println(au.Magenta(buf.String()))
			continue
		}

		vals, err := root.Values()
		if err != nil {
			fmt.Fprintln(w, "ERROR: ", err)
			fmt.Println(au.Magenta(buf.String()))
			continue
		}
		for _, val := range vals {
			fmt.Fprintln(w, "---")
			inspect(val, nil, "")
			fmt.Fprintln(w, "---")
		}
		fmt.Println(au.Magenta(buf.String()))
	}
	//
	waitInterrupt(n.Quiting())
}

func pipe() {
	for _, nd := range pipes {
		if err := nd.start(); err != nil {
			log.Println(au.Green("["+nd.name+"]"), "starting error: ", err)
			continue
		}
		// connect to source
		nd.connect(ADDRESS + ":" + strconv.Itoa(SOURCE))
		// connect to drain
		nd.connect(ADDRESS + ":" + strconv.Itoa(DRAIN))
		// connect to other
		for _, on := range pipes {
			if !nd.started {
				continue
			}
			nd.connect(ADDRESS + ":" + nd.address)
		}
	}
}

func (p *pnode) start() (err error) {
	p.cmd = exec.Cmd(CXOD,
		"-address", ADDRESS,
		"-port", p.address,
		"-rpc", "t",
		"-rpc-address", ADDRESS+":"+p.rpc,
		"-remote-close", "t",
		"-name", p.name,
	)
	if err = nd.cmd.Start(); err != nil {
		return
	}
	p.started = true
	return
}

func (p *pnode) connect(address string) {
	connect := exec.Cmd(CLI,
		"-a", ADDRESS+":"+p.rpc,
		"-e", "connect "+address)
	if err = connect.Run(); err != nil {
		log.Println(au.Green("["+p.name+"]"), "connect error:", err)
	}
}

func term() {
	addresses := []string{
		ADDRESS + ":" + RSOURCE,
		ADDRESS + ":" + RDRAIN,
		ADDRESS + ":" + RN1,
		ADDRESS + ":" + RN2,
		ADDRESS + ":" + RN3,
		ADDRESS + ":" + RN4,
		ADDRESS + ":" + RN5,
	}
	for _, a := range addresses {
		term := exec.Cmd(CLI, "-a", a, "term")
		if err := term.Run(); err != nil {
			log.Println(au.Red("[TERM TEST]"), "term error:", err)
		}
	}
}

// service functions

func waitInterrupt(quit <-chan struct{}) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	select {
	case <-sig:
	case <-quit:
	}
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
		inspect(d, err, prefix+"  ")
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
			inspect(iv, err, prefix+"  ")
		}
	case reflect.Struct:
		fmt.Fprintf(w, "<struct %s>\n", val.Schema().String())
		err = val.RangeFields(func(name string, val *skyobject.Value) error {
			fmt.Fprint(w, prefix, name, ": ")
			inspect(val, nil, prefix+"  ")
			return nil
		})
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
	}
}
