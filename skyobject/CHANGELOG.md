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

*There are breaking changes*

- [x] rename `(*Root).Inject, InjectMany -> Append`

Also:

+ introduced `(*Root).Slice(i, j int) (dr []Dynamic, err error)` method
+ introduced `(*Root).Index(i int) (dr Dynamic, err error)` method

So, the changes are for semantics. A `Root` contains slice of references and
methods `Append`, `Index` and `Slice` are intuitive. Methods `Inject` and
`InjectMany` are not depricated, but removed

####  1:00 16 June 2017 UTC

+ added `(*Root).Inspect() (tree string)` method that retuerns printed tree of
the Root
+ added `(*Container).RootByHash() (r *Root, ok bool)`
+ added `(*Container).RangeFeed(RangeFeedFun) (err error)`
