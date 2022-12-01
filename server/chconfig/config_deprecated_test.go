package chconfig

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

type SampleConfig struct {
	Current    string
	Deprecated string `replaced_by:"Current"`
}

func TestServerConfigReplaceDeprecated(t *testing.T) {
	cfg := SampleConfig{
		Deprecated: "foo",
	}
	repl, err := ConfigReplaceDeprecated(&cfg)
	require.NoError(t, err)
	assert.Equal(t, "foo", cfg.Current)
	expected := map[string]string{"deprecated": "current"}
	assert.Equal(t, repl, expected)
}
