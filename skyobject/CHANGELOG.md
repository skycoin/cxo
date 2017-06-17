changes
=======

### TODO

- [ ] add ability to remove all non-full root objects
- [ ] move `node.Rootwalker` here
- [ ] fix user-provided-schema-name issue #98
- [ ] add tests for `(*Container).RangeFeed`
- [ ] add tests for `(*Container).RootByHash`
- [ ] add test for `(*Root).Index`
- [ ] add test for `(*Root).Slice`

### Done

#### 13:20 16 June 2017 UTC

- [x] `(*Root).Inject, InjectMany -> Append`

+ `(*Root).Inject` and `injectMany` has been marked as __deprecated__
+ introduced `(*Root).Append()drs ...Dynamic)` method
+ introduced `(*Root).Slice(i, j int) (dr []Dynamic, err error)` method
+ introduced `(*Root).Index(i int) (dr Dynamic, err error)` method
+ introduced `(*Root).Len() int` method

So, the changes are for semantics. A `Root` contains slice of references and
methods `Append`, `Index` and `Slice` are intuitive

The `Slice` method returns unsafe slice. But `Refs` method returns
safe copy of references of a `Root`.


####  1:00 16 June 2017 UTC

+ added `(*Root).Inspect() (tree string)` method that retuerns printed tree of
the Root
+ added `(*Container).RootByHash() (r *Root, ok bool)`
+ added `(*Container).RangeFeed(RangeFeedFun) (err error)`
