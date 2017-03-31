package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"

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
		"-r", "55000", // rpc address
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
	if err := node.Start(); err != nil {
		log.Fatal(err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	log.Printf("got signal %q, exiting...", <-sig)

}
