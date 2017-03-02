CX Object System
===
Specs: https://pad.riseup.net/p/cxo

## SkyObjects Quickstart

`import "github.com/skycoin/cxo/skyobject"`

### Creating a SkyObjects Container

To create a container, use the `SkyObjects` function. Only a database source needs to be specified. Here is an example:

```go
import (
    "github.com/skycoin/cxo/skyobject"
    "github.com/skycoin/cxo/data"
)

func main() {    
    db := data.NewDB() // Create database.
    container := skyobject.SkyObjects(db) // Create container with database.
    container.Inspect() // Example container use.
}
```

### Registering Schemas

Before storing objects into the SkyObjects container, we need to register the object's Schema. Schema tells the container what 'type' of object it is, and what fields the object will have.

Schemas are registered into a SkyObjects container using the `RegisterSchema` member function:

```go
type Board struct {
	Name string
}

container.RegisterSchema(Board{}) // Register Schema.
```

We can list all Schema types from the container using the `GetSchemas` member function which returns `[]Schema`:

```go
schemas := container.GetSchemas() // Get array of all schemas from container.

fmt.Println("Schemas:") // Print results.
for _, schema := range schemas {
    fmt.Println(" -", schema.Name)
}
```

### Storing Objects

Objects can be stored into the container using the `Save` member function. It returns a `Href` type of the object stored.

Here is an example of storing an object into the container, and obtaining the object's key:

```go
board1 := Board{"board1"} // Create a new Board.
boardRef := skyobject.NewObject(board1) // Make Board reference object.
container.Save(&boardRef) // Store Board in container.

board1Key := boardRef.Ref // Obtain the board's key.
```

An array of objects can be stored in the container using the following method:

```go
boards := []Board{{"board1"}, {"board1"}, {"board1"}} // Create a Board array.
boardsRef := skyobject.NewArray(boards) // Make array reference object.
container.Save(&boardRef) // Store boards in container.
```

Note that the objects in the array are stored separately, and the array is infact an array of references.

### Storing Objects That Reference Other Objects

Objects can be stored to refer to one another.

Here is an example where the Parent object references the Child object:

```go
type Parent struct {
    Name  string
    Child skyobject.HashObject // Reference to Child object.
}

type Child struct {
    Name string
}

// Create and store a Child.
child := Child{Name: "Ben"}
childRef := skyobject.NewObject(child)
container.Save(&childRef)

// Create and store a Parent that references a Child.
parent := Parent{Name: "Bob", Child: childRef}
parentRef := skyobject.NewObject(parent)
container.Save(&parentRef)
```

Here is an example where the Parent object references multiple Child objects:

```go
type Parent struct {
    Name     string
    Children skyobject.HashArray // Reference to multiple children.
}

type Child struct {
    Name string
}

// Create and store multiple children.
children := []Child{{"Ben"}, {"James"}, {"Ryan"}}
childrenRef := skyobject.NewArray(children)
container.Save(&childrenRef)

// Create and store a Parent that references multiple children.
parent := Parent{Name: "Bob", Children: childrenRef}
parentRef := skyobject.NewObject(parent)
container.Save(&parentRef)
```

### Retrieving Objects

The simplest way to retrieve sky objects from the container is by using the `GetObjRef` member function. There is only one input, which is the sky object's key (of type `cipher.SHA256` from `"github.com/skycoin/skycoin/src/cipher"`).

The output will be of type `ObjRef`, which we can use to obtain useful data, and deserialize the skyObject into our original object type.

Here is an example of retrieving an object with a given key:

```go
// Get the parent's reference object with 'cipher.SHA256' key.
parentRef := container.GetObjRef(parentKey) 

// Obtain original object from container.
var parent Parent
parentRef.Deserialize(&Parent)

fmt.Println("I've obtained a parent:", parent.Name)

```

We can also use `ObjRef` to obtain children objects:

```go
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
```

Here is an example of retrieving all objects of a specific Schema:

```go
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

```