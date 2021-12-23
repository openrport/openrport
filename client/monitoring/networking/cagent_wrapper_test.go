package networking

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCAgentNetWatcher(t *testing.T) {
	netWatcher := NewCAgentNetWatcher()
	assert.NotNil(t, netWatcher)
	excludeRegexSlice := netWatcher.InterfaceExcludeRegexCompiled()
	assert.NotNil(t, excludeRegexSlice)
	assert.Equal(t, 5, len(excludeRegexSlice))
}
