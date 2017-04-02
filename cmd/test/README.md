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
# build conductor (conductor is WHITE, node is GREEN)
cd cmd/test
go build

# build the source (CYAN)
cd source
go build
cd ..

# build the drain (MAGENTA)
cd drain
go build
cd ..

# run everything
./test
```

### Result

Every 5 seconds the drain print something like that

```
Inspect
=======
---
<struct Board>
Head: "Board #1"
Threads: <slice -A>
  <reference>
      missing object "-S": e6bd9e18302d5a09b377a756a3215530206cedcac999e03c7f33d1df6eb818d3
  <reference>
      missing object "-S": b0104666ac12ff87ecf8c28b850bc7188b16385d561d345c0bd87e33d84ea677
  <reference>
      missing object "-S": 91236073b9d97b2725671b565ac96dd7c8af2cf39c6ec51d7d0ba3ea0a19cb1f
  <reference>
      missing object "-S": 83c2bef900a49975838cc016892757d6e260ba802eaf98ab4c8409604ab0cdc7
Owner: <reference>
    missing schema: 4d0ddc0fe6a6be8a82cac5dc1d52e58d0f5dc1c26c472d8285429b87edf61bd7
---
---
<struct Board>
Head: "Board #2"
Threads: <slice -A>
  <reference>
      missing object "-S": b87b81bc5b7d5db62d350a972c4cd81a6baa119c7dffc0eefa41fcd5a2974b88
Owner: <reference>
    missing schema: 096e09dabf97787df9f952215c45cb9faa53e9213bad3dbb75894d349d8573b8
---
```

The result is wrong. Because it replicates root objects but don't replicate
the data the root objects refer to.

I think that public API of the skyobject, of the node, of the data,
of the all RPC-packages, of cmd/cli and of cmd/cxod are stable. I will change
something internal from the node.
