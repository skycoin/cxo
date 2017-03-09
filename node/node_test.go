package node

import (
	"testing"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

func TestNode_connecting(t *testing.T) {
	conf := NewConfig()
	db := data.NewDB()
	so := skyobject.NewContainer(db)
	n1, n2 := NewNode(conf, db, so), NewNode(conf, db, so)
	if err := n1.Start(); err != nil {
		t.Error(err)
		return
	}
	defer n1.Close()
	if err := n2.Start(); err != nil {
		t.Error(err)
		return
	}
	defer n2.Close()
	if address, err := n1.Info(); err != nil {
		t.Error(err)
		return
	} else {
		if err := n2.Connect(address); err != nil {
			t.Error(err)
		}
	}
}

//
// TODO: root replication test
//
