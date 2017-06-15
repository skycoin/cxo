package node_test

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/disiqueira/gotree"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/skyobject"
)

// tpyes

type User struct {
	Name string
	Age  uint32
}

type Group struct {
	Name   string
	Leader skyobject.Reference  `skyobject:"schema=cxo.User"`
	Users  skyobject.References `skyobject:"schema=cxo.User"`
}

func Example() {

	// feed and owner

	feed, owner := cipher.GenerateKeyPair()

	// launch

	src, err := SourceNode(feed, owner)
	if err != nil {
		log.Print(err)
		return
	}
	defer src.Close()

	dst, err := DestinationNode(feed, src.Pool().Address())
	if err != nil {
		log.Print(err)
		return
	}
	defer dst.Close()

	select {
	case <-src.Quiting():
	case <-dst.Quiting():
	}

}

func SourceNode(feed cipher.PubKey, owner cipher.SecKey) (src *node.Node,
	err error) {

	// registry

	reg := skyobject.NewRegistry()

	reg.Register("cxo.User", User{})
	reg.Register("cxo.Group", Group{})

	// config

	conf := node.NewNodeConfig()
	conf.Listen = "127.0.0.1:0" // arbitrary assignment
	conf.InMemoryDB = true      // use in-memory database
	conf.EnableRPC = false      // disable RPC
	conf.Log.Prefix = "[SRC] "
	// conf.Log.Debug = true

	// node

	if src, err = node.NewNodeReg(conf, reg); err != nil {
		return
	}

	if err = src.Start(); err != nil {
		src.Close()
		return
	}

	// feed

	src.Subscribe(nil, feed)

	// don't block

	go generate(src, feed, owner)

	return

}

func generate(src *node.Node, feed cipher.PubKey, owner cipher.SecKey) {

	defer src.Close()

	// for tis example (never need in real case)
	time.Sleep(1 * time.Second)

	// container
	cnt := src.Container()

	// generate groups
	root, err := cnt.NewRoot(feed, owner)
	if err != nil {
		log.Print(err)
		return
	}

	for i := 0; i < 2; i++ {
		generateGroup(i, cnt, root)

		// for this example
		time.Sleep(1 * time.Second)
	}
}

func generateGroup(i int, cnt *node.Container, root *node.Root) {
	cnt.LockGC()
	defer cnt.UnlockGC()

	_, _, err := root.Inject("cxo.Group", Group{
		Name: fmt.Sprint("Group #", i),
		Leader: root.Save(User{
			Name: "Elisabet Bathory",
			Age:  30,
		}),
		Users: root.SaveArray(
			User{fmt.Sprintf("Alice #%d", i), 19 + uint32(i)},
			User{fmt.Sprintf("Eva #%d", i), 21 + uint32(i)},
		),
	})

	if err != nil {
		log.Print(err)
	}

}

func DestinationNode(feed cipher.PubKey, address string) (dst *node.Node,
	err error) {

	// config

	conf := node.NewNodeConfig()
	conf.EnableListener = false // disable listener for this example
	conf.InMemoryDB = true      // use database in memory
	conf.EnableRPC = false
	conf.Log.Prefix = "[DST] "
	// conf.Log.Debug = true

	// while a root object and all related objects received
	conf.OnRootFilled = func(root *node.Root) {
		go printTree(root)
	}

	// node

	dst, err = node.NewNode(conf)
	if err != nil {
		return
	}

	if err = dst.Start(); err != nil {
		dst.Close()
		return
	}

	var src *gnet.Conn

	if src, err = dst.Pool().Dial(address); err != nil {
		log.Print(err)
		dst.Close()
		return
	}

	if err = dst.SubscribeResponse(src, feed); err != nil {
		log.Print(err)
		dst.Close()
		return
	}

	return

}

func printTree(root *node.Root) {

	fmt.Println("----")
	defer fmt.Println("----")

	var rt gotree.GTStructure

	hash := root.Hash()
	rt.Name = hash.String()

	rt.Items = rootItems(root)

	gotree.PrintTree(rt)
}

func rootItems(root *node.Root) (items []gotree.GTStructure) {
	vals, err := root.Values()
	if err != nil {
		items = []gotree.GTStructure{
			gotree.GTStructure{Name: "error: " + err.Error()},
		}
		return
	}
	for _, val := range vals {
		items = append(items, valueItem(val))
	}
	return
}

func valueItem(val *skyobject.Value) (item gotree.GTStructure) {
	if val.IsNil() {
		item.Name = "nil"
		return
	}

	var err error
	switch val.Kind() {
	case reflect.Struct:
		item.Name = "struct " + val.Schema().Name()

		err = val.RangeFields(func(name string,
			val *skyobject.Value) (_ error) {

			var field gotree.GTStructure
			field.Name = "(field) " + name
			field.Items = []gotree.GTStructure{
				valueItem(val),
			}
			item.Items = append(item.Items, field)
			return
		})
	case reflect.Ptr:
		// static or dyncmic reference
		item.Name = "*" // TODO: add reference hash
		if val, err = val.Dereference(); err == nil {
			item.Items = []gotree.GTStructure{
				valueItem(val),
			}
		} else {
			item.Items = []gotree.GTStructure{
				gotree.GTStructure{
					Name: "error: " + err.Error(),
				},
			}

		}
		return
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// TODO
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var u uint64
		if u, err = val.Uint(); err == nil {
			item.Name = fmt.Sprint(u)
		}
	case reflect.Float32, reflect.Float64:
		// TODO
	case reflect.Array:
		// TODO
	case reflect.Slice:
		// including skyobject.References
		item.Name = "[]" + val.Schema().Elem().Name()
		err = val.RangeIndex(func(i int, val *skyobject.Value) (_ error) {
			item.Items = append(item.Items, valueItem(val))
			return
		})
	case reflect.String:
		var s string
		if s, err = val.String(); err == nil {
			item.Name = s
		}

	}
	if err != nil {
		item.Name = "error: " + err.Error()
		item.Items = nil // clear
	}
	return
}
