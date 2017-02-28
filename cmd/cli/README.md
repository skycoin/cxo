cli
===

The cli is command line tool for cxo daemon

### Command line flags

+ a  
  http-address of daemon
+ t  
  request/response timeout
+ h
  show help
+ d  
  print debug logs
+ e
  execute given command and exit, for example `./cli -e 'list subscriptions'`

### Commands

+ list subscriptions  
  list all subscriptions
+ list subscribers  
  list all subscribers
+ add subscription <address> [desired public key]  
  add subscription to given address, the public key is optional
+ remove subscription <id or address>  
  remove subscription by id or address
+ remove subscriber <id or address>  
  remove subscriber by id or address
+ stat  
  get statistic (total objects, memory) of all objects
+ info  
  print node id and address
+ close  
  terminate daemon
+ help  
  show this help message
+ exit or quit  
  quit cli
