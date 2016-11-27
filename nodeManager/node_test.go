package nodeManager

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/stretchr/testify/assert"
)

func TestNewNode(t *testing.T) {
	newConfig := NewManagerConfig()

	newManager, err := NewManager(newConfig)
	if err != nil {
		t.Error(err)
	}

	newNode := newManager.NewNode()

	assert.False(t, newNode.quitting, "should be false")

	assert.NotNil(t, newNode.config, "should NOT be nil")

	assert.NotNil(t, newNode.pubKey, "should NOT be nil")
	assert.NotNil(t, newNode.secKey, "should NOT be nil")
	assert.NotNil(t, newNode.mu, "should NOT be nil")

	assert.NotNil(t, newNode.upstream, "should NOT be nil")
	assert.NotNil(t, newNode.upstream.callbacks, "should NOT be nil")
	assert.NotNil(t, newNode.upstream.subscriptions, "should NOT be nil")

	assert.NotNil(t, newNode.downstream, "should NOT be nil")
	assert.NotNil(t, newNode.downstream.callbacks, "should NOT be nil")
	assert.NotNil(t, newNode.downstream.subscribers, "should NOT be nil")
}

func TestNewNodeFromKeyPair(t *testing.T) {
	newConfig := NewManagerConfig()

	newManager, err := NewManager(newConfig)
	if err != nil {
		t.Error(err)
	}

	pubKey, secKey := cipher.GenerateKeyPair() //generate a keypair

	newNode, err := newManager.NewNodeFromSecKey(secKey)
	if err != nil {
		t.Error(err)
	}

	assert.False(t, newNode.quitting, "should be false")

	assert.NotNil(t, newNode.config, "should NOT be nil")

	assert.NotNil(t, newNode.pubKey, "should NOT be nil")
	assert.NotNil(t, newNode.secKey, "should NOT be nil")
	assert.NotNil(t, newNode.mu, "should NOT be nil")

	assert.NotNil(t, newNode.upstream, "should NOT be nil")
	assert.NotNil(t, newNode.upstream.callbacks, "should NOT be nil")
	assert.NotNil(t, newNode.upstream.subscriptions, "should NOT be nil")

	assert.NotNil(t, newNode.downstream, "should NOT be nil")
	assert.NotNil(t, newNode.downstream.callbacks, "should NOT be nil")
	assert.NotNil(t, newNode.downstream.subscribers, "should NOT be nil")

	assert.EqualValues(t, &pubKey, newNode.pubKey, "they should be equal")
	assert.EqualValues(t, &secKey, newNode.secKey, "they should be equal")
}
//
//func TestNewNodeFromKeyPair_withWrongKeys(t *testing.T) {
//	newConfig := NewManagerConfig()
//
//	newManager, err := NewManager(newConfig)
//	if err != nil {
//		t.Error(err)
//	}
//
//	_, secKey := cipher.GenerateKeyPair() //generate a keypair
//
//	_, err = newManager.NewNodeFromSecKey(secKey)
//
//	assert.True(t, err != nil, "should be true")
//}
//
//func TestNewNode_validate(t *testing.T) {
//	newConfig := NewManagerConfig()
//
//	newManager, err := NewManager(newConfig)
//	if err != nil {
//		t.Error(err)
//	}
//
//	newNode := newManager.NewNode()
//
//	err = newNode.validate()
//	assert.True(t, err == nil, "should be no error")
//
//	newNode.pubKey = &cipher.PubKey{}
//	err = newNode.validate()
//	assert.False(t, err == nil, "should be an error")
//
//	// reset keys
//	newNode = newManager.NewNode()
//
//	newNode.secKey = &cipher.SecKey{}
//	err = newNode.validate()
//	assert.False(t, err == nil, "should be an error")
//}

func TestAddNode(t *testing.T) {
	newConfig := NewManagerConfig()

	newManager, err := NewManager(newConfig)
	if err != nil {
		t.Error(err)
	}

	newNode := newManager.NewNode()

	err = newManager.AddNode(newNode)
	assert.True(t, err == nil, "should be no error")

	assert.EqualValues(t, newNode.pubKey, newManager.nodes[*newNode.pubKey].pubKey, "they should be equal")
	assert.EqualValues(t, newNode.secKey, newManager.nodes[*newNode.pubKey].secKey, "they should be equal")

}

func TestNode_PubKey(t *testing.T) {
	newConfig := NewManagerConfig()

	newManager, err := NewManager(newConfig)
	if err != nil {
		t.Error(err)
	}

	newNode := newManager.NewNode()

	assert.EqualValues(t, *newNode.pubKey, newNode.PubKey(), "they should be equal")
}
