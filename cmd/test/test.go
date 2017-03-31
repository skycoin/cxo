package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

func main() {
	log.SetPrefix("[TEST] [ERROR] ")
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
	source.Stderr, source.Stdout = os.Stderr, os.Stdout
	if err := source.Start(); err != nil {
		log.Print(err)
		return
	}
	defer source.Process.Kill()

	// start drain
	drain := exec.Command("./drain/drain",
		"-a", "[::]",
		"-p", "44006",
		"-pub", pub.Hex())
	drain.Stderr, drain.Stdout = os.Stderr, os.Stdout
	if err := drain.Start(); err != nil {
		log.Print(err)
		return
	}
	defer drain.Process.Kill()

	// start node
	node := exec.Command("../cxod/cxod",
		"-address", "[::]",
		"-port", "44001",
		"-name", "NODE",
		"-rpc-address", "[::]:55001",
		"-debug",
		pub.Hex(), // subscribe to the feed on start
	)
	node.Stderr, node.Stdout = os.Stderr, os.Stdout
	if err := node.Start(); err != nil {
		log.Print(err)
		return
	}
	defer node.Process.Kill()

	time.Sleep(1 * time.Second)

	connectToSorce := exec.Command("../cli/cli",
		"-a", "[::]:55001",
		"-e", "connect [::]:44000")
	if err := connectToSorce.Run(); err != nil {
		log.Print(err)
		return
	}

	connectToDrain := exec.Command("../cli/cli",
		"-a", "[::]:55001",
		"-e", "connect [::]:44006")
	if err := connectToDrain.Run(); err != nil {
		log.Print(err)
		return
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	log.Printf("got signal %q, exiting...", <-sig)
}
