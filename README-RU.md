![cxo logo](https://user-images.githubusercontent.com/26845312/32426759-2a7c367c-c282-11e7-87bc-9f0a936046af.png)


[中文文档](./README-CN.md) |
[English](./README.md)


CXO - система обмена объектами на основе блокчейн
=================================================

[![Build Status](https://travis-ci.org/skycoin/cxo.svg)](https://travis-ci.org/skycoin/cxo)
[![GoReportCard](https://goreportcard.com/badge/skycoin/cxo)](https://goreportcard.com/report/skycoin/cxo)
[![Telegram group link](telegram-group.svg)](https://t.me/joinchat/B_ax-A6oCR9eQuAPiJtvaw)
[![Google Groups](https://img.shields.io/badge/google%20groups-skycoincxo-blue.svg)](https://groups.google.com/forum/#!forum/skycoincxo)

CXO - это система объектов, цель которой - обмен любыми объектами (включая
дерево объектов, и обновления в этих деревьях и так далее). CXO - это
библиотека низкого уровня разработаная для построения приложения поверх.

### Быстрый старт и API документация

Смотри [CXO wiki (рус.)](https://github.com/skycoin/cxo/wiki/Get-Started-(Rus)-%D0%9D%D0%B0%D1%87%D0%B0%D0%BB%D0%BE)

### API документация

Смотри [CXO wiki (англ.)](https://github.com/skycoin/cxo/wiki)

### Установка и версии

Используйте [dep](https://github.com/golang/dep) для выбора версии CXO.
Мастер ветка репозитория указывает на последний стабильный релиз.

Установить библиотеку
```
go get -u -t github.com/skycoin/cxo/...
```
И протестировать
```
go test -cover -race github.com/skycoin/cxo/...
```

### Docker

```
docker run -ti --rm -p 8870:8870 -p 8871:8871 skycoin/cxo
```


### Разработка

- [группа в Телеграм (англ.)](https://t.me/joinchat/B_ax-A6oCR9eQuAPiJtvaw)
- [группа в Телеграм (рус.)](https://t.me/joinchat/EUlzX0a5byZxH5MdnAOLLA)
- [Google Groups (eng.)](https://groups.google.com/forum/#!forum/skycoincxo)

#### Модули

- `cmd` - приложения
  - `cxocli` - CLI для администрирования CXO-ноды основаное на RPC
    ([wiki/CLI (англ.)](https://github.com/skycoin/cxo/wiki/CLI)).
  - `cxod` - CXO-демон который принимает все подписки
- `cxoutils` - базовые утилиты
- `data` - интерфейсы базы данных, объекты и ошибки связанные с базой данных
  - `data/cxds` - хранилище типа ключ-значени для объектов, хранилище типа
    хэш -> данные, с некоторыми дополнениями
  - `data/idxdb` - хранилище для корнеывых объектов, подписок и так далее
  - `data/tests` - наборы тестов для интерфейсов бах данных
- `node` - TCP-транспорт с функциями подписки и для получения объектов и
  обмена ими
  - `node/log` - логгер
  - `node/msg` - протокол (поверх протокола `github.com/skycoin/net`)
- `skyobject` - ядро CXO
  - `registry` - схемы, регистры типов, типы данных CXO и так далее

И

- [`intro`](./intro) - примеры


#### Форматирование и стиль

Смотри [CONTRIBUTING.md (англ.)](CONTRIBUTING.md)

#### Что значит номер версии

Испоьзуется MAJOR.MINOR версии. Где MAJOR
- API изменения
- изменения протокола
- изменения в базах данных (другой формат храниния)

и MINOR это
- небольшие измениния API
- поправки, исправления
- улучшения

Таким образом файлы баз данных от разных MAJOR версий не совместимы. И ноды
различных версий не могут коммуницировать.

##### Версии

<!-- 1.0 -->

<details>
<summary>1.0</summary>

не определена

</details>

<!-- 2.1 -->

<details>
<summary>2.1</summary>

- git тэг: `v2.1`
- коммит: `d4e4ab573c438a965588a651ee1b76b8acbb3724`

Gopkg.toml

```toml
[[constraint]]
name = "github.com/skycoin/cxo"
revision = "d4e4ab573c438a965588a651ee1b76b8acbb3724"
```

или

```toml
[[constraint]]
name = "github.com/skycoin/cxo"
version = "v2.1"
```

</details>

<!-- 3.0 -->

<details>
<summary>3.0</summary>

- git тэг: `v3.0`
- коммит: `8bc2f995634cd46d1266e2120795b04b025e0d62`

Gopkg.toml

```toml
[[constraint]]
name = "github.com/skycoin/cxo"
revision = "8bc2f995634cd46d1266e2120795b04b025e0d62"
```

или

```toml
[[constraint]]
name = "github.com/skycoin/cxo"
version = "v3.0"
```

</details>

### Зависимости

Для контроля версий и зависимостей используется [dep](https://golang.github.io/dep/).
Dep помещает все зависимости в подпапку `vendor/`. Установите `dep` используя
ссфлку выше и запустите

```
dep ensure
```

если возникли каки-то проблемы с компиляцией CXO.

---
