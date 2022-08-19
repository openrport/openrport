package rportplus_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	rportplus "github.com/cloudradar-monitoring/rport/rport-plus"
	chserver "github.com/cloudradar-monitoring/rport/server"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

const (
	defaultPluginPath = "../../rport-plus/plugin.so"
)

var defaultValidMinServerConfig = chserver.ServerConfig{
	URL:          []string{"http://localhost/"},
	DataDir:      "./",
	Auth:         "abc:def",
	UsedPortsRaw: []string{"10-20"},
}

var plusLog = logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

type MockFileSystem struct {
	*files.FileSystem

	ShouldNotExist bool
	ExistPath      string
}

func (m *MockFileSystem) MakeDirAll(dir string) error {
	return nil
}

func (m *MockFileSystem) Exist(path string) (bool, error) {
	m.ExistPath = path
	if m.ShouldNotExist {
		return false, nil
	}
	return true, nil
}

func TestShouldFailToLoadPluginWhenNoPath(t *testing.T) {
	config := &chserver.Config{
		Server:     defaultValidMinServerConfig,
		PlusConfig: &rportplus.PlusConfig{},
	}

	fs := &MockFileSystem{
		ShouldNotExist: true,
	}

	_, err := rportplus.NewPlusManager(config.PlusConfig, plusLog, fs)
	assert.EqualError(t, err, "plugin not found at path \"\"")
}

// Requires a working plugin
func TestShouldLoadPluginWhenPath(t *testing.T) {
	config := &chserver.Config{
		Server: defaultValidMinServerConfig,
		PlusConfig: &rportplus.PlusConfig{
			PluginPath: defaultPluginPath,
		},
	}

	fs := &MockFileSystem{}
	_, err := rportplus.NewPlusManager(config.PlusConfig, plusLog, fs)

	assert.NoError(t, err)
	assert.Equal(t, config.PlusConfig.PluginPath, fs.ExistPath)
}
