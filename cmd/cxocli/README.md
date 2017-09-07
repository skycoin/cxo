CXOCLI
======

CXOCLI is command-line interface for a CXO node.

### Flags

Command-line flas are:

- `-a` - RPC address, defaults to `[::]:8997`
- `-e` - execute cgiven command and exit
- `-h` - show help about command-line flags

### Commands

- `add feed <public key>` - start shareing given feed
- `del feed <public key>` - stop sharing given feed
- `subscribe <address> <pub key>` - subscribe to feed of a connected peer
- `unsubscribe <address> <public key>` - unsubscribe from feed of a connected
  peer
- `connect <address>` - connect to node with given address
- `disconnect <address>` - disconnect from given address
- `connections` - list all connections, in the list "-->" means that this
  connection is incoming and "<--" means that this connection is outgoing; (✓)
  means that conenction is established, and (⌛) means that this connection is
  establishing
- `incoming connections` - list all incoming connections
- `outgoing connections` - list all outgoing connections
- `connection <address>` - list feeds of given connection
- `feed <public key>` - list connections of given feed
- `feeds` - list all feeds
- `roots <public key>` - print brief information about all root objects of
  given feed
- `tree <pub key> [seq]` - print root by public key and seq number, if the seq
  omitted then last full root printed
- `info` - get brief information about node
- `listening address` - print listening address
- `stat` - statistic
- `terminate` - terminate server if allowed
- `help` - show help message
- `quit` and `exit` - leave the cli

### History

CXOCLI saves history into `.cxocli.history` path at home root.
