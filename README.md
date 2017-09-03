CX Object System
================

[![Build Status](https://travis-ci.org/skycoin/cxo.svg)](https://travis-ci.org/skycoin/cxo)
[![Coverage Status](https://coveralls.io/repos/skycoin/cxo/badge.svg?branch=master)](https://coveralls.io/r/skycoin/cxo?branch=master)
[![GoReportCard](https://goreportcard.com/badge/skycoin/cxo)](https://goreportcard.com/report/skycoin/cxo)
[![Telegram group link](telegram-group.svg)](https://t.me/joinchat/B_ax-A6oCR9eQuAPiJtvaw)

![logo](cxo_logo.png)

The CXO is objects system, goal of which is sharing any objects. The CXO
is low level and designed to build application on top of it

### How it wroks

So, there is an owner who want to publicate something. This man creates feed,
and adds a Root (R1) obejct to this feed. The Root refers to another obejcts.
It's tree of any objects. The tree is immutable and object are immutable too.
But the owner can update the tree publishing new version of the Root (R2).
This root is based on previous version. This R2 is replacement for the R1.
Thus, if the owner publishes some `Root` and updates it R1->R2->R3->R4. Any
node that receive this feed can ignore R1,R2 and R3, because they are outdated
since R4 borns

For example
```go
type User struct {
    Name string
    Age  uint32
    Bio  skyobject.Ref `schema:"pkg.Bio"`
}

type Bio struct {
    Height uint32
    Weight uint32
}
```

It's like
```go
type User struct {
   Name string
   Age  int
   Bio  *Bio
}
```

The Ref is SHA256 hash of encoded value. Thus, we are using uint32 instead
of int. Because the int can be int32 or int64 and not acceptable for
encoding and decoding. E.g. the `User` will point to some encoded `Bio`.

And if this `Bio` has been changed, then the `User` have to be changed too.
And the owner, who want to share this information, creates feed. Creates Root
obejct. Attaches actual instance of `User` to the `Root`. Signs the `Root` and
publishes it. And when the owner want to update this information, he publishes
new version of this `Root`.

Thus, the `Root` can point to many objects. But, updating one obejct, we
updates only one branch of the objects tree. All other obejcts already
replicated by peers.

Trusting. The owner has secret key, and feed is public key based
on the secret key. And every Root is signed by the owner. Thus, a node,
receiving a Root, can verify singature and be sure that this Root comes from
that owner. This way if we know a feed (public key) then we can trust all
received data from its feed.

### Version

Use [dep](https://github.com/golang/dep) to use particular commit.

To get mainline use
```
go get -u -t github.com/skycoin/cxo/...
```
Test them all
```
go test -cover -race github.com/skycoin/cxo/...
```

### Get started. How it works? Documentation

See [CXO wiki](../../wiki) to get this information

### Help and contacts

- [telegram group](https://t.me/joinchat/B_ax-A6oCR9eQuAPiJtvaw)

---





<!--

## Database

- [data/README.md](data/README.md)
- Binary data representation: [data/ABI.md](data/ABI.md)

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


-->