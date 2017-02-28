package main

import (
	"log"

	"github.com/skycoin/cxo/client"
)

func main() {
	c, err := client.NewClient()
	if err != nil {
		log.Print(err)
		return
	}
	if err = c.Start(); err != nil {
		log.Print(err)
		return
	}
	defer c.Close()
	c.WaitInterrupt()
}
