CXDS
====

[![GoDoc](https://godoc.org/github.com/skycoin/cxo/data/cxds?status.svg)](https://godoc.org/github.com/skycoin/cxo/data/cxds)

CXDS is CX data store. The package is stub and will be replaced with
client library for CX data server. API of the package is close to goal
and can be used without caring about future.

The CXDS based on [boltdb](github.com/boltdb/bolt) and
[buntdb](github.com/tidwall/buntdb). E.g. it can be used as long time store
on HDD (SSD or vinil disks). And there is in-memory store for test.


## Schema

Schema is simple.

```
hash - > object
```

Where the hash is SHA256 hash of the object. And the object is any
object encoded to `[]byte`


## Binary internals

Every object is

```
[4 byte: refs count][encode object]
```

Where `4 byte: refs count` is little-endian encoded `uint32` that represents
references count for this object.


---
