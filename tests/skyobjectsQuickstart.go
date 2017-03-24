package main

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"text/tabwriter"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

type Age uint32

type Group struct {
	Name    string
	Leader  Reference  `skyobject:"schema=User"` // single User object
	Members References `skyobject:"schema=User"` // slice of Users
	Curator Dynamic    // a dynamic reference
}

type User struct {
	Name   string ``
	Age    Age    `json:"age"`
	Hidden int    `enc:"-"` // don't use the field
}

type List struct {
	Name     string
	Members  References `skyobject:"schema=User"` // alice of Users
	MemberOf []Group
}

type Man struct {
	Name    string
	Age     Age
	Seecret []byte
	Owner   Group // not a reference
	Friends List  // not a reference
}

func main() {
	// create database and container instance
	db := data.NewDB()
	c := skyobject.NewContainer(db)

	pk, sec := cipher.GenerateKeyPair() // public and secret keys

	// create new empty root object deteched from the container
	root := c.NewRoot(pk)

	// register schema of User{} with name "User",
	// thus, tags like `skyobject:"schema=User"` will refer to
	// schema of the User{}
	root.RegisterSchema("User", User{})

	// Inject the Group to the root
	root.Inject(Group{
		Name: "a group",
		Leader: root.Save(User{
			"Billy Kid", 16, 90,
		}),
		Members: root.SaveArray(
			User{"Bob Marley", 21, 0},
			User{"Alice Cooper", 19, 0},
			User{"Eva Brown", 30, 0},
		),
		Curator: root.Dynamic(Man{
			Name:    "Ned Kelly",
			Age:     28,
			Seecret: []byte("secret key"),
			Owner:   Group{},
			Friends: List{},
		}),
	})

	// Note: feel free to use empty references they are treated as "nil"

	root.Touch() // update timestamp of the root and increment seq number

	// attach the root to container

	// Add/replace with older. The AddRoot method also signs the root object
	c.AddRoot(root, sec)

	// Common tricks
	// 1) Create new root
	//      root := c.NewRoot(pubKey)
	//      root.Inject(Blah{})
	//      c.AddRoot(root, secKey)
	// 2) Replace existing root with new one (the same)
	//      root := c.NewRoot(pubKey)
	//      root.Inject(Blah{})
	//      c.AddRoot(root, secKey)
	// 3) Inject some object to existing root
	//      root := c.Root(pubKey)
	//      root.Inject(Blah{})
	//      root.Touch()
	//      c.AddRoot(root, secKey) // replace

	// prepare writer for inspect function
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', 0)

	// TODO: |
	//      \/

	// prepare schema printer for inspect fucntion
	printSchema := func(s *skyobject.Schema) {
		fmt.Printf("schema %s {", s.Name)
		if len(s.Fields) == 0 {
			fmt.Println("}")
			return
		}
		fmt.Println() // break line
		for _, sf := range s.Fields {
			fmt.Fprintln(tw, "   ",
				sf.Name, "\t",
				sf.Type, "\t",
				"`"+sf.Tag+"`", "\t",
				reflect.Kind(sf.Kind))
		}
		tw.Flush()
		fmt.Println("}")
	}

	// prepare fields printer for inspect function
	printFields := func(fields map[string]interface{}) {
		for k, v := range fields {
			fmt.Fprintln(tw, fmt.Sprintf("%s: \t%v", k, v))
		}
		tw.Flush()
	}

	// create function to inspecting
	inspect := func(s *skyobject.Schema, fields map[string]interface{},
		deep int, err error) error {

		fmt.Println("---")

		if err != nil {
			fmt.Println("Err: ", err)
		} else {
			printSchema(s)
			printFields(fields)
		}

		fmt.Println("---")
		return nil
	}

	// print the tree
	if err := c.Inspect(inspect); err != nil {
		log.Fatal(err)
		return
	}

	// database statistic
	// ------------------------
	// members:               4
	// leader:               +1
	// small group:          +1
	// man:                  +1
	// schema of Man         +1
	// schema of User:       +1
	// schema of SmallGroup: +1
	// ------------------------
	//                       10
	fmt.Println("===\n", db.Stat(), "\n===")
}
