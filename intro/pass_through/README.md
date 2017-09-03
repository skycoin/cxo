Pass Through
============


### Briefly

Server -> Pipe -> Client

### Explain

The Server generates data. The pipe knows nothing about data types. It
conenects to the Server and subscribes to the same feed. Thus, the
Server and the Pipe exchange this feed. The Client conencts to the Pipe
and subscribes to the feed.

Other words, the Server generates data, and the Client receives the data
through the Pipe.
