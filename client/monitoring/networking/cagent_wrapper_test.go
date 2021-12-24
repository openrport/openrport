package networking

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCAgentNetWatcher(t *testing.T) {
	netWatcher := NewCAgentNetWatcher()
	assert.NotNil(t, netWatcher)
}
