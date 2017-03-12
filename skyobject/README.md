Skyobject
=========

[![GoDoc](https://godoc.org/github.com/skycoin/cxo/skyobject?status.svg)](https://godoc.org/github.com/skycoin/cxo/skyobject)

# About

Use `Container` to manage skyobjects. Any object can contains references to
another object. But we need to register type of objects we refer and provide
appropriate tags (except dynamic references).

+ `skyobject:"schema=User"` refers to single object, type of which is User
+ `skyobject:"array=User" refers to array of objects
+ `skyobject.Dynamic` refers to dynamic reference

Anyway the field type must be `cipher.SHA256`. For example

```go

type Wrap struct {
	Name string
	Single cipher.SHA256 `skyobject:"schema=Nest"`
	Array  []cipher.SHA256 `skyobject:"array=Element"`
	Any    skyobject.Dynamic
	Not    cipher.SHA256 // not a reference
}

```

How to use the Container (base on the `Wrap` above):

```go

// declaration of the Wrap

type Nest struct {
	Name string
}

type Element struct {
	Value int64
}

type Random struct {
	Seed int64
	Last int64
}

db := data.NewDB()
c := skyobject.NewContainer(db)

// create root object wrapper
root := c.NewRoot()

root.Register("Nest", Nest{})
root.Register("Element", Element{})

root.Set(Wrap{
	Name: "a name",
	Single: c.Save(Nest{
		"name of the Nest",
	}),
	Array: c.SaveArray(
		Element{1},
		Element{2},
	),
	Any: c.NewDynamic(Random{234, 456}), // not registered
	Not: cipher.SHA256{0,1,2,3},
})

// we need to set the root as root of the container
if !c.SetRoot(root) {
	// our root is older than root of the container,
	// but for the example it never happens, because
	// container is empty
}


```

Okay. There is container, there is root object. There are objects it has.
Let's explore the Container

+ Obtain a root object

```go
root := c.Root

if root == nil {
	// we haven't got a root
	return
}
```

+ Obtain list of references we need (that doesn't downloaded yet)

```
list, err := c.Want()
if err != nil {
	// encoding/unregistered schema/invalid tag
	return
}

// if the list is empty then (one of):
//  - we haven't got a root
//  - we already have all objects of the tree
```

+ Inspect the tree

```
// we need a function for inspecting

// s      - schema pointer
// fields - fields of the object (field name -> field value)
// deep   - how far from the root (for root it is 0)
// err    - missing error (ErrMissingRoot or MissingError)

func inspect(s *skyobject.Schema, fields map[string]interface{},
	deep int, err error) error {

	// stuff

	// use ErrStopInspection to stop inspecting

	return nil

}

if err := c.Inspect(inspect); err != nil {
	// encoding/unregistered schema/invalid tag
	//
	// this errors are never passed to InspectFunc;
	// if InspectFunc returns any error (except
	// ErrStopInspection) the Inspect method returns
	// this error
}

```
