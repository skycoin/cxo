skybrief
========

The skybrief is brief example for the skyobject package

- Root is objects root
- Ref is like pointer (`*User`)
- Refs is like slice of pointers (`[]*User`)
- Dynamic is like `interface{}`

The Root contains list of the Dynamic references, every one of which
is root of its subtree, etc, etc.

A Root can be blank.
