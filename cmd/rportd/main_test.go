package main

import (
	"testing"

	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldLoadConfigFromFile(t *testing.T) {
	mLog := logger.NewMemLogger()
	bindPFlags()

	*cfgPath = "./test.conf"

	err := decodeAndValidateConfig(&mLog)
	require.NoError(t, err)

	// simple checks that config was loaded

	assert.NotEmpty(t, cfg.Server.ListenAddress)
	assert.NotEmpty(t, cfg.Server.DataDir)

	assert.NotEmpty(t, cfg.API.Address)
	assert.NotEmpty(t, cfg.SMTP.Server)

	assert.NotNil(t, cfg.PlusConfig.PluginConfig)
	assert.NotNil(t, cfg.PlusConfig.OAuthConfig)
}
