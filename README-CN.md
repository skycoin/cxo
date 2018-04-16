![cxo logo](https://user-images.githubusercontent.com/26845312/32426759-2a7c367c-c282-11e7-87bc-9f0a936046af.png)

CXO 对象存储系统
================

[![Build Status](https://travis-ci.org/skycoin/cxo.svg)](https://travis-ci.org/skycoin/cxo)
[![GoReportCard](https://goreportcard.com/badge/skycoin/cxo)](https://goreportcard.com/report/skycoin/cxo)
[![Telegram group link](telegram-group.svg)](https://t.me/joinchat/B_ax-A6oCR9eQuAPiJtvaw)
[![Google Groups](https://img.shields.io/badge/google%20groups-skycoincxo-blue.svg)](https://groups.google.com/forum/#!forum/skycoincxo)


CXO 是一个对象系统，它的作用是用来分享不同的对象。CXO是一个底层的平台，可以在它的上面构建更多的应用。


（注：对象存储，也叫做基于对象的存储，是用来描述解决和处理离散单元的方法的通用术语，这些离散单元被称作为对象。
就像“文件”一样，“对象”包含数据，但是和文件不同的是，对象在一个层结构中不会再有层级结构。每个对象都在一个被称作存储池的扁平地址空间的同一级别里，一个对象不会属于另一个对象的下一级。
文件和对象都有与它们所包含的数据相关的元数据，但是对象是以扩展元数据为特征的。每个对象都被分配一个唯一的标识符，允许一个服务器或者最终用户来检索对象，而不必知道数据的物理地址。这种方法对于在云计算环境中自动化和简化数据存储有帮助。）


### 开始使用与API接口文档


参阅 [CXO wiki](https://github.com/skycoin/cxo/wiki/Get-Started) 包含相关信息（尚未完善）

### API文档

参阅 [CXO wiki](https://github.com/skycoin/cxo/wiki) 
包含相关信息（尚未完善）

### 安装与版本

使用[dep](https://github.com/golang/dep)特定版本来使用CXO。存储库的主分支指向最新的稳定版本。实际上，它现在是alpha发布。


开始使用
```
go get -u -t github.com/skycoin/cxo/...
```
测试全部的包
```
go test -cover -race github.com/skycoin/cxo/...
```

### 使用 Docker

```
docker run -ti --rm -p 8870:8870 -p 8871:8871 skycoin/cxo
```


### 开发社区

- [telegram group (eng.)](https://t.me/joinchat/B_ax-A6oCR9eQuAPiJtvaw)
- [telegram group (rus.)](https://t.me/joinchat/EUlzX0a5byZxH5MdnAOLLA)
- [google group (eng.)](https://groups.google.com/forum/#!forum/skycoincxo)

#### 模块

- `cmd` - apps
  - `cxocli` - CLI是管理基于RPC的工具来控制任何CXO节点
    ([wiki/CLI](https://github.com/skycoin/cxo/wiki/CLI)).
  - `cxod` - 一个CXO的守护进程，调和接受所有订阅
- `cxoutils` - 基础设施
- `data` - 数据库接口、对象和错误 
  - `data/cxds` - CX数据存储是键值存储的实现。
  - `data/idxdb` - 执行索引数据库
  - `data/tests` - “`data`”接口的测试
- `node` - 对于CXO 的 TCP传输
  - `node/log` - 记录器
  - `node/msg` - 协议消息
- `skyobject` - cxo编码/解码核心，等
  - `registry` - 模式、类型等， 

以及

- [`intro`](./intro) - 实例


#### 格式化和编码风格

参阅 [CONTRIBUTING.md](CONTRIBUTING.md) 包含更多细节。

#### 版本控制

 CXO 使用 主/副 版本.  “主” 体现在哪里？
- API的变化
- 协议的变化
- 数据表达的变化

“副”是： 
- API变化小
- 修复
- 改进

因此，db文件在不同的主要版本之间是不兼容的。节点不同的主要版本不能沟通。保存的数据可能有另一个不同的表示。

##### 版本

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

### 依赖

依赖项是采用  [dep](https://github.com/golang/dep).
（Dep 是 Go 依赖管理工具） 

安装 `dep`:

```sh
go get -u github.com/golang/dep
```

`dep` vendors 会引入所有的依赖到库中。


如果更改依赖项，则应根据需要使用`dep ensure`更新它们。


使用 `dep help`帮助文档查阅相关说明，或更新它们。


在添加一个新的依赖项（使用`dep ensure`）后，运行`dep prune`删除任何不必要的子包的依赖关系。


更新或初始化时，`dep` 将找到最新版本的依赖项将其编译。


实例:

初始化所有依赖项：

```sh
dep init
dep prune
```

更新所有依赖项：

```sh
dep ensure -update -v
dep prune
```

添加一个独立的依赖项（最新版本）：

```sh
dep ensure github.com/foo/bar
dep prune
```

添加一个独立的（更具体的版本），或降级现有的依赖项:

```sh
dep ensure github.com/foo/bar@tag
dep prune
```


---
