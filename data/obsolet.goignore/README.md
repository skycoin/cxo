data
====

[![GoDoc](https://godoc.org/github.com/skycoin/cxo/data?status.svg)](https://godoc.org/github.com/skycoin/cxo/data)

Data represents CX objects database

#### Structure

To see DB schema, objets representation and similar see
[ABI.md](https://github.com/logrusorgru/cxo/blob/master/data/ABI.md)

#### Development

The are sublime snippets

- [`dbupdate`](https://github.com/logrusorgru/cxo/blob/master/data/cxo-data-update.sublime-snippet)

```
Update(func(tx data.Tu) (_ error) {
	//
})
```


- [`dbview`](https://github.com/logrusorgru/cxo/blob/master/data/cxo-data-view.sublime-snippet)

```
View(func(tx data.Tv) (_ error) {
	//
})
```
