intro
=====

The `intro` folder contains examples.

- [`cleanup`](./cleanup) - about removing old and ownerless objects
- [`discovery/`](./discovery) - about discovery server
- [`preview/`](./preview) - about feed preview
- [`send_receive/`](./send_receive) - about source and destination, and how to
  decode values back to golang-values
- [`skybrief/`](./skybrief) - breifly about skyobject

The `types.go` file implements basic types for these
examples. And the file also implements regsitry of
these types.

In CXO SHA256 hash used instead of pointers and Merkle-tree
instead of slices. And all values should be encoddable (e.g.
`int` -> `uint32` or any other).


And types below

```go

type User struct {
	Name string
	Age  int
}

type Feed struct {
	Head  string
	Info  string
	Posts []*Post
}

type Post struct {
	Head string
	Body string
}

```

should be rewritten like in the `types.go` and registered.
