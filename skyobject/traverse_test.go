package skyobject

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"text/tabwriter"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func ExampleContainer_Inspect() {
	//
	// nessesary types
	//

	type User struct {
		Name   string
		Age    int64
		Hidden string `enc:"-"` // ignore the field
	}

	type Man struct {
		Name   string
		Height int64
		Weight int64
	}

	type SamllGroup struct {
		Name     string
		Leader   cipher.SHA256   `skyobject:"schema=User"` // single User
		Outsider cipher.SHA256   // not a reference
		FallGuy  Dynamic         // dynamic href
		Members  []cipher.SHA256 `skyobject:"array=User"` // array of Users
	}

	// create database and container instance
	db := data.NewDB()
	c := NewContainer(db)

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
	printSchema := func(s *Schema) {
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
		// sorted by key
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if k == "Members" {
				fmt.Fprintln(tw, fmt.Sprintf("%s: \t<binary>", k))
				continue
			}
			fmt.Fprintln(tw, fmt.Sprintf("%s: \t%v", k, fields[k]))
		}
		tw.Flush()
	}

	// create function to inspecting
	inspect := func(s *Schema, fields map[string]interface{}, deep int,
		err error) error {

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
		// fatal error
		return
	}

	// Output:
	//
	// ---
	// schema SamllGroup {
	//     Name            string  ``                            string
	//     Leader          SHA256  `skyobject:"schema=User"`     array
	//     Outsider        SHA256  ``                            array
	//     FallGuy.Schema  SHA256  `skyobject:"dynamic_schema"`  array
	//     FallGuy.ObjKey  SHA256  `skyobject:"dynamic_objkey"`  array
	//     Members                 `skyobject:"array=User"`      slice
	// }
	// FallGuy.ObjKey: dd4a77107ed51ab122bb85ffc43fb4fc59fba545528430107ad70b702c08ca1c
	// FallGuy.Schema: 9d53ca9f46576992f0bdb8fd355985515107026e06d091673fa327e4e1d73c9f
	// Leader:         03d9a33bd9e53cbc06db0fcb7ac015fc3ca276b291c425b3b786708753f9a604
	// Members:        <binary>
	// Name:           Average small group
	// Outsider:       0001020300000000000000000000000000000000000000000000000000000000
	// ---
	// ---
	// schema User {
	//     Name  string  ``  string
	//     Age   int64   ``  int64
	// }
	// Age:  16
	// Name: Billy Kid
	// ---
	// ---
	// schema Man {
	//     Name    string  ``  string
	//     Height  int64   ``  int64
	//     Weight  int64   ``  int64
	// }
	// Height: 182
	// Name:   Bob
	// Weight: 82
	// ---
	// ---
	// schema User {
	//     Name  string  ``  string
	//     Age   int64   ``  int64
	// }
	// Age:  21
	// Name: Alice
	// ---
	// ---
	// schema User {
	//     Name  string  ``  string
	//     Age   int64   ``  int64
	// }
	// Age:  22
	// Name: Eva
	// ---
	// ---
	// schema User {
	//     Name  string  ``  string
	//     Age   int64   ``  int64
	// }
	// Age:  23
	// Name: John
	// ---
	// ---
	// schema User {
	//     Name  string  ``  string
	//     Age   int64   ``  int64
	// }
	// Age:  24
	// Name: Michel
	// ---
	//
}
