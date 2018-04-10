Exchange
========


### Briefly

- `ca` - generates A-feed, and subscribed to B-feed
- `cb` - generates B-feed, and subscribed to A-feed
- `server` - public server that subscribed to A- and B-feed

```
A-feed: 022241cd1857ec73b7741ccb5d5494743e1ef7075ec71d6eca59979fd6be58758c
B-feed: 030c398e49cb77e83baa3110f99a105a33e5caf4e63c4ff55dbbaabbc98159e792
```


Both, `ca` and `cb` are not listening and can't connect between. But the
`server` is listening and the `server` subscribed to A-feed and to B-feed
too.

In the example, the `server` is something like proxy or super-seed. The `ca`
receive B-feed through the `server` and `cb` receive A-feed through the same
`server`.

All nodes connected to discovery server and the `server` (not discovery server)
is public, that means the `server` share list of its feeds. The discovery server
request list of feeds all nodes interest, and connect them between. But the 
`ca` and the `cb` are not public, and are not listening. Thus, the discovery
server connects them to the `server`.

![image](./discovery_through.png)

### Discovery

Original discovery server placed at `github.com/skycoin/skywire/cmdx/discovery`,
but for this examples used modified version of the discovery that uses
random seed for every start (and doesn't keep it) and SQlite3 DB in memory.

### Run

Four terminals required.

Launch the discovery server using "[::1]:8008" address in first terminal
```
go run $GOPATH/src/github.com/skycoin/cxo/intro/discovery/discovery \
    -a [::1]:8008
```

Launch the `server` first
```
go run $GOPATH/src/github.com/skycoin/cxo/intro/discovery/through/server/server.go
```

In another terminal launch the `ca` node
```
go run $GOPATH/src/github.com/skycoin/cxo/intro/discovery/through/ca/ca.go
```

And finally, in another terminal launch the `cb` node
```
go run $GOPATH/src/github.com/skycoin/cxo/intro/discovery/through/cb/cb.go
```

The `ca` and the `cb` nodes show received Root objects

---
