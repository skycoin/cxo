Pass Through
============


### Briefly

- `ca` - generates A-feed, and subscribed to B-feed
- `cb` - generates B-feed, and subscribed to A-feed
- `server` - public server

### Explain

The `server` uploads public keys of A-, and B-feeds to [skycoin messanger
server](https://github.com/skycoin/net/tree/master/skycoin-messenger/server)
(later: SMS).

The SMS conects `ca` to `cb` to make then exchangind A-, and B-feeds

### Run

#### Prepare

```
go get -d github.com/skycoin/net/skycoin-messenger/server
```

#### Start

Four terminals required.

Launch the skycoin messanger server using "[::1]:8008" address in first terminal
```
go run $GOPATH/src/github.com/skycoin/net/skycoin-messenger/server/main.go \
    -address [::1]:8008
```

Launch the `server` first
```
go run $GOPATH/src/github.com/skycoin/cxo/intro/exchange/server/server.go
```

In another terminal launch the `ca` node
```
go run $GOPATH/src/github.com/skycoin/cxo/intro/exchange/ca/ca.go
```

And finally, in another terminal lauch the `cb` node
```
go run $GOPATH/src/github.com/skycoin/cxo/intro/exchange/cb/cb.go
```

Feed represents
```
Root -> Content
        |
        +--> Thread --> []Vote
        |
        +--> Post   --> []Vote
```
And every second the ca and cb appends some votes to own feed.

---
