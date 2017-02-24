CX Object System
===

Spec
https://pad.riseup.net/p/cxo

## SkyObjects Quickstart

#### Import the Library
`import "github.com/skycoin/cxo/skyobject"`

That's it.

#### Create a SkyObjects Container
To create a container, use the `SkyObjects(ds data.IDataSource) *skyObjects` function.
Only a database source needs to be specified.

Here is an example:

```go
package main

import (
    "github.com/skycoin/cxo/skyobject"
    "github.com/skycoin/cxo/data"
)

func main() {
    // Create database.
    db := data.NewDB()

    // Create container with database.
    container := skyobject.SkyObjects(db)

    container.Inspect() // Example container use.
}
```
