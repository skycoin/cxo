package node

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewClient(t *testing.T) {
	defer clean()
	t.Run("memory db", func(t *testing.T) {
		clean()
		conf := newClientConfig() // in-memory
		c, err := NewClient(conf, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if _, err := os.Stat(conf.DataDir); err == nil {
			t.Error("unexpected data dir:", conf.DataDir)
			if _, err = os.Stat(conf.DBPath); err == nil {
				t.Error("unexpected db file:", conf.DBPath)
			} else if !os.IsNotExist(err) {
				t.Error("unexpected error")
			}
		} else if !os.IsNotExist(err) {
			t.Error("unexpected error:", err)
		}
	})
	t.Run("data dir", func(t *testing.T) {
		clean()
		conf := newClientConfig()
		conf.InMemoryDB = false
		c, err := NewClient(conf, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if _, err := os.Stat(conf.DataDir); err != nil {
			t.Error(err)
		} else {
			if _, err := os.Stat(conf.DBPath); err != nil {
				t.Error(err)
			}
		}
	})
	t.Run("dbpath", func(t *testing.T) {
		clean()
		conf := newClientConfig()
		conf.InMemoryDB = false
		conf.DBPath = filepath.Join(testDataDir, "another.db")
		c, err := NewClient(conf, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if _, err := os.Stat(conf.DataDir); err != nil {
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
