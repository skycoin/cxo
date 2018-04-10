discovery
=========

A discovery server used to connect cxo nodes between, depending on
feeds they interest and "public" flag.

There are examples:

- [`exchange/`](./exchange) - two nodes exchange feeds
- [`through/`](./through) - two nodes exchange feeds through third one

and

- [`discovery/`](./discovery) - discovery server's used for examples above;
in real life `github.com/skycoin/skywire/cmdx/discovery`
- [`discovery/db`](./discovery/db) - database used for the discovery above;
since original one (`github.com/skycoin/skywire/discovery/db`) can't be used
because of `vendor/` imports

