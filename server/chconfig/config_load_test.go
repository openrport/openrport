package chconfig

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chshare "github.com/realvnc-labs/rport/share"
)

func TestLoadingExampleConf(t *testing.T) {
	var (
		viperCfg *viper.Viper
		cfg      = &Config{}
	)
	viperCfg = viper.New()
	viperCfg.SetConfigType("toml")
	viperCfg.SetConfigFile("../../rportd.example.conf")
	path, err := os.Getwd()
	require.NoError(t, err)
	t.Logf("Testing example config %s.rportd.example.conf\n", path)
	err = chshare.DecodeViperConfig(viperCfg, cfg, nil)
	require.NoError(t, err)
	assert.Equal(t, "<YOUR_SEED>", cfg.Server.KeySeed)
	assert.Equal(t, "clientAuth1:1234", cfg.Server.Auth)
	assert.Equal(t, "/var/lib/rport", cfg.Server.DataDir)
}

const (
	sampleCfg = `
[plus-plugin]
	plugin_path = "/usr/local/lib/rport/rport-plus.so"
[plus-license]
  id = "<your-license-id>"
  key = "<your-license-key>"
  proxy_url = "http://user:pass@proxy.example.com:8080"
[plus-oauth]
	provider = "github"
`
)

func TestLoadingPlusConf(t *testing.T) {
	var (
		viperCfg *viper.Viper
		cfg      = &Config{}
	)

	viperCfg = viper.New()
	viperCfg.SetConfigType("toml")

	cfgReader := strings.NewReader(sampleCfg)

	err := chshare.DecodeViperConfig(viperCfg, cfg, cfgReader)
	require.NoError(t, err)

	assert.Equal(t, "/usr/local/lib/rport/rport-plus.so", cfg.PlusConfig.PluginConfig.PluginPath)
	assert.NotEmpty(t, "github", cfg.PlusConfig.OAuthConfig.Provider)

	assert.Equal(t, "<your-license-id>", cfg.PlusConfig.LicenseConfig.ID)
	assert.Equal(t, "<your-license-key>", cfg.PlusConfig.LicenseConfig.Key)
	assert.Equal(t, "http://user:pass@proxy.example.com:8080", cfg.PlusConfig.LicenseConfig.ProxyURL)
}
