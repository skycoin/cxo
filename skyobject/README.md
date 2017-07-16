skyobject
=========

[![GoDoc](https://godoc.org/github.com/skycoin/cxo/skyobject?status.svg)](https://godoc.org/github.com/skycoin/cxo/skyobject)

Skobject represents CX objects tree

# For developers


```go


c := NewContainer(db, reg)

c.NewRoot(pk, sk, func(r Root) {
	ref := r.Save(asdf)
	refs := r.SaveArraf(a, s, d, f)
	dr := r.Dynamic("cxo.User", User{
		Ref: ref,
		Refs: refs,
		Name: "Alex",
	})

	r.Refs = dr

})

```

