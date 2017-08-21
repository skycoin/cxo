CXDS
====

[![GoDoc](https://godoc.org/github.com/skycoin/cxo/data/cxds?status.svg)](https://godoc.org/github.com/skycoin/cxo/data/cxds)

CXDS is CX data store. The package is stub and will be repalced with
client library for CX data server. API of the package is close to goal
and can be used wihtout caring about future.

The CXDS based on [boltdb](github.com/boltdb/bolt) and
[buntdb](github.com/tidwall/buntdb). E.g. it can be used as long time store
on HDD (SSD or vinil disks). And there is in-memeory store for test.

---
