CX Object System
===

Specifications: https://pad.riseup.net/p/cxo

## SkyObjects Quickstart
`import "github.com/skycoin/cxo/skyobject"`

### Create a SkyObjects Container
To create a container, use the `SkyObjects(ds data.IDataSource) *skyObjects` function. Only a database source needs to be specified.

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
### Register Schema
Before storing objects into the SkyObjects container, we need to register the object's Schema. Schema tells the container what 'type' of object it is, and what fields the object will have.

Later, we can get list all Schema types from the container, and retrieve all objects of a specific Schema.

Schemas are registered into a SkyObjects container using the `RegisterSchema(types ...interface{})` member function.

```go
type Board struct {
	Name string
}

var container = skyobject.SkyObjects(data.NewDB())
container.RegisterSchema(Board{}) // Register Schema.
```
### Store Object
Objects can be saved into the container using the `Save(hr IHashObject) Href` member function of the container. The input of the function needs to be of type IHashObject. 
