cxodbfix
========

The cxodbfix used to change CXO databases for new versions. The CXO uses
two separate database files IdxDB (index) and CXDS (objects). To see
how to fix get this utility using

```
go get -u github.com/skycoin/cxo/cxodbfix
cd $GOPATH/src/github.com/skycoin/cxo/cxodbfix
go build
```


and run it with `-h` flag

```
cxodbfix -h
```
