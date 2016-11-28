package main

import "github.com/skycoin/cxo/replicator"

func main() {
	client := replicator.Client()
	client.Run()
}
