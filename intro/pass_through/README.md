Pass Through
============


### Briefly

Server -> Pipe -> Client

### Explain

The Server generates data. The pipe knows nothing about data types. It
conenects to the Server and subscribes to the same feed. Thus, the
Server and the Pipe exchange this feed. The Client conencts to the Pipe
and subscribes to the feed.

Other words, the Server generates data, and the Client receives the data
through the Pipe.

### Run

Three terminals required.

Launch the server first
```
cd $GOPATH/src/github.com/skycoin/cxo/intro/pass_through/server
rm -f *.db && go build && ./server
```

In another terminal launch the pipe
```
cd $GOPATH/src/github.com/skycoin/cxo/intro/pass_through/pipe
rm -f *.db && go build && ./pipe
```

And finally, in another terminal lauch the client
```
cd $GOPATH/src/github.com/skycoin/cxo/intro/pass_through/client
rm -f *.db && go build && ./client
```

Feed represents
```
Root -> Content
        |
        +--> Thread --> []Vote
        |
        +--> Post   --> []Vote
```
And every second the server appends some votes.

---
