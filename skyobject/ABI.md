Binary data format
==================

## Data

### Int, Uint

All signed and unsigned integers encoded using little-endian order to
bytes slice with appropriate size

- int8, uint8 - one byte
- int16, uint16 - 2 byte
- int32, uint32 - 4 byte
- int64, uint64 - 8 byte

That is obvious.

See [LittleEndian](http://godoc.org/encoding/binary#LittleEndian) and
[ByteOrder](http://godoc.org/encoding/binary#ByteOrder)

### Float32, Flaot64

For float point number used IEEE 754 binary representation.
E.g. floatXX -> IEEE -> uintXX -> little-endian -> bytes

- float32 - 4 byte
- float64 - 8 byte

See [`math.Float32bits`](http://godoc.org/math#Float32bits),
[`math.Float32frombits`](http://godoc.org/math#Float32frombits) and
[`math.Float64bits`](http://godoc.org/math#Float64bits),
[`math.Float64frombits`](http://godoc.org/math#Float64frombits) methods

### String

Strings are length-prefixed, where length is represented as encoded `uint32`

```
[4 byte length][string body or nothing if the length is zero]
```

### Arrays

Arrays is just encoded elements one by one

### Slices

Slices are length-prefixed like strings

```
[4 byte length][elements one by one]
```

### Structures

Structures is encoded elements from first field to last

For example

```go
type Age uint32

type User strut {
	Age  Age
	Name string
}
```

should be encoded to

```
[4 byte of the Age]
[4 byte of the Name length]
[the name in person]
```

That's all about data

## References

Since we can't use pointers, we are using SHA256 hash of encoded value.
This can be applied only to structures. (E.g. only "pointers" to structures).
There are 3 types of references:

- `Ref` like a pointer (`*User`)
- `Refs` like slice of pointers (`[]*User`)
- `Dynamic` like `interface{}`

Because we are using SHA256 hash, we can't determine type of element of
a reference (except the `Dynamic`). Thus, we need to  provide this infomation
inside schema (see below).

The `Ref` is SHA256 hash.
The `Dynamic` is struct that contains SHA256-hash that points to schema, and
SHA256-hash that points to object. The schema-reference is first

```go
type Dynamic struct {
	SchemaRef cipher.SHA256
	Object    cipher.SHA256
}
```

Thus the `Dynamic` encoded as `[encoded schema hash][encoded object hash]`


The `Refs` represented as Merkle-tree with degree. Take a look the image
![cxo refs representation](https://drive.google.com/open?id=0BxQawgmXLZarNUJPekJKRmZLMUk)

Thus we need:

1. SHA256 that points to root of Refs
2. encoded Refs root
3. encoded refs nodes

What the fields means?

- root
```go
type refsRoot struct {
	Depth  uint32
	Degree uint32
	Length uint32
	Nested []SHA256
}
```

- node
```go
type refsNode struct {
	Length uint32
	Nested []SHA256
}
```

The `Depth` is `depth - 1` of the tree, starting from root. E.g. inversed height
of the refs-merkle-tree. If the `Depth` is 0, then this root or node contains
references to elements (not nodes of the Refs). For exampls if the root is

```go
{Depth: 0, Degree: 16, Length: 2, Nested []SHA256{xxx, yyy}}
```
then the `xxx` points to first element, and the `yyy` points to second element.

If the `Depth` is 1 then hashes of the `Nested` points to node, depth of
which is the `Depth - 1`. We can walk from root decrementing depth every node
to find which contains leafs. This way all branches has the same length.

The `Degree` is max length of the `Nested`. Every root and every node can
have maximum `Degree` childrens. It can contain no childrens or less then the
`Degree`. If we know `Depth` and if we know `Degree`, we can calculate how
many elements the Refs can fit inside.

Since the `Depth` is `depth - 1` (0,1,2 not 1,2,3). Max items is
```
Degree^ Depth+1
```

But we are treating all zero-elements as deleted. To speed up deleting.
And if a hash of a `Nested` is blank, then this element of nested node
is not created or deleted. We never use roots like

```
{Degree: 16, Depth: 0, Length: 0, Nested: nil}
```
We are using blank Refs.Hash instead. One more time: if Refs doesn't contains
anything then we are using blank SHA256 hash instead of hash of encoded
`refsRoot` that contains nothing.

The smae for `refsNode`. If a node empty, then we aren't using encoded
hash of some `{Length: 0, Nested: nil}` node. We are using blank hash.

For example
```
type User struct {
	Name    string
	Friends skyobject.Refs `skyobject:"schema=User"`
}
```

If the `Friends` field is blank SHA256, then this user has no friends. And
if Degree is 3 (for example, actually we are using 16 by default), and the user
has 4 friends then Refs-tree looks like
```
(refs-root)
  depth: 1
  degree: 3
  length: 4
  nested
   |
   +--- (refs-node)
   |     length: 3
   |     nested --> {3 users}
   |
   +--- (refs-node)
   |     length: 1
   |     nested --> {1 user}
   |
   +-- blank hash, because 3rd refs-node is empty

```


The `Length` is total length of subtree. Thus root contains actual length.
And the `Length` of every subtree contains total length of all its subtrees
etc. We have to keep the length because of zero-elements. To find an element
by index we need to know how many elements every subtree fit inside.

#### Schema of a Reference

As mentioned above, we can't determine type of object using SHA256-hash.
Thus, we are using schemas. We have to register every type to use. We are
using struct-tags to define the type. For example:

```
type User struct {
	Name string

	Friends    skyobject.Refs `skyobject:"schema=User"`
	Haters     skyobject.Refs `skyobject:"schema=User"`

	BestFriend skyobject.Ref `skyobject:"schema=User"`
}
```

It's just example

## Root

Binary representation of a Root

... TODO

## Schema

Binary representation of a Schema.

... TODO
