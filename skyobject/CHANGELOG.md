changes
=======

### TODO

- [ ] move `node.Rootwalker` here
- [ ] add test for `RemoveNonFullRoots`
- [ ] move to new approach of handling Root objects
- [ ] add `Schema.HasReferences() bool` method for performance


### Done

#### 10:00 28 June 2017

- [x] add ability to remove all non-full root objects
- [x] add test for `(*Root).Index`
- [x] add test for `(*Root).Slice`
- [x] add tests for `(*Container).RangeFeed`
- [x] add tests for `(*Container).RootByHash`

and

- add test for `(*Root).Len`

Introducing `(*Container).RemoveNonFullRoots` method. The method removes
non-full roots of a feed from database. The method used by node to clean up
during shutdown


####  4:15 22 June 2017 UTC

- [x] fix user-provided-schema-name issue #98

#### 22:00 19 June 2017 UTC

- fix `Root.WantFunc` blank reference bug
- improve `Root.Inspect` method

#### 16:15 17 June 2017 UTC

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
