package main

import (
	"flag"
	"net"
	"net/http"
	"net/rpc"

	"github.com/skycoin/skycoin/src/cipher"
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

// SHA256SliceHas determines if a value is included in slice.
func SHA256SliceHas(slice []cipher.SHA256, check cipher.SHA256) bool {
	for _, v := range slice {
		if v == check {
			return true
		}
	}
	return false
}
