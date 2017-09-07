Exchange
========


### Briefly

- `ca` - generates A-feed, and subscribed to B-feed
- `cb` - generates B-feed, and subscribed to A-feed
- `server` - public server that subscribed to A- and B-feed

### Explain

TODO [skycoin messanger
server](https://github.com/skycoin/net/tree/master/skycoin-messenger/server)

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


### Difference between the `ca` and `cb`

```diff
$ git diff --no-index ../../intro/exchange/ca/ca.go ../../intro/exchange/cb/cb.go
diff --git a/../../intro/exchange/ca/ca.go b/../../intro/exchange/cb/cb.go
index 342819d..8c434f0 100644
--- a/../../intro/exchange/ca/ca.go
+++ b/../../intro/exchange/cb/cb.go
@@ -21,11 +21,11 @@ import (
 
 // defaults
 const (
-       Host        string = "[::1]:8001" // default host address of the server
-       RPC         string = "[::1]:7001" // default RPC address
+       Host        string = "[::1]:8002" // default host address of the server
+       RPC         string = "[::1]:7002" // default RPC address
        RemoteClose bool   = false        // don't allow closing by RPC by default
 
-       Discovery string = "[::1]:8008" // discovery server
+       Discovery string = "[::1]:8008"
 )
 
 func waitInterrupt(quit <-chan struct{}) {
@@ -69,7 +69,7 @@ func main() {
        c.InMemoryDB = true // use DB in memeory
 
        // node logger
-       c.Log.Prefix = "[ca] "
+       c.Log.Prefix = "[cb] "
        c.Log.Debug = true
        c.Log.Pins = log.All &^ (node.DiscoveryPin | node.HandlePin | node.FillPin |
                node.ConnPin)
@@ -80,7 +80,7 @@ func main() {
        c.Skyobject.Registry = reg // <-- registry
 
        // skyobject logger
-       c.Skyobject.Log.Prefix = "[ca cxo]"
+       c.Skyobject.Log.Prefix = "[cb cxo]"
        c.Skyobject.Log.Debug = true
        c.Skyobject.Log.Pins = log.All &^ (cxo.VerbosePin | cxo.PackSavePin |
                cxo.FillVerbosePin)
@@ -98,10 +98,10 @@ func main() {
        c.FromFlags()
        flag.Parse()
 
-       // apk and bk is A-feed and B-feed, the ask is
+       // apk and bk is A-feed and B-feed, the bsk is
        // secret key required for creating
-       apk, ask := cipher.GenerateDeterministicKeyPair([]byte("A"))
-       bpk, _ := cipher.GenerateDeterministicKeyPair([]byte("B"))
+       apk, _ := cipher.GenerateDeterministicKeyPair([]byte("A"))
+       bpk, bsk := cipher.GenerateDeterministicKeyPair([]byte("B"))
 
        var s *node.Node
        var err error
@@ -130,7 +130,7 @@ func main() {
        stop := make(chan struct{})
 
        wg.Add(1)
-       go fictiveVotes(s, &wg, apk, ask, stop)
+       go fictiveVotes(s, &wg, bpk, bsk, stop)
 
        defer wg.Wait()   // wait
        defer close(stop) // stop fictiveVotes call
```