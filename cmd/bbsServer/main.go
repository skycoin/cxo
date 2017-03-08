package main

import (
	"flag"
	"net"
	"net/http"
	"net/rpc"
)

var (
	port = flag.String("port", "1235", "port to serve BBS server")
)

func main() {
	if e := rpc.Register(NewServer()); e != nil {
		panic(e)
	}
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":"+*port)
	if e != nil {
		panic(e)
	}
	if e := http.Serve(l, nil); e != nil {
		panic(e)
	}
}
