client
======

The client is a core of cxo daemon. The client launches client-to-client
communication and web-interface to manage.

### How it works

Every client can be a feed and has subscribers simulatenously. When new
data appears on the client it sends announce to own subscribers. If
some subscriber has got announce but has not data it send request back to
get data

```
[feed] --> announce (SHA256) --> [s1]
[feed] <-- request (SHA256)  <-- [s1]
[feed] --> data              --> [s1]
```

That's all
