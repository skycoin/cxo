package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/skycoin/skycoin/src/cipher"
)

func main() {
	log.SetPrefix("[TEST]")
	log.SetFlags(log.Lshortfile | log.Ltime)

	// generate key pairs
	pub, sec := cipher.GenerateKeyPair()

	// start source
	source := exec.Command("./source/source",
		"-a", "[::]",
		"-p", "44000",
		"-pub", pub.Hex(),
		"-sec", sec.Hex())
	if err := source.Start(); err != nil {
		log.Fatal(err)
	}

	// start drain
	drain := exec.Command("./drain/drain",
		"-a", "[::]",
		"-p", "44006",
		"-pub", pub.Hex())
	if err := drain.Start(); err != nil {
		log.Fatal(err)
	}

	// start node
	node := exec.Command("../cxod/cxod",
		"-address", "[::]",
		"-port", "44001",
		"-name", "NODE",
		pub.Hex(), // subscribe to the feed on start
	)

}
