package client

import (
	"os"
	"os/signal"

	"github.com/skycoin/cxo/gui"
	"github.com/skycoin/skycoin/src/util"
)

func (c *Client) launchWebInterface() {

	c.config.WebInterface.GUIDirectory = util.ResolveResourceDirectory(
		c.config.WebInterface.GUIDirectory)

	if c.config.WebInterface.Enable == true {

		logger.Info("starting web interface")
		logger.Info("web interface address: %s", c.config.fullAddress())

		var err error
		if c.config.WebInterface.HTTPS == true {
			logger.Panic("HTTPS support is not implemented yet")
		} else {
			err = gui.LaunchWebInterfaceAPI(c.config.address(),
				c.config.WebInterface.GUIDirectory,
				c.node,
				SkyObjectsAPI(c.skyobject),
				clientApi{c})
		}

		if err != nil {
			logger.Error("failed to start web interface: %v", err)
			return
		}

		c.config.launchBrowser()
		return
	}
	logger.Debug("web interface disabled")
}

//
// utility
//

// WaitInterrupt waits for SIGINT to shutdown. You need to
// call (*Client).Close separately
func (c *Client) WaitInterrupt() {
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	select {
	case s := <-sigint:
		logger.Infof("got signal %q, shutting down...", s)
	case <-c.quit:
		logger.Info("terminate daemon from command line")
	}
}
