package client

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/skycoin/cxo/gui"
	"github.com/skycoin/cxo/node"
	"github.com/skycoin/skycoin/src/util"
)

//
// TODO: refactor
//
func RunAPI(c *Config, node node.Node, controllers ...gui.IRouterApi) {
	c.WebInterface.GUIDirectory = util.ResolveResourceDirectory(
		c.WebInterface.GUIDirectory)

	if c.WebInterface.Enable == true {

		logger.Info("starting web interface")
		logger.Info("web interface address: %s", c.fullAddress())

		var err error
		if c.WebInterface.HTTPS == true {
			log.Panic("HTTPS support is not implemented yet")
		} else {
			err = gui.LaunchWebInterfaceAPI(c.address(),
				c.WebInterface.GUIDirectory,
				node,
				controllers...)
		}

		if err != nil {
			logger.Error("failed to start web interface: %v", err)
			return
		}

		c.launchBrowser()
	}

	// subscribe to SIGINT (Ctrl+C)
	sigint := catchInterrupt()

	// waiting for SIGINT (Ctrl+C)
	logger.Info("Got signal %q, shutting down...", <-sigint)
	node.Close() // TOOD

	logger.Info("Cya")
}

// subscribe to SIGINT
func catchInterrupt() <-chan os.Signal {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	return sig
}
