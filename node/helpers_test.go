package node

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"
)

// timeout to fail slow opertions
const TM time.Duration = 50 * time.Millisecond

var (
	testDataDir      = filepath.Join(".", "test")
	testClientDBPath = filepath.Join(testDataDir, "client.db")
	testServerDBPath = filepath.Join(testDataDir, "server.db")
)

func clean() {
	os.RemoveAll(testDataDir)
}

func shouldPanic(t *testing.T) {
	if recover() == nil {
		t.Error("missing panic")
	}
}

// name for logs (empty for default)
// memory - to use databas in memory (otherwise it will be ./test/test.db)
func newServerConfig() (conf ServerConfig) {
	conf = NewServerConfig()
	conf.Log.Debug = testing.Verbose()
	conf.Listen = "" // arbitrary assignment

	conf.EnableRPC = false

	conf.InMemoryDB = true
	conf.DataDir = testDataDir
	conf.DBPath = testServerDBPath
	return
}

// newServer returns testing server. Listenng address of the
// server is empty and any possible address will be used.
// To determine address of the server use
//
//     s.pool.Address()
//
// after Start
func newServer() (*Server, error) {
	return newServerConf(newServerConfig())
}
