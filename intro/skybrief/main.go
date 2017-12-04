package main

import (
	"fmt"
	"log"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/registry"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type User struct {
	Name   string
	Age    uint32
	Hidden []byte `enc:"-"`
}

type Group struct {
	Name string

	Leader  registry.Ref  `skyobject:"schema=intro.User"`
	Members registry.Refs `skyobject:"schema=intro.User"`

	Developer registry.Dynamic
}

type Man struct {
	Name   string
	GitHub string
}

func main() {

	var reg = registry.NewRegistry(func(r *registry.Reg) {
		r.Register("intro.User", User{})
		r.Register("intro.Group", Group{})
		r.Register("intro.Man", Man{})
	})

	var conf = skyobject.NewConfig() // get default config
	conf.InMemoryDB = true           // for the example

	var c, err = skyobject.NewContainer(conf)

	if err != nil {
		log.Fatal(err)
	}

	var pk, sk = cipher.GenerateKeyPair()

	if err = c.AddFeed(pk); err != nil {
		log.Fatal(err)
	}

	var up *skyobject.Unpack
	if up, err = c.Unpack(sk, reg); err != nil {
		log.Fatal(err)
	}

	var r = new(registry.Root)

	r.Pub = pk
	r.Nonce = 90210
	r.Descriptor = []byte("intro, version: 3")

	var alice, eva, ammy = User{
		Name: "Alice",
		Age:  19,
	}, User{
		Name: "Eva",
		Age:  21,
	}, User{
		Name: "Ammy",
		Age:  23,
	}

	var group Group

	group.Name = "vdpg"

	if err = group.Leader.SetValue(up, alice); err != nil {
		log.Fatal(err)
	}

	if err = group.Members.AppendValues(up, &eva, &ammy); err != nil {
		log.Fatal(err)
	}

	// it can be done manually
	group.Developer.SetValue(up, &Man{
		Name:   "kostyarin",
		GitHub: "logrusorgru",
	})

	// set schema for the group.Developer
	var sch registry.Schema
	if sch, err = reg.SchemaByName("intro.Man"); err != nil {
		log.Fatal(err)
	}

	group.Developer.Schema = sch.Reference()

	var dr registry.Dynamic

	if err = dr.SetValue(up, &group); err != nil {
		log.Fatal(err)
	}

	if sch, err = reg.SchemaByName("intro.Group"); err != nil {
		log.Fatal(err)
	}

	dr.Schema = sch.Reference()

	r.Refs = append(r.Refs, dr)

	if err = c.Save(up, r); err != nil {
		log.Fatal(err)
	}

	var tree string
	if tree, err = r.Tree(up); err != nil {
		log.Fatal(err)
	}

	fmt.Println(tree)

}

func add(up *skyobject.Unpack, val interface{}) (hash cipher.SHA256) {
	var err error
	if hash, err = up.Add(encoder.Serialize(val)); err != nil {
		log.Fatal(err)
	}
	return
}
