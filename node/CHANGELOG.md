changes
=======

### Done

#### ...

- [x] re-refactored (ha-ha)

#### ...

The skyobejct package has been refactored. And there are chagnes related to it
and little improvements

- [x] improve GC (#97)
- [x] add `OnDropRoot` callback
- [x] move `RootWalker` to skyobject package
- [x] rid out of RPC logs `method * has wrong number of ins: *`

#### 10:00 28 June 2017

- [x] add ability to remove non-full root objects that will not be filled

All non-full root objects will be removed from database during Close

#### 22:00 19 June 2017 UTC

- add linked list example

#### 16:15 17 June 2017 UTC

- [x] move `(*Node).Start()` to `NewNodeReg`

The `(*Node).Start` is deprecated and has been replaced by a stub.

Also

- there are changes in skyobject package
- GC disabled by default


And there is breaking changes:

- `NodeConfig` renamed to `Config` following golang linter recommendations

####  1:00 16 June 2017 UTC

- [x] implement `Tree` RPC method

Also

+ added `Roots` RPC method
+ changed arguments for `Subscribe` and `Unsubscribe` methods of `RPCClient`

#### 20:40 15 June 2017 UTC

- [x] add examples
- [x] add benchmarks

Also, there are some documentation improvements.

Benchmarks output is:

```
$ go test -bench . -benchtime 10s
BenchmarkNode_srcDst/empty_roots_memory-4       10000    2390448 ns/op  100237 B/op  1690 allocs/op
BenchmarkNode_srcDst/empty_roots_boltdb-4         100  113579500 ns/op  135854 B/op  1911 allocs/op
BenchmarkNode_passThrough/empty_roots_memory-4   5000    3082852 ns/op  128090 B/op  2159 allocs/op
BenchmarkNode_passThrough/empty_roots_boltdb-4    100  173033312 ns/op  184738 B/op  2549 allocs/op
PASS
ok      github.com/logrusorgru/cxo/node 73.241s
```

Machine is: 

```
Intel Core i5-6200U
8GB DDR4
Linux 4.10.0-22-generic #24-Ubuntu SMP Mon May 22 17:43:20 UTC 2017 x86_64
```


#### 18:20 14 June 2017 UTC

- [x] add OnDial filter to close connection that closed by other side

Now, a Node doesn't redial if a remote peer closes connection (by
`io.EOF` and `connection reset by peer` errors). Introduced `OnDialFilter`
function that is in `NodeConfig` returned by `NewNodeConfig`.
A developer free to set it to nil or use own filter. For example:

**own filter**

```
conf := node.NewNodeConfig()
conf.Config.OnDial = func(c *gnet.Conn, err error) error {
	// my filter
	return node.OnDialFilter(c, err) // + default filter
}
```

**set to nil**

```
conf := node.NewNodeConfig()
conf.Config.OnDial = nil // that is obvious
```

If you create NodeConfig not using NewNodeConfig, then you need to set
this field manually. Otherwise the Node created using this NodeConfig will
redials even if remote peer resets connection.
