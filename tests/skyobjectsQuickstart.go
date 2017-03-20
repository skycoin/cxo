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
	Leader  Reference  `skyobject:"schema=User"`
	Members References `skyobject:"schema=User"`
	Curator Dynamic
}

type User struct {
	Name   string ``
	Age    Age    `json:"age"`
	Hidden int    `enc:"-"`
}

type List struct {
	Name     string
	Members  References `skyobject:"schema=User"`
	MemberOf []Group
}

type Man struct {
	Name    string
	Age     Age
	Seecret []byte
	Owner   Group
	Friends List
}

func main() {
	// create database and container instance
	db := data.NewDB()
	c := skyobject.NewContainer(db)

	pk, _ := cipher.GenerateKeyPair() // public key

	// create new empty root (that is, actually, wrapper of root, but who cares)
	root := c.NewRoot(pk)
	// register schema of User{} with name "User",
	// thus, tags like `skyobject:"schema=User"` will refer to
	// schema of User{}
	root.RegisterSchema("User", User{})

	// Set SmallGroup as root
	root.Set(SamllGroup{
		Name: "Average small group",
		// Save the Leader and get its key
		Leader: c.Save(User{"Billy Kid", 16, ""}),
		// Outsider is not a reference, it's just a SHA256
		Outsider: cipher.SHA256{0, 1, 2, 3},
		// Create and save dynamic reference to the Man
		FallGuy: c.NewDynamic(Man{"Bob", 182, 82}),
		// Save objects and get array of their references
		Members: c.SaveArray(
			User{"Alice", 21, ""},
			User{"Eva", 22, ""},
			User{"John", 23, ""},
			User{"Michel", 24, ""},
		),
	})

	// Set the root as root of the container
	c.SetRoot(root)

	// prepare writer for inspect function
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', 0)

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
