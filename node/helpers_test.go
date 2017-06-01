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

func newClientConfig() (conf ClientConfig) {
	conf = NewClientConfig()
	conf.Log.Debug = testing.Verbose()
	conf.InMemoryDB = true
	conf.DataDir = testDataDir
	conf.DBPath = testClientDBPath
	return
}

func newClient(reg *skyobject.Registry) (c *Client, err error) {
	c, err = NewClient(newClientConfig(), reg)
	if err != nil {
		return
	}
	if !testing.Verbose() {
		c.Logger.SetOutput(ioutil.Discard)
	}
	return
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

// newServerConf like newServer, but it uses provided config
func newServerConf(conf ServerConfig) (s *Server, err error) {
	s, err = NewServer(conf)
	if err != nil {
		return
	}
	if !testing.Verbose() {
		s.Logger.SetOutput(ioutil.Discard)
	}
	return
}

func newRunninServer(feeds []cipher.PubKey) (s *Server, err error) {
	s, err = newServer()
	if err != nil {
		return
	}
	if err = s.Start(); err != nil {
		s.Close()
		return
	}
	for _, f := range feeds {
		s.AddFeed(f)
	}
	return
}

// start server and connect given client to the server
// returning the server you need to close later. The server
// subscribed to given list of feeds, but client doesn't
// and some time required to sync between client and server.
// To handle the syncing use ClientConfig.OnAddFeed callback
func startClient(c *Client, feeds []cipher.PubKey) (s *Server, err error) {
	if s, err = newRunninServer(feeds); err != nil {
		return
	}
	if err = c.Start(s.pool.Address()); err != nil {
		s.Close()
		return
	}
	return
}

func newRunningClient(conf ClientConfig, reg *skyobject.Registry,
	feeds []cipher.PubKey) (c *Client, s *Server, err error) {

	if c, err = NewClient(conf, reg); err != nil {
		return
	}
	if s, err = startClient(c, feeds); err != nil {
		c.Close()
		return
	}
	return
}

func addFeedTM(addFeed <-chan cipher.PubKey, t *testing.T) (pk cipher.PubKey) {
	select {
	case pk = <-addFeed:
	case <-time.After(TM):
		t.Fatal("slow")
	}
	return
}
