Exchange
========


### Briefly

- `ca` - generates A-feed, and subscribed to B-feed
- `cb` - generates B-feed, and subscribed to A-feed

```
A-feed: 022241cd1857ec73b7741ccb5d5494743e1ef7075ec71d6eca59979fd6be58758c
B-feed: 030c398e49cb77e83baa3110f99a105a33e5caf4e63c4ff55dbbaabbc98159e792
```


Both, `ca` and `cb` are listening, public and connected to discovery server.
The discovery server connects the nodes between (one connection, actually).

![image](./discovery_exchange.png)

### Discovery

[Skycoin messenger server](https://github.com/skycoin/net/tree/master/skycoin-messenger/server)

### Run

#### Prepare

Get fresh version of the discovery server

```
go get -d -u github.com/skycoin/net/skycoin-messenger/server
```

#### Start

Three terminals required.

Launch the skycoin messenger server using "[::1]:8008" address in first terminal
```
go run $GOPATH/src/github.com/skycoin/net/skycoin-messenger/server/main.go \
    -address [::1]:8008
```

In another terminal launch the `ca` node
```
go run $GOPATH/src/github.com/skycoin/cxo/intro/discovery/exchange/ca/ca.go
```

And finally, in another terminal launch the `cb` node
```
go run $GOPATH/src/github.com/skycoin/cxo/intro/discovery/exchange/cb/cb.go
```


The `ca` and the `cb` nodes show received Root objects

---
