package nodeManager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManagerConfig(t *testing.T) {
	newConfig := NewManagerConfig()

	assert.Equal(t, defaultIP, newConfig.IP, "they should be equal")
	assert.Equal(t, defaultPort, newConfig.Port, "they should be equal")
	assert.Equal(t, defaultMaxNodes, newConfig.MaxNodes, "they should be equal")
	assert.Equal(t, defaultMaxConnections, newConfig.MaxConnections, "they should be equal")
	assert.Equal(t, defaultMaxMessageLength, newConfig.MaxMessageLength, "they should be equal")

	err := newConfig.validate()
	if err != nil {
		t.Error(err)
	}

	assert.NotNil(t, newConfig.ip, "should NOT be nil")
}

func TestNewManager(t *testing.T) {
	newConfig := NewManagerConfig()

	newManager, err := NewManager(newConfig)
	if err != nil {
		t.Error(err)
	}

	assert.NotNil(t, newManager.nodes, "should NOT be nil")
	assert.NotNil(t, newManager.downstreamCallbacks, "should NOT be nil")
	assert.NotNil(t, newManager.upstreamCallbacks, "should NOT be nil")
	assert.NotNil(t, newManager.mu, "should NOT be nil")

}
