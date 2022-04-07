package chserver

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func TestLoadingExampleConf(t *testing.T) {
	var (
		viperCfg *viper.Viper
		cfg      = &Config{}
	)
	viperCfg = viper.New()
	viperCfg.SetConfigType("toml")
	viperCfg.SetConfigFile("../rportd.example.conf")
	path, _ := os.Getwd()
	t.Logf("Testing example config %s.rportd.example.conf\n", path)
	err := chshare.DecodeViperConfig(viperCfg, cfg)
	require.NoError(t, err)
	assert.Equal(t, "5448e69530b4b97fb510f96ff1550500b093", cfg.Server.KeySeed)
	assert.Equal(t, "clientAuth1:1234", cfg.Server.Auth)
	assert.Equal(t, "/var/lib/rport", cfg.Server.DataDir)
}
