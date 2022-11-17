package rportplus_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	rportplus "github.com/cloudradar-monitoring/rport/plus"
	"github.com/cloudradar-monitoring/rport/server/chconfig"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

var defaultValidMinServerConfig = chconfig.ServerConfig{
	URL:          []string{"http://localhost/"},
	DataDir:      "./",
	Auth:         "abc:def",
	UsedPortsRaw: []string{"10-20"},
}

type mockFileSystem struct {
	*files.FileSystem

	ShouldNotExist bool
	CheckedPath    string
}

func (m *mockFileSystem) MakeDirAll(dir string) error {
	return nil
}

func (m *mockFileSystem) Exist(path string) (bool, error) {
	m.CheckedPath = path
	if m.ShouldNotExist {
		return false, nil
	}
	return true, nil
}

func TestShouldErrorWhenPluginPathDoesNotExist(t *testing.T) {
	plusLog := logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	config := &chconfig.Config{
		Server: defaultValidMinServerConfig,
		PlusConfig: rportplus.PlusConfig{
			PluginConfig: &rportplus.PluginConfig{
				PluginPath: "./invalid/path",
			},
		},
	}

	fs := &mockFileSystem{
		ShouldNotExist: true,
	}

	_, err := rportplus.NewPlusManager(&config.PlusConfig, plusLog, fs)
	assert.EqualError(t, err, `plugin not found at path "./invalid/path"`)
}

func TestShouldNotErrorWhenCorrectPluginPath(t *testing.T) {
	plusLog := logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	config := &chconfig.Config{
		Server: defaultValidMinServerConfig,
		PlusConfig: rportplus.PlusConfig{
			PluginConfig: &rportplus.PluginConfig{
				PluginPath: "./invalid/path",
			},
		},
	}

	fs := &mockFileSystem{}
	_, err := rportplus.NewPlusManager(&config.PlusConfig, plusLog, fs)

	assert.NoError(t, err)
	assert.Equal(t, config.PlusConfig.PluginConfig.PluginPath, fs.CheckedPath)
}
