package main

import (
	//"fmt"
	"log"

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
	Leader   cipher.SHA256 `skyobject:"href,schema=User"`
	Outsider cipher.SHA256 // not a reference
	//FallGuy  skyobject.DynamicHref `skyobject:"href"`
	Members []cipher.SHA256 `skyobject:"href,schema=User"`
}

func main() {
	c := skyobject.NewContainer(data.NewDB())

	root := c.NewRoot()
	root.Register("User", User{})

	root.Set(SamllGroup{
		Name:     "Average small group",
		Leader:   c.Save(User{"Billy Kid", 16, ""}),
		Outsider: cipher.SHA256{0, 1, 2, 3},
		//FallGuy:  c.NewDynamicHref(Man{"Bob", 182, 82}),
		Members: c.SaveArray(
			User{"Alice", 21, ""},
			User{"Eva", 22, ""},
			User{"Jhon", 23, ""},
			User{"Michel", 24, ""},
		),
	})

	c.SetRoot(root)

	if err := c.Inspect(); err != nil {
		log.Fatal(err)
	}

	//
	//
	//

	/*
		// Retrieve the 'Children' field of the Parent object as an 'ObjRef'.
		childrenRef, e := parentRef.GetFieldAsObj("Children")
		if e != nil {
			fmt.Println("Unable to retrieve field. Error:", e)
		}

		// We know that 'childrenRef' references a HashArray, not a HashObject.
		// We can retrieve all the array elements using the `GetValuesAsObjArray` member function of 'childrenRef'.
		childRefArray, e := childrenRef.GetValuesAsObjArray()
		if e != nil {
			fmt.Println("Unable to retrieve elements. Error:", e)
		}

		// 'childRefArray' is of type '[]ObjRef'.
		// We can now deserialize all the child references into child objects.
		fmt.Println("Children:")

		for _, childRef := range childRefArray {
			var child Child
			childRef.Deserialize(&child)

			fmt.Println(" -", child.Name)
		}

		// Schemas.
		fmt.Println("Schemas:")
		schemas := container.GetSchemas()
		for _, schema := range schemas {
			fmt.Println("-", schema.Name)
		}

		// Inspect.
		fmt.Println("Inspect:")
		container.Inspect()

		func() {
			// Get all keys of objects with Schema type 'Child'.
			schemaKey, ok := container.GetSchemaKey("Child")
			if ok == false {
				fmt.Println("Woops! This Schema doesn't exist in the container.")
				return
			}
			childrenKeys := container.GetAllBySchema(schemaKey)

			// Get all children.
			fmt.Println("Children:")

			for _, childKey := range childrenKeys {
				var child Child
				childRef := container.GetObjRef(childKey)
				childRef.Deserialize(&child)

				fmt.Println(" -", child.Name)
			}
		}()
	*/
}
