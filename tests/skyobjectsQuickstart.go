package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

type Age uint32

type Group struct {
	Name    string
	Leader  skyobject.Reference  `skyobject:"schema=User"` // single User object
	Members skyobject.References `skyobject:"schema=User"` // slice of Users
	Curator skyobject.Dynamic    // a dynamic reference
}

type User struct {
	Name   string ``
	Age    Age    `json:"age"`
	Hidden int    `enc:"-"` // don't use the field
}

type List struct {
	Name     string
	Members  skyobject.References `skyobject:"schema=User"` // alice of Users
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
	c.Register("User", User{})

	// Inject the Group to the root
	root.Inject(Group{
		Name: "a group",
		Leader: c.Save(User{
			"Billy Kid", 16, 90,
		}),
		Members: c.SaveArray(
			User{"Bob Marley", 21, 0},
			User{"Alice Cooper", 19, 0},
			User{"Eva Brown", 30, 0},
		),
		Curator: c.Dynamic(Man{
			Name:    "Ned Kelly",
			Age:     28,
			Seecret: []byte("secret key"),
			Owner:   Group{},
			Friends: List{},
		}),
	})

	// Inject another object using its hash
	usrHash := c.Save( // save an object to get its hash
		c.Dynamic(User{ // the object must be Dynamic
			Name: "Old Uncle Tom Cobley and all",
			Age:  89,
		}),
	)
	root.InjectHash(usrHash) // inject

	// Feel free to use empty references they are treated as "nil"
	// But for InjectHash an empty reference is illegal
	if err := root.InjectHash(skyobject.Reference{}); err != nil {
		log.Print("InjectHash(Reference{}) error: ", err)
	}

	root.Touch() // update timestamp of the root and increment seq number

	// attach the root to the container

	// Add/replace with older. The AddRoot method also signs the root object
	c.AddRoot(root, sec)

	// Common tricks
	// 1) Create new root
	//      root := c.NewRoot(pubKey)
	//      root.Inject(Blah{})
	//      c.AddRoot(root, secKey)
	// 2) Replace existing root with new one (replacing references of the root)
	//      root := c.NewRoot(pubKey)
	//      root.Inject(Blah{})
	//      c.AddRoot(root, secKey)
	// 3) Inject some object to existing root
	//      root := c.Root(pubKey)
	//      root.Inject(Blah{})
	//      root.Touch()
	//      c.AddRoot(root, secKey) // replace

	// print the tree
	if vals, err := root.Values(); err != nil {
		log.Print(err)
	} else {
		for _, val := range vals {
			fmt.Println("---")
			inspect(val, nil, "")
			fmt.Println("---")
		}
	}

	//
	fmt.Println("===\n", db.Stat(), "\n===")
}

// create function to inspecting
func inspect(val *skyobject.Value, err error, prefix string) {
	if err != nil {
		fmt.Println(err)
		return
	}
	// const (
	// 	cross  = "├── "
	// 	road   = "│   "
	// 	corner = "└── "
	// 	blank  = "    "
	// )
	switch val.Kind() {
	case reflect.Invalid: // nil
		fmt.Println("nil")
	case reflect.Ptr: // reference
		fmt.Println("<reference>")
		fmt.Print(prefix + "  ")
		d, err := val.Dereference()
		inspect(d, err, prefix+"  ")
	case reflect.Bool:
		if b, err := val.Bool(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(b)
		}
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, err := val.Int(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(i)
		}
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if u, err := val.Uint(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(u)
		}
	case reflect.Float32, reflect.Float64:
		if f, err := val.Float(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(f)
		}
	case reflect.String:
		if s, err := val.String(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("%q\n", s)
		}
	case reflect.Array, reflect.Slice:
		if val.Kind() == reflect.Array {
			fmt.Printf("<array %s>\n", val.Schema().String())
		} else {
			fmt.Printf("<slice %s>\n", val.Schema().String())
		}
		el, err := val.Schema().Elem()
		if err != nil {
			fmt.Println(err)
			break
		}
		if el.Kind() == reflect.Uint8 {
			fmt.Print(prefix)
			b, err := val.Bytes()
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(hex.EncodeToString(b))
			}
			break
		}
		ln, err := val.Len()
		if err != nil {
			fmt.Println(err)
			return
		}
		for i := 0; i < ln; i++ {
			iv, err := val.Index(i)
			fmt.Print(prefix)
			inspect(iv, err, prefix+"  ")
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
			return
		}
	}
}
