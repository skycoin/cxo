cxod
====

The cxod is cxo daemon.

### Command line flags

#### Main

+ testing  
  enable testing mode; if set, then the daemon will generate some test
  data and send to its subscribers
+ secret-key  
  hexadecmal encoded secret key, if it's not set then it will be generated
  automatically
+ remote-term  
  allow to terminate daemon using commandl ine tool

#### Intercommunication related flags

+ name  
  provide some name for the node that will be used as log prefix for
  logs related to intercommunication
+ d  
  enable debug mode and show debug logs
+ a  
  TCP listening address; set to empty string for arbitrary assignment;
  by default it is empty string
+ p  
  TCP listening port; set to zero for arbitrary assignment
+ max-incoming  
  Maximum incoming connections (subscribers). Set to zero to disable listening
+ max-outgoing  
  Maximum outgoign connections (subscriptions). Set to zero to disable
  subscribing

+ max-pending  
  maximum pending connections; a new connetion must perform
  handshake with other side; the connection is pending untill handshake
  was performed
+ max-msg-len  
  limit of message size
+ event-queue  
  size of queue of events, such as broadcast, send, etc
+ result-queue  
  size of queue of results of sending
+ write-queue  
  write queue size of connection
+ dt  
  dial timeout (set to zero to ignore)
+ rt  
  reading timeout (set to zero to use system's default)
+ wt  
  write timeout (set to zero to use system's default)
+ ping  
  ping interval (set to zero to disable pinging); it's better to set some
  ping interval if you expect to use nodes that send-receive messages
  too infrequently
+ ht  
  handshake timeout (set to zero to disable)
+ rate  
  messages handling rate (set to zero to use immediate handling)
+ man-chan-size  
  size of channel of managing events, such as request list of connections

#### Web-interface related flags

+ web-interface  
  enable the web interface
+ web-interface-port  
  port to serve web interface on
+ web-interface-addr  
  addr to serve web interface on
+ web-interface-cert  
  cert.pem file for web interface HTTPS. If not provided, will use cert.pem
  in -data-directory
+ web-interface-key  
  key.pem file for web interface HTTPS. If not provided, will use key.pem
  in -data-directory
+ web-interface-https  
  enable HTTPS for web interface
+ launch-browser  
  launch system default webbrowser at client startup
+ gui-dir  
  static content directory for the html gui

+ data-dir  
  directory to store app data (defaults to ~/.skyhash)

#### Logs related flags

+ log-level  
  Choices are: debug, info, notice, warning, error, critical
+ color-log  
  Add terminal colors to log output
