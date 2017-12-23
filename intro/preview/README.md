Preview
=======

There are `seed` node, that has two feeds, and `peer` node that has not feeds
and connected to the `seed`.


#### Start

Two terminals required.

Launch the `seed` node first
```
go run $GOPATH/src/github.com/skycoin/cxo/intro/preview/seed/seed.go
```

Wait a little time to let the `seed` start its TCP listener. Waiting more, you
can see how node skips old Root objects sending last.

And finally, in another terminal launch the `peer` node
```
go run $GOPATH/src/github.com/skycoin/cxo/intro/preview/peer/peer.go
```

The `peer` node has some interactive commands to preview feeds from the `seed`.
