changes
=======

### TODO

- [ ] add ability to remove all non-full root objects
- [ ] move `node.Rootwalker` here
- [ ] fix user-provided-schema-name issue #98
- [ ] add tests for `(*Container).RangeFeed`
- [ ] add tests for `(*Container).RootByHash`
- [ ] rename `(*Root).Inject, InjectMany -> Append`

### Done

####  1:00 16 June 2017 UTC

+ added `(*Root).Inspect() (tree string)` method that retuerns printed tree of
the Root
+ added `(*Container).RootByHash() (r *Root, ok bool)`
+ added `(*Container).RangeFeed(RangeFeedFun) (err error)`
