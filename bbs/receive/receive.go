package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/skyobject"
)

func main() {
	var (
		serverAddress string = "[::]:8998"
		publicKey     string

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

	flag.Parse()

	pk, err := cipher.PubKeyFromHex(publicKey)
	if err != nil {
		log.Fatal(err)
	}

	c, err := node.NewClient(cc, skyobject.NewContainer(nil))
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

	go printTree(c, pk) // print tree of the feed

	waitInterrupt() // exit on SIGINT

	return
}

func waitInterrupt() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

func printTree(c *node.Client, pk cipher.PubKey) {
	for {
		<-time.After(5 * time.Second)
		fmt.Println("---")
		fmt.Println("---")
		fmt.Println("---")

		root := c.Container().LastRoot(pk)
		if root == nil {
			fmt.Println("empty root")
			continue
		}
		vals, err := root.Values()
		if err != nil {
			fmt.Println("error: ", err)
			continue
		}
		for _, val := range vals {
			inspect(val, err, "")
		}
	}
}

// create function for inspecting
func inspect(val *skyobject.Value, err error, prefix string) {
	if err != nil {
		fmt.Println(err)
		return
	}
	switch val.Kind() {
	case reflect.Invalid: // nil
		fmt.Println("nil")
	case reflect.Ptr: // reference
		fmt.Println("<reference>")
		fmt.Print(prefix + "  ")
		d, err := val.Dereference()
		inspect(d, err, prefix+"  ")
	case reflect.Bool:
		b, err := val.Bool()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(b)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := val.Int()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(i)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := val.Uint()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(u)
	case reflect.Float32, reflect.Float64:
		f, err := val.Float()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(f)
	case reflect.String:
		s, err := val.String()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("%q\n", s)
	case reflect.Array, reflect.Slice:
		if val.Kind() == reflect.Array {
			fmt.Printf("<array %s>\n", val.Schema().String())
		} else {
			fmt.Printf("<slice %s>\n", val.Schema().String())
		}
		if val.Schema().Elem().Kind() == reflect.Uint8 {
			fmt.Print(prefix)
			b, err := val.Bytes()
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(hex.EncodeToString(b))
			}
			return
		}
		err := val.RangeIndex(func(_ int, d *skyobject.Value) (_ error) {
			fmt.Print(prefix)
			inspect(d, err, prefix+"  ")
			return
		})
		if err != nil {
			fmt.Println(err)
		}
	case reflect.Struct:
		fmt.Printf("<struct %s>\n", val.Schema().String())
		err = val.RangeFields(func(name string, val *skyobject.Value) error {
			fmt.Print(prefix, name, ": ")
			inspect(val, nil, prefix+"  ")
			return nil
		})
		if err != nil {
			fmt.Println(err)
		}
	}
}
