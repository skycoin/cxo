package gui

import (
	//"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	//"os"
	"path/filepath"
	//"strings"

	"gopkg.in/op/go-logging.v1"

	"github.com/skycoin/skycoin/src/util"
	//http,json helpers
	"github.com/skycoin/cxo/nodeManager"
	"github.com/skycoin/cxo/skyobject"
)

var (
	logger = logging.MustGetLogger("skyhash.gui")
)

const (
	resourceDir = "dist/"
	devDir = "dev/"
	indexPage = "index.html"
)

// Begins listening on http://$host, for enabling remote web access
// Does NOT use HTTPS
func LaunchWebInterfaceAPI(host, staticDir string, shm *nodeManager.Manager, schemaProvider skyobject.ISkyObjects, messanger *Messenger) error {
	logger.Info("Starting web interface on http://%s", host)
	logger.Warning("HTTPS not in use!")
	logger.Info("Web resources directory: %s", staticDir)

	appLoc, err := util.DetermineResourcePath(staticDir, devDir, resourceDir)
	if err != nil {
		return err
	}

	router := NewRouter()

	router.StaticFile("/", filepath.Join(appLoc, indexPage))

	// register API handlers
	RegisterNodeManagerHandlers(router, shm)
	RegisterSchemaHandlers(router, schemaProvider, messanger)

	RegisterStaticFolders(router, appLoc)

	/*
		// Wallet interface
		RegisterWalletHandlers(mux, daemon.Gateway)
	*/

	// Runs http.Serve() in a goroutine
	serve(router, host)

	return nil
}

func serve(router *Router, address string) {
	// http.Serve() blocks
	// Minimize the chance of http.Serve() not being ready before the
	// function returns and the browser opens
	ready := make(chan struct{})
	go func() {
		ready <- struct{}{}
		if err := router.Serve(address); err != nil {
			log.Panic(err)
		}
	}()
	<-ready
}

//move to util

// Creates an http.ServeMux with handlers registered
func RegisterStaticFolders(router *Router, appLoc string) {

	fileInfos, _ := ioutil.ReadDir(appLoc)
	for _, fileInfo := range fileInfos {
		route := fmt.Sprintf("/%s", fileInfo.Name())
		if fileInfo.IsDir() {
			route = route + "/"
		}
		router.Folder(route, appLoc)
	}

}
