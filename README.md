CX Object System
================

[![Build Status](https://travis-ci.org/skycoin/cxo.svg)](https://travis-ci.org/skycoin/cxo)
[![Coverage Status](https://coveralls.io/repos/skycoin/cxo/badge.svg?branch=master)](https://coveralls.io/r/skycoin/cxo?branch=master)
[![GoReportCard](https://goreportcard.com/badge/skycoin/cxo)](https://goreportcard.com/report/skycoin/cxo)

Specs: https://pad.riseup.net/p/cxo

## Packages

- `cmd/cxod` - CXO daemon (example node)
- `cmd/cxocli` - cli to control the CXO daemon
- `data` - CX obejcts database
- `node` - CXO networking core
- `skyobejct` - CX objects tree

### Version

Use [dep](https://github.com/golang/dep) to use particular commit

## Database

- [data/README.md](https://github.com/logrusorgru/cxo/blob/master/data/README.md)
- Binary data representation: [data/ABI.md](https://github.com/logrusorgru/cxo/blob/master/data/ABI.md)

## Skyobject

- [skyobject/README.md](https://github.com/logrusorgru/cxo/blob/master/skyobject/README.md)
- Binary data representation: [skyobject/ABI.md](https://github.com/logrusorgru/cxo/blob/master/skyobject/ABI.md)


## Node

- [node/README.md](https://github.com/logrusorgru/cxo/blob/master/node/README.md)

## CLI

See [cmd/cxocli/README.md](https://github.com/logrusorgru/cxo/blob/master/cmd/cxocli/README.md)

## Daemon

See [cmd/cxod/README.md](https://github.com/logrusorgru/cxo/blob/master/cmd/cxod/README.md)

---


# How it works


##### There are

```go
type User struct {
	Name   string
	Age    int
	Hidden []byte // for local usage
}

type Group struct {
	Name    string
	Leader  *User
	Members []*User
	Curator interface{}
}

type Man struct {
	Name string
}
```

##### Turn them to

```go

import (
	// [...]

	cxo "github.com/skycoin/cxo/skyobject"
)

type User struct {
	Name   string
	Age    uint32           // fixed size
	Hidden []byte `enc:"-"` // don't send through network
}

type Group struct {
	Name    string
	Leader  cxo.Ref  `skyobject:"schema=User"` // pointer
	Members cxo.Refs `skyobject:"schema=User"` // slice of pointers
	Curator cxo.Dynamic                        // interface{}-like
}

type Man struct {
	Name string
}
```

##### Register them

```go
reg := cxo.NewRegistry(func(r *cxo.Reg) {
	r.Register("pkg.User", User{})
	r.Register("pkg.Group", Group{})
	r.Regsiter("pkg.Man", Man{})
})
```

Feel free to choose names you want

##### Create a `Node`

```go

import (
	// [...]

	"github.com/skycoin/cxo/node"
)


// [...]

conf := node.NewConfig()

// The Regsitry can be nil for nodes that pass
// objects through. But we are going to create and
// use obejcts. Thus we need a Regsitry
conf.Skyobject.Regsitry = reg

// other configs


n, err := node.NewNode(conf)
if err != nil {
	// fatality
}

// So, the Node has been created and started

defer n.Close()

```

##### About `node.Node`

A `Node` used to create (emit), replicate, and receive feeds. A `Node` doesn't
do it autmatically. You need to subscribe to a feed. You need to associate a
connection to another node with the subscription. Both, this-side and remote
nodes should be subscribed to the feed to start exchanging it. If
`(node.Config).PublicServer` set to true, then you can obtain list of feeds
from a remote node. Otherwise, it can be done uisng RPC (e.g. administrative
access required). A `Node` is thread-safe but you must not block callbacks,
because they are performing inside message-handling goroutine of a connection

##### About `skyobject.Root`

A `Root` object is root of objects tree. It contains slice of
`skyobject.Dynamic`. The `Dynamic` references are main branches of the obejcts
tree. Every of them has (or has not) own branches, and so on. A `Root`
signed by owner. A `Root` belongs to its feed. A `Root` has sequence number
(seq) and timestamp. Every `Root` has reference to its `Regsitry`. Thus,
obejcts in the tree can be created (and obtained) only using the `Registry`.
Also, a `Root` contains hash of previous `Root` (that is empty if the seq is 0).
The Root is not thread-safe. Only the Refs field requires the safety


```go

type Root struct {
	Refs []Dynamic // main branches

	Reg RegistryRef   // registry
	Pub cipher.PubKey // feed

	Seq  uint64 // seq number
	Time int64  // timestamp (unix nano)

	Sig cipher.Sig `enc:"-"` // signature (not part of the Root)

	Hash cipher.SHA256 `enc:"-"` // hash (not part of the Root)
	Prev cipher.SHA256 // hash of previous root
}

```

##### Create, Update and Publish a `Root`

```go

import (
	// [...]

	"github.com/skycoin/skycoin/src/cipher"
)

// [...]

// pk - is kind of cipher.PubKey, it is our feed
// sk - is kind of cipher.SecKey, it is owner of the feed
// n  - is our *node.Node instance

// note: to generate random pk, sk pair use
pk, sk := cipher.GenerateKeyPair()


cnt := n.Container() // the call is cheap, the Container is thread-safe


// create new Root object of the pk feed
// to fill the Root we are using *skyobject.Pack
pack, err := cnt.NewRoot(pk, sk, 0, cnt.CoreRegistry().Types())
if err != nil {
	// something wrong
}
```

So, the `Root` has been created. It doesn't stored in DB or somewhere else.
If you want to update the Root often, then you can keep the `Pack` in its
goroutine performing changes if need. For example

```go
// in this example the Event is an event, that requires updates in our Root
func startHandlingSomePack(events <-chan Event, pack *skyobject.Pack,
	n *node.Node, wg *sync.WaitGroup) {

	go handleSomePack(n, pack, events, n.Quiting(), wg)
}

func handleSomePack(n *node.Node, pack *cxo.Pack, events <-chan Event,
	quit <-chan struct{}, wg *sync.WaitGroup) {

	defer wg.Done()

	for {
		select {
		case evt := <-events:
			// perform changes using the evt (Event)
			performCahnges(pack, evt)
			perfomAllChangesPossible(quit, events, pack, n)
		case <-quit:
			return
		}
	}
}

func performAllCahgnesPossible(quit <-chan struct{}, events <-chan Event,
	pack *skyobject.Pack, n *node.Node) {

	tm := time.NewTimer(MinPublishInterval)
	defer tm.Stop()

	// perform all changes possible and publish
	for {
		select {
		case <-quit:
			return
		case evt := <-events:
			performCahnges(pack, evt)
		case <-tm.C:
			// to many changes performed but not published yet
			publish(n, pack)
			tm.Reset(MinPubishInterval) // start the timer again
		default:
			// no more events, no more changes
			publish(n, pack)
			return
		}
	}
}

func publish(n *node.Node, pack *skyobject.Pack) {
	if err := pack.Save(); err != nil {
		// fatality: handle the err
	}
	n.Publish(pack.Root())
}

func performCahgnes(pack *skyobject.Pack, evt Event) {
	// stuff
}

```

Where

+ `pack.Save()` saves changes, updating seq number, hash to previous Root
  timestamp and changed branches. It also, signs the Root using the `sk`. After
  this call (if it's successful) you can get `pack.Root()` and all fields or
  the Root will be actual.
+ `n.Publish()` sends given root to subscribers

The `Pack` is not thread-safe.

Actually, a Node can't be a subscriber. Nodes exchange a feed or not
exchange. A Node, which got new Root obejct, sends this root to peers
that interested in the feed of the Root. For more information see
godoc documentation of the node package

Also, there is `(node.Node).Quiting()` method that returns `<-chan struct{}`.
The chan is closed after `(node.Node).Close()` call

##### Track Changes

// TODO (kostyarin): example

##### Pack/Unpack flags

- `EntireTree` - unpack entire tree. This way, a whole will be unpacked inside
  `Unpack` call. The unpaking stops on first error. E.g. the tree should be
  in database, otherwise the flag doesn't make sence
- `EntireMerkleTree` - unpack all branches of Refs. If the flag is not set,
  then branhces of Refs will be unacked by needs
- `HashTableIndex` - use hash-table index for Refs to speed up lookup by hash
- `ViewOnly` - don't allow modifications
- `AutoTrackChanges` - track changes automatically. This flag is experimental

##### Receive and Explore a Root

TODO (kostyarin): explain detailed (after "track changes")

##### References

TODO (kostyarin): explain detailed

##### Unpack Flags

TODO (kostyarin): explain detailed

##### Double-Linked List Example

TODO (kostyarin): do it
