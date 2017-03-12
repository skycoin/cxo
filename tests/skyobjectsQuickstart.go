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

type User struct {
	Name   string
	Age    int64
	Hidden string `enc:"-"`
}

type Man struct {
	Name   string
	Height int64
	Weight int64
}

type SamllGroup struct {
	Name     string
	Leader   cipher.SHA256 `skyobject:"schema=User"` // single User
	Outsider cipher.SHA256 // not a reference
	FallGuy  cipher.SHA256 `skyobject:"dynamic"`    // dynamic reference
	Members  cipher.SHA256 `skyobject:"array=User"` // aray of Users
}

func main() {
	// create database and container instance
	db := data.NewDB()
	c := skyobject.NewContainer(db)

	// create new empty root (that is, actually, wrapper of root, but who cares)
	root := c.NewRoot()
	// register schema of User{} with name "User",
	// thus, tags like `skyobject:"href,schema=User"` will refer to
	// schema of User{}
	root.Register("User", User{})

	// Set SmallGroup as root
	root.Set(SamllGroup{
		Name: "Average small group",
		// Save the Leader and get its key
		Leader: c.Save(User{"Billy Kid", 16, ""}),
		// Outsider is not a reference, it's just a SHA256
		Outsider: cipher.SHA256{0, 1, 2, 3},
		// Create and save dynamic reference to the Man
		FallGuy: c.SaveDynamicHref(Man{"Bob", 182, 82}),
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
	// members:               4
	// leader:               +1
	// small group:          +1
	// dynamic reference     +1
	// array of Users        +1
	// man:                  +1
	// schema of Man         +1
	// schema of User:       +1
	// schema of SmallGroup: +1
	// ------------------------
	//                       12
	fmt.Println("===\n", db.Stat(), "\n===")
}
