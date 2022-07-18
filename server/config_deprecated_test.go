package chserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerConfigReplaceDeprecated(t *testing.T) {
	cfg := ServerConfig{
		CleanupLostClients: true,
	}
	new, repl, err := ServerConfigReplaceDeprecated(cfg)
	assert.NoError(t, err)
	assert.Equal(t, true, new.PurgeDisconnectedClients)
	expected := map[string]string{"cleanup_lost_clients": "purge_disconnected_clients"}
	assert.Equal(t, repl, expected)
}
