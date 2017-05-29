package node

import (
	"os"
	"path/filepath"
	"testing"
)

var (
	testDataDir = filepath.Join(".", "test")
	testDBPath  = filepath.Join(testDataDir, "test.db")
)

func clean() {
	os.Remove(testDataDir)
}

func TestNewClient(t *testing.T) {
	t.Run("memory db", func(t *testing.T) {
		clean()
		conf := NewClientConfig()
		conf.InMemoryDB = true
		conf.DataDir = testDataDir
		conf.DBPath = testDBPath
		c, err := NewClient(conf, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if _, err := os.Stat(testDataDir); err == nil {
			t.Error("unexpected data dir")
			if _, err = os.Stat(testDBPath); err == nil {
				t.Error("unexpected db file")
			} else if !os.IsNotExist(err) {
				t.Error("unexpected error")
			}
		} else if !os.IsNotExist(err) {
			t.Error("unexpected error:", err)
		}
	})
	t.Run("data dir", func(t *testing.T) {
		clean()
		conf := NewClientConfig()
		conf.InMemoryDB = false
		conf.DataDir = testDataDir
		conf.DBPath = testDBPath
		c, err := NewClient(conf, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if _, err := os.Stat(testDataDir); err != nil {
			t.Error(err)
		} else {
			if _, err := os.Stat(testDBPath); err != nil {
				t.Error(err)
			}
		}
	})
	t.Run("dbpath", func(t *testing.T) {
		clean()
		conf := NewClientConfig()
		conf.InMemoryDB = false
		conf.DataDir = testDataDir
		conf.DBPath = filepath.Join(testDataDir, "another.db")
		c, err := NewClient(conf, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if _, err := os.Stat(testDataDir); err != nil {
			t.Error(err)
		} else {
			if _, err := os.Stat(conf.DBPath); err != nil {
				t.Error(err)
			}
		}
	})
}

func TestClient_Start(t *testing.T) {
	// Start(address string) (err error) {

}

func TestClient_Close(t *testing.T) {
	// Close() (err error) {

}

func TestClient_IsConnected(t *testing.T) {
	// IsConnected() bool {

}

func TestClient_setIsConnected(t *testing.T) {
	// setIsConnected(t bool) {

}

func TestClient_connectHandler(t *testing.T) {
	// connectHandler(cn *gnet.Conn) {

}

func TestClient_disconnectHandler(t *testing.T) {
	// disconnectHandler(cn *gnet.Conn) {

}

func TestClient_handle(t *testing.T) {
	// handle(cn *gnet.Conn) {

}

func TestClient_handlePingMsg(t *testing.T) {
	// handlePingMsg() {

}

func TestClient_sendMessage(t *testing.T) {
	// sendMessage(msg Msg) (ok bool) {

}

func TestClient_Subscribe(t *testing.T) {
	// Subscribe(feed cipher.PubKey) (ok bool) {

}

func TestClient_Unsubscribe(t *testing.T) {
	// Unsubscribe(feed cipher.PubKey) (ok bool) {

}

func TestClient_Feeds(t *testing.T) {
	// Feeds() (feeds []cipher.PubKey) {

}

func TestClient_Container(t *testing.T) {
	// Container() *Container {

}
