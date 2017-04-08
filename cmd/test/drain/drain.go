package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node"
	nlog "github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/rpc/server"
	"github.com/skycoin/cxo/skyobject"
)

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

		// flags
		pubf string
		addr string
		rpca string

		sig chan os.Signal
		buf *bytes.Buffer
	)

	log.SetFlags(log.Lshortfile | log.Ltime)
	log.SetPrefix("[DRAIN] ")

	// parse flags
	flag.StringVar(&pubf, "pub", "", "public key (feed)")
	flag.StringVar(&addr, "a", "[::]44006", "address")
	flag.StringVar(&rpca, "r", "[::]55006", "rpc address")

	flag.Parse()

	// parse public key
	if pub, err = cipher.PubKeyFromHex(pubf); err != nil {
		log.Fatal(err)
	}

	db = data.NewDB()
	so = skyobject.NewContainer(db)

	nc, rc = node.NewConfig(), server.NewConfig()
	nc.Logger = nlog.NewLogger("[DRAIN]", true)
	nc.Config.Logger = nlog.NewLogger("[P:DRAIN]", true) // pool
	nc.Listen = addr

	rc.Address = rpca

	n = node.NewNode(nc, db, so)
	n.Start()
	defer n.Close()

	n.Subscribe(pub) // subscribe to the feed

	// rpc
	rpc = server.NewServer(rc, n) // , so)
	if err = rpc.Start(); err != nil {
		log.Fatal("error starting RPC:", err)
	}
	defer rpc.Close()

	sig = make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	buf = new(bytes.Buffer)
	lastStat := db.Stat()
	for {
		buf.Reset()

		// SIGINT, remote close, or print the tree every 5 seconds
		// if the tree was updated
		select {
		case <-sig:
			return
		case <-n.Quiting():
			return
		default:
			if stat := db.Stat(); lastStat.Total != stat.Total {
				lastStat = stat
				// print updated tree
			} else {
				continue
			}
		}

		n.Execute(func() {
			fmt.Fprintln(buf, "Inspect")
			fmt.Fprintln(buf, "=======")

			fmt.Fprintln(buf, "Database:", lastStat.String())

			fmt.Fprintln(buf, "Objects tree")

			root := so.Root(pub)
			if root == nil {
				fmt.Fprintln(buf, "  no root object")
				fmt.Println(buf.String())
				return
			}

			vals, err := root.Values()
			if err != nil {
				fmt.Fprintln(buf, "ERROR: ", err)
				fmt.Println(buf.String())
				return
			}
			for _, val := range vals {
				fmt.Fprintln(buf, "---")
				inspect(buf, val, nil, "")
				fmt.Fprintln(buf, "---")
			}
			fmt.Println(buf.String())
		})

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
