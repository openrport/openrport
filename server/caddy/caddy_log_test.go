package caddy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldGetCaddyLogLevel(t *testing.T) {
	caddyLogStr := `{"level":"debug","ts":1672119742.1765668,"logger":"http","msg":"starting server loop","address":"[::]:443","tls":true,"http3":true}`

	level, err := extractCaddyLogLevel(caddyLogStr)
	require.NoError(t, err)

	assert.Equal(t, "debug", level)
}
