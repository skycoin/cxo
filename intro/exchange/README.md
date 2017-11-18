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


### Explain

All node uses [skycoin messenger
 server](https://github.com/skycoin/net/tree/master/skycoin-messenger/server),
this way, `ca` and `cb` connect to `server`. And since, the `server` shares
A-, and B-feed, nodes `ca` and `cb`  will receive opposite feed through the
`server`.

### Run

#### Prepare

```
go get -d -u github.com/skycoin/net/skycoin-messenger/server
```

#### Start

Four terminals required.

Launch the skycoin messenger server using "[::1]:8008" address in first terminal
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

And finally, in another terminal launch the `cb` node
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
diff --git a/ca/ca.go b/cb/cb.go
index 4103868..a7ce535 100644
--- a/ca/ca.go
+++ b/cb/cb.go
@@ -21,8 +21,8 @@ import (
 
 // defaults
 const (
-       Host        string = "[::1]:8001" // default host address of the node
-       RPC         string = "[::1]:7001" // default RPC address
+       Host        string = "[::1]:8002" // default host address of the node
+       RPC         string = "[::1]:7002" // default RPC address
        RemoteClose bool   = false        // don't allow closing by RPC by default
 
        Discovery string = "[::1]:8008" // discovery server
@@ -69,7 +69,7 @@ func main() {
        c.InMemoryDB = true // use DB in memeory
 
        // node logger
-       c.Log.Prefix = "[ca] "
+       c.Log.Prefix = "[cb] "
 
        // suppress gnet logger
        c.Config.Logger = log.NewLogger(log.Config{Output: ioutil.Discard})
@@ -77,7 +77,7 @@ func main() {
        c.Skyobject.Registry = reg // <-- registry
 
        // skyobject logger
-       c.Skyobject.Log.Prefix = "[ca cxo]"
+       c.Skyobject.Log.Prefix = "[cb cxo]"
 
        // show full root objects
        c.OnRootFilled = func(c *node.Conn, r *cxo.Root) {
@@ -92,10 +92,10 @@ func main() {
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
@@ -124,7 +124,7 @@ func main() {
        stop := make(chan struct{})
 
        wg.Add(1)
-       go fictiveVotes(s, &wg, apk, ask, stop)
+       go fictiveVotes(s, &wg, bpk, bsk, stop)
 
        defer wg.Wait()   // wait
        defer close(stop) // stop fictiveVotes call
```
