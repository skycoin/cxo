CX Object System
===

Specifications: https://pad.riseup.net/p/cxo

## SkyObjects Quickstart
`import "github.com/skycoin/cxo/skyobject"`

#### Creating a SkyObjects Container
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
#### Registering Schemas
Before storing objects into the SkyObjects container, we need to register the object's Schema. Schema tells the container what 'type' of object it is, and what fields the object will have.

Schemas are registered into a SkyObjects container using the `RegisterSchema` member function:

```go
type Board struct {
	Name string
}

container.RegisterSchema(Board{}) // Register Schema.
```
Later, we can list all Schema types from the container, and retrieve all objects of a specified Schema. (LINK HERE)

#### Storing Objects
Objects can be stored into the container using the `Save` member function of the container:
```go
board1 := Board{"board1"} // Create a new Board.
boardRef := skyobject.NewObject(board1) // Make Board reference object.
container.Save(&boardRef) // Store Board in container.
```
An array of objects can be stored in the container using the following method:
```go
boards := []Board{{"board1"}, {"board1"}, {"board1"}} // Create a Board array.
boardsRef := skyobject.NewArray(boards) // Make array reference object.
container.Save(&boardRef) // Store boards in container.
```
Note that the objects are stored separately, and the array is infact an array of references.

#### Storing Objects That Reference Other Objects
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
