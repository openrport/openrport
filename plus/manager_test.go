package rportplus_test

import (
	"context"
	"os"
	"plugin"
	"testing"

	"github.com/stretchr/testify/assert"

	rportplus "github.com/cloudradar-monitoring/rport/plus"
	"github.com/cloudradar-monitoring/rport/plus/license"
	chserver "github.com/cloudradar-monitoring/rport/server"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

var defaultValidMinServerConfig = chserver.ServerConfig{
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

	config := &chserver.Config{
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

	ctx := context.Background()

	_, err := rportplus.NewPlusManager(ctx, &config.PlusConfig, nil, plusLog, fs)
	assert.EqualError(t, err, `plugin not found at path "./invalid/path"`)
}

type MockPluginLoader struct{}

func (pl *MockPluginLoader) LoadSymbol(pluginPath string, name string) (sym plugin.Symbol, err error) {
	return nil, nil
}

func TestShouldNotErrorWhenCorrectPluginPath(t *testing.T) {
	plusLog := logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	licConfig := &license.Config{
		ID:      "83c5afc7-87a7-4a3d-9889-3905ec979045",
		Key:     "6OO1STn0b0XUahz+RN6jBJ93KBuSbsKPef+SMl98NEU=",
		DataDir: ".",
	}

	config := &chserver.Config{
		Server: defaultValidMinServerConfig,
		PlusConfig: rportplus.PlusConfig{
			PluginConfig: &rportplus.PluginConfig{
				PluginPath: "./valid/path",
			},
			LicenseConfig: licConfig,
		},
	}

	ctx := context.Background()

	fs := &mockFileSystem{}
	_, err := rportplus.NewPlusManager(ctx, &config.PlusConfig, &MockPluginLoader{}, plusLog, fs)

	assert.NoError(t, err)
	assert.Equal(t, config.PlusConfig.PluginConfig.PluginPath, fs.CheckedPath)
}
