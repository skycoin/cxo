package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

const (
	CYAN    = "\033[36m"
	GREEN   = "\033[32m"
	MAGENTA = "\033[35m"
)

// copy stdout and stderr of a process to stdout colorizing strings
type ColoredPipe struct {
	color string
}

func (c *ColoredPipe) Write(p []byte) (n int, err error) {
	if _, err = os.Stdout.WriteString(c.color); err != nil {
		return
	}
	if n, err = os.Stdout.Write(p); err != nil {
		return
	}
	_, err = os.Stdout.WriteString("\033[0m") // clear
	return
}

func main() {
	log.SetPrefix("[TEST] ")
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
	sourcePipe := &ColoredPipe{CYAN}
	source.Stderr, source.Stdout = sourcePipe, sourcePipe
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
	drainPipe := &ColoredPipe{MAGENTA}
	drain.Stderr, drain.Stdout = drainPipe, drainPipe
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
	nodePipe := &ColoredPipe{GREEN}
	node.Stderr, node.Stdout = nodePipe, nodePipe
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
