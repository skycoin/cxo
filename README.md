![cxo logo](https://user-images.githubusercontent.com/26845312/32426759-2a7c367c-c282-11e7-87bc-9f0a936046af.png)


[中文文档](./README-CN.md) |
[по русски](./README-RU.md)


CX Object System
================

[![Build Status](https://travis-ci.org/skycoin/cxo.svg)](https://travis-ci.org/skycoin/cxo)
[![GoReportCard](https://goreportcard.com/badge/skycoin/cxo)](https://goreportcard.com/report/skycoin/cxo)
[![Telegram group](telegram-group.svg)](https://t.me/joinchat/B_ax-A6oCR9eQuAPiJtvaw)
[![Google Groups](https://img.shields.io/badge/google%20groups-skycoincxo-blue.svg)](https://groups.google.com/forum/#!forum/skycoincxo)


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

### Docker

```
docker run -ti --rm -p 8870:8870 -p 8871:8871 skycoin/cxo
```


### Development

- [telegram group (eng.)](https://t.me/joinchat/B_ax-A6oCR9eQuAPiJtvaw)
- [telegram group (rus.)](https://t.me/joinchat/EUlzX0a5byZxH5MdnAOLLA)
- [google group (eng.)](https://groups.google.com/forum/#!forum/skycoincxo)

#### Modules

- `cmd` - apps
  - `cxocli` - CLI is admin RPC based tool to control any CXO-node
    ([wiki/CLI](https://github.com/skycoin/cxo/wiki/CLI)).
  - `cxod` - an averga CXO daemon that accepts all subscriptions
- `cxoutils` - basic utilities
- `data` - database interfaces, objects and errors
  - `data/cxds` - CX data store is implementation of key-value store
  - `data/idxdb` - implementation of index DB
  - `data/tests` - tests for the `data` interfaces
- `node` - TCP transport for CXO
  - `node/log` - logger
  - `node/msg` - protocol messages
- `skyobject` - CXO core: encode/decode, etc
  - `registry` - schemas, types, etc,

And

- [`intro`](./intro) - examples


#### Formatting and Coding Style

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

#### Versioning

The CXO uses MAJOR.MINOR versions. Where MAJOR is
- API changes
- protocol changes
- data representation changes

and MINOR is
- small API changes
- fixes
- improvements

Thus, DB files are not compatible between different major versions. Nodes
with different major versions can't communicate. Saved data may have another
representation.

##### Versions

<!-- 1.0 -->

<details>
<summary>1.0</summary>

not defined

</details>

<!-- 2.1 -->

<details>
<summary>2.1</summary>

- git tag: `v2.1`
- commit: `d4e4ab573c438a965588a651ee1b76b8acbb3724`

Gopkg.toml

```toml
[[constraint]]
name = "github.com/skycoin/cxo"
revision = "d4e4ab573c438a965588a651ee1b76b8acbb3724"
```

or

```toml
[[constraint]]
name = "github.com/skycoin/cxo"
version = "v2.1"
```

</details>

<!-- 3.0 -->

<details>
<summary>3.0</summary>

- git tag: `v3.0`
- commit: `8bc2f995634cd46d1266e2120795b04b025e0d62`

Gopkg.toml

```toml
[[constraint]]
name = "github.com/skycoin/cxo"
revision = "8bc2f995634cd46d1266e2120795b04b025e0d62"
```

or

```toml
[[constraint]]
name = "github.com/skycoin/cxo"
version = "v3.0"
```

</details>

### Dependencies

Dependencies are managed with [dep](https://golang.github.io/dep/). The dep
place all dependencies in `vendor/` subfolder. Install the `dep` using link
above and call

```
dep ensure
```

if you have problems with building the CXO.

---
