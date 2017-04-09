package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

const (
	CYAN    = "\033[36m" // source [::]:44000 [::]:55000
	MAGENTA = "\033[35m" // drain [::]:44006 [::]:55006

	RED   = "\033[31m" // pipe 1 [::]:44001 [::]:55001
	GREEN = "\033[32m" // pipe 2 [::]:44002 [::]:55002
	BROWN = "\033[33m" // pipe 3 [::]:44003 [::]:55003
	BLUE  = "\033[34m" // pipe 4 [::]:44004 [::]:55004
	GRAY  = "\033[37m" // pipe 5 [::]:44005 [::]:55005
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
		"-a", "[::]:44000",
		"-r", "[::]:55000", // rpc address
		"-pub", pub.Hex(),
		"-sec", sec.Hex())
	sourcePipe := &ColoredPipe{CYAN}
	source.Stderr, source.Stdout = sourcePipe, sourcePipe
	if err := source.Start(); err != nil {
		log.Print(err)
		return
	}
	defer source.Process.Signal(os.Kill)

	// start drain
	drain := exec.Command("./drain/drain",
		"-a", "[::]:44006",
		"-r", "[::]:55006",
		"-pub", pub.Hex())
	drainPipe := &ColoredPipe{MAGENTA}
	drain.Stderr, drain.Stdout = drainPipe, drainPipe
	if err := drain.Start(); err != nil {
		log.Print(err)
		return
	}
	defer drain.Process.Signal(os.Kill)

	type Pipe struct {
		Color  string
		Listen string
		RPC    string
	}

	//
	// start pipes
	//
	var prev Pipe = Pipe{Listen: "[::]:44000"} // previous (source)
	for i, p := range []Pipe{
		{RED, "[::]:44001", "[::]:55001"},
		{GREEN, "[::]:44002", "[::]:55002"},
		{BROWN, "[::]:44003", "[::]:55003"},
		{BLUE, "[::]:44004", "[::]:55004"},
		{GRAY, "[::]:44005", "[::]:55005"},
	} {
		// start node
		fmt.Printf(`../cxod/cxod -address %s \
			-log-prefix %s  \
			-debug          \
			-rpc-address %s \
			%s
`,
			p.Listen, fmt.Sprintf("'NODE #%d'", i+1), p.RPC, pub.Hex())
		node := exec.Command("../cxod/cxod",
			"-address", p.Listen,
			"-log-prefix", fmt.Sprintf("NODE #%d", i+1),
			"-debug",
			"-rpc-address", p.RPC,
			pub.Hex(), // subscribe to the feed on start
		)
		nodePipe := &ColoredPipe{p.Color}
		node.Stderr, node.Stdout = nodePipe, nodePipe
		if err := node.Start(); err != nil {
			log.Print(err)
			return
		}
		defer node.Process.Signal(os.Kill)

		time.Sleep(1 * time.Second)

		fmt.Printf("../cli/cli -a %s -e connect %s\n", p.RPC, prev.Listen)
		connectToPrevious := exec.Command("../cli/cli",
			"-a", p.RPC,
			"-e", "connect "+prev.Listen)
		if err := connectToPrevious.Run(); err != nil {
			log.Print(err)
			return
		}

		prev = p // keep previous

	}

	fmt.Printf("../cli/cli -a %s -e connect [::]:44006\n", prev.RPC)
	connectToDrain := exec.Command("../cli/cli",
		"-a", prev.RPC,
		"-e", "connect [::]:44006") // address of the drain
	if err := connectToDrain.Run(); err != nil {
		log.Print(err)
		return
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	log.Printf("got signal %q, exiting...", <-sig)
}
