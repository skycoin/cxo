test
====

### Structure

```
cmd/test        - run source, drain and intermediate cxod
cmd/test/source - generate two filled root objects
cmd/test/drain  - print its tree every 5 seconds
```

All applications subscribed to the same public key. The intermediate cxod
connects to source and connects to drain.

This way the inetermediate cxod will take data from the source,
keep the data and pass the data to the drain.

### Run

```
# working dir
cd $GOPATH/src/github.com/skycoin/cxo/cmd/test

# build cli
cd ../cli
go build

# build cxod
cd ../cxod
go build


# build conductor (conductor is WHITE, node is GREEN)
cd ../test
go build

# build the source (CYAN)
cd source
go build
cd ..

# build the drain (MAGENTA)
cd drain
go build
cd ..

# run everything (including GREEN cxod)
./test

# hit Ctrl+C to terminate
```

### Result

The drain ccheck its tree every 5 seconds. If the tree updated, then the drain
print somthing like following text

```
Inspect
=======
---
<struct Board>
Head: "Board #1"
Threads: <slice -A>
  <reference>
    <Blah>
Owner: <reference>
    <Blah-Blah>
---
---
<struct Board>
Head: "Board #2"
Threads: <slice -A>
  <reference>
    <Blah>
Owner: <reference>
    <Blah-Blah>
---
```

### Bash script

```sh
cd $GOPATH/src/github.com/skycoin/cxo/cmd/test
cd ../cli && go build
cd ../cxod && go build
cd ../test && go build
cd source && go build && cd ..
cd drain && go build && cd ..
./test
```
