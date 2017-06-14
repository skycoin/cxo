changes
=======

#### TODO

- [ ] add examples
- [ ] improve documentation
- [ ] add RPC methods to see conn->feeds and feed->conns
- [ ] implement Tree RPC method
- [ ] add OnDropRoot callback
- [ ] add ability to remove non-full root objects that will not be filled
- [ ] add more tests
- [ ] add benchmarks

#### Done

##### 18:20 14 June 2017 UTC

- [x] add OnDial filter to close connection that closed by other side

Now, a Node doesn't redial if a remote peer closes conenction (by
`io.EOF` and `connection reset by peer` errors). Introducted `OnDialFilter`
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
redials even if remote peer resets conenction.