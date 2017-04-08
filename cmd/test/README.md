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

##### RPC addresses

+ source "[::]:55000"
+ drain "[::]:55006"
+ cxod (pipe) "[::]:55001"

##### Explore running instances

Example for the `source`

```bash
cd cmd/cli
./cli -a "[::]:55000"
> blah
```

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

The drain checks its tree every 5 seconds. If the tree updated, then the drain
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

There is `run.sh` file
