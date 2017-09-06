package node

import (
	"io/ioutil"
	"testing"

	"github.com/skycoin/cxo/node/log"
	stdlog "log"
)

func init() {
	if false == testing.Verbose() {
		stdlog.SetOutput(ioutil.Discard)
	}
}

func testNewConfig() (c Config) {
	c = NewConfig()

	c.PingInterval = 0
	c.EnableRPC = false
	c.InMemoryDB = true

	if false == testing.Verbose() {
		c.Log.Output = ioutil.Discard
		c.Skyobject.Log.Output = ioutil.Discard
	} else {
		c.Log.Debug = true
		c.Log.Pins = log.All
		c.Skyobject.Log.Debug = true
		c.Skyobject.Log.Pins = log.All
	}

	return
}

func TestNode_Close(t *testing.T) {
	c := testNewConfig()
	c.EnableRPC = true                            // to test closing
	c.DiscoveryAddresses = Addresses{"[::]:8008"} // to test closing

	s, err := NewNode(c)
	if err != nil {
		t.Fatal(err)
	}

	if err = s.Close(); err != nil {
		t.Error(err)
	}

	// twice
	if err = s.Close(); err != nil {
		t.Error(err)
	}
}
