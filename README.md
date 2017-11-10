![cxo logo](https://user-images.githubusercontent.com/26845312/32426759-2a7c367c-c282-11e7-87bc-9f0a936046af.png)

CX Object System
================

[![Build Status](https://travis-ci.org/skycoin/cxo.svg)](https://travis-ci.org/skycoin/cxo)
[![Coverage Status](https://coveralls.io/repos/skycoin/cxo/badge.svg?branch=master)](https://coveralls.io/r/skycoin/cxo?branch=master)
[![GoReportCard](https://goreportcard.com/badge/skycoin/cxo)](https://goreportcard.com/report/skycoin/cxo)
[![Telegram group link](telegram-group.svg)](https://t.me/joinchat/B_ax-A6oCR9eQuAPiJtvaw)

The CXO is objects system, goal of which is sharing any objects. The CXO
is low level and designed to build application on top of it

### Get Started and API Documentation

See [CXO wiki](https://github.com/skycoin/cxo/wiki/Get-Started) to get this information

### API Documentation

See [CXO wiki](https://github.com/skycoin/cxo/wiki) to get this information

### Installation and Version

Use [dep](https://github.com/golang/dep) to use particular version of the
CXO. The master branch of the repository points to latest stable release.
Actually, it points to alpha-release for now.

To get the release use
```
go get -u -t github.com/skycoin/cxo/...
```
Test all packages
```
go test -cover -race github.com/skycoin/cxo/...
```


### Development

- [telegram group](https://t.me/joinchat/B_ax-A6oCR9eQuAPiJtvaw)

#### Modules

- `cmd` - apps
  - `cxocli` - CLI is admin RPC based tool to control any CXO-node
    ([wiki/CLI](https://github.com/skycoin/cxo/wiki/CLI)).
  - `cxod` - an example CXO daemon that accepts all subscriptions
- `data` - database interfaces, obejcts and errors
  - `data/cxds` - CX data store is implementation of key-value store
  - `data/idxdb` - implementation of index DB
  - `data/tests` - tests for the `data` interfaces
- `node` - TCP transport for CXO
  - `node/gnet` - raw TCP transport
  - `node/log` - logger
  - `node/msg` - protocol messages
- `skyobject` - CXO core: schemas, encode/decode, etc

And

- `intro` - examples
  - `intro/exchange` - three nodes example (doesn't ready)
    [README.md](intro/exchange)
<!-- - `intro/cxtweet` - CXO based command-line tweetter like app -->


#### Formatting and Coding Style

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.


### Dependencies

Dependencies are managed with [dep](https://github.com/golang/dep).

To install `dep`:

```sh
go get -u github.com/golang/dep
```

`dep` vendors all dependencies into the repo.

If you change the dependencies, you should update them as needed with
`dep ensure`.

Use `dep help` for instructions on vendoring a specific version of a dependency,
or updating them.

After adding a new dependency (with `dep ensure`), run `dep prune` to remove any
unnecessary subpackages from the dependency.

When updating or initializing, `dep` will find the latest version of a
dependency that will compile.

Examples:

Initialize all dependencies:

```sh
dep init
dep prune
```

Update all dependencies:

```sh
dep ensure -update -v
dep prune
```

Add a single dependency (latest version):

```sh
dep ensure github.com/foo/bar
dep prune
```

Add a single dependency (more specific version), or downgrade an existing
dependency:

```sh
dep ensure github.com/foo/bar@tag
dep prune
```


---
