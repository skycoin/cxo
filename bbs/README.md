RUN
===

##### daemon

```
cd cxo/cmd/cxod
go build
./cxod -debug -remote-close
```

##### cli

```
cd cxo/cmd/cli
go build
./cli
> add_feed 03f4ceb316a5cbdb9e128f965d2622ea2fbe2f66a068747f46c31f2387175929cd
```

#### bbs/generate

```
cd cxo/bbs/generate
go build
./generate -pk 03f4ceb316a5cbdb9e128f965d2622ea2fbe2f66a068747f46c31f2387175929cd -sk df0b49643782ca78befa8d88f2095a0c21fccf0757e7da7201fbb43571b59751 -debug
```

#### bbs/receive

```
cd cxo/bbs/receive
go build
./receive -debug -pk 03f4ceb316a5cbdb9e128f965d2622ea2fbe2f66a068747f46c31f2387175929cd
```
