package node

import (
	//"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
	//"github.com/skycoin/skycoin/src/cipher"
	//"github.com/skycoin/cxo/skyobject"
)

// timeout to fail slow opertions
const TM time.Duration = 50 * time.Millisecond

var (
	testDataDir      = filepath.Join(".", "test")
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
// listening enabled by agrument
func newNodeConfig(listen bool) (conf NodeConfig) {
	conf = NewNodeConfig()
	conf.Log.Debug = testing.Verbose()
	conf.Listen = "" // arbitrary assignment
	conf.EnableListener = listen

	conf.EnableRPC = false

	conf.InMemoryDB = true
	conf.DataDir = testDataDir
	conf.DBPath = testServerDBPath
	return
}
