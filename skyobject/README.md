skyobject
=========

[![GoDoc](https://godoc.org/github.com/skycoin/cxo/skyobject?status.svg)](https://godoc.org/github.com/skycoin/cxo/skyobject)

```go
import (
    "github.com/skycoin/cxo/data"
    "github.com/skycoin/cxo/skyobject"
)
```

Example `github.com/skycoin/cxo/quickStart/skyobject`

## Usage

Create container to manage root objects

```go
db := data.NewDB()
cnt := skyobject.NewContainer(db)
```

Create root object using public key

```go
root := cnt.NewRoot(publicKey)
```

Register all schemas to use them

```go
cnt.Register("User", User{})
```

Register set of objects with recursive dependencise

```go

type Group struct {
	Name  string
	Owner skyobject.Reference  `skyobject:"schema=User"` // a'la *User
	Users skyobject.References `skyobject:"schema=User"` // a'la []*User
}

type User struct {
	Name string
	MemberOf skyobject.Reference `skyobject:"schema=Group"` // a'la *Group
}

// order doesn't matter
cnt.Register(
	"User", User{},
	"Group", Group{},
)
```

References and tags

+ `skyobject.Reference` with schema tag - single reference
+ `skyobject.References` with schema tag - array of references
+ `skyobject.Dynamic` - dynamic reference (schema+object)

```go
type User struct {
    Name   string
    Age    uint32
    Secret []byte `enc:"-"` // skip the field
}

type Group struct {
    Leader  skyobject.Reference  `skyobject:"schema=User"`
    Members skyobjecr.References `skyobject:"schema=User"`
    Curator skyobject.Dynamic
}
```

Save an object.

```go
ref := cnt.Save(User{"Alice", 21, nil})
```

Save an array of objects

```go
refs := cnt.SaveArray(
    User{"Alice", 21, nil},
    User{"Bob", 89, nil},
    User{"Eva", 23, nil},
)
```

Create Dynamic object

```go
type Man struct {
    Name   string
    Gender bool
    Age    uint32
}

man := cnt.Dynamic(Man{"Tom Cobley", true, 98})

// and save the Dynamic object if need
ref := cnt.Save(man)
```

Add an object to a Root

```go
root.Inject(User{"Alice", 21, nil})
```

The `Inject` is short hand for

+ create Dynamic by an object (saving the object)
+ save the Dynamic
+ add reference of the saved Dynamic to the set of references of the root

Add a prepared, saved dynamic object to a Root by hash

```go
hash := cnt.Save(cnt.Dynamic(Man{
    Name:   "Tom Cobley",
    Gender: true,
    Age:    98,
}))

root.InjectHash(hash)
```

The `NewRoot` method creates detached root object. To attach the object to
the Container use AddRoot method that requires secret key to sign the root

```go
root := cnt.NewRoot(publicKey)
cnt.Register("User", User{})
root.Inject(Group{
    Leader: cnt.Save(User{"Alice", 21, nil}),
    Members: cnt.SaveArray(
        User{"Bob", 44, nil},
        User{"Eva", 32, nil},
    ),
    Curator: cnt.Dynamic(Man{
        "Tom Cobley",
        true,
        98,
    }),
})
cnt.AddRoot(root, secretKey)
```

To modify existing root get it first

```go
root := cnt.Root(publicKey)
if root == nil { // not found
    root = cnt.NewRoot(publicKey) // create new
}
root.Inject(User{"Jack Simple", 34, nil})
root.Touch() // update timestamp and increment seq number
cnt.AddRoot(root, secretKey)
```

If AddRoot receive a root older than existing one (with same public key),
then it reject the argument

```go
if !cnt.AddRoot(root, secretKey) {
    // older or the same (was not replaced)
}
```

## Travers

Get values of direct references of a root

```go
vals, err := root.Values()
if err != nil {
    // handle error
}
```

##### Kind of value

To explore value use its `Kind` method

+ `reflect.Invalid` - an empty reference (a'la nil)
+ `reflect.Bool`,
  `reflect.Int8`, `reflect.Uint8`,
  `reflect.Int16`, `reflect.Uint16`,
  `reflect.Int32`, `reflect.Uint32`,
  `reflect.Float32`, `reflect.Int64`,
  `reflect.Uint64`, `reflect.Float64`,
  `reflect.String` - scalar values
+ `reflect.Slice` - a slice
+ `reflect.Array` - an array
+ `reflect.Struct` - a structure
+ `reflect.Ptr` - a reference

Use following methods

##### References

+ `Dereference` if value is reference to get value

##### Scalar value

+ `Bool` to get boolean
+ `Int` to get an `int8`, `int16`, `int32` or `int64`
+ `Uint` to get an `uint8`, `uint16`, `uint32` or `uint64`
+ `String` to get `string`
+ `Bytes` to get `[]byte`
+ `Float` to get `float32` or `float64`

##### Structures

+ `Fields` to get names of fields of a struct
+ `FieldByName` to get value of a field by field name
+ `RangeFields` to explore a struct using
  `func(filedName string, val *Value) error`

##### Arrays and slices

+ `Len` to get length of an array or a slice
+ `Index` to get element by index

##### Schema of value

+ `Schema` returns schema of value. The schema can be nil schema

```go
s := val.Schema()
if s.IsNil() {
    // the schema is nil
}
```
---

