package rportplus_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	rportplus "github.com/cloudradar-monitoring/rport/plus"
	chserver "github.com/cloudradar-monitoring/rport/server"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

const (
	defaultPluginPath = "../../rport-plus/rport-plus.so"
)

var defaultValidMinServerConfig = chserver.ServerConfig{
	URL:          []string{"http://localhost/"},
	DataDir:      "./",
	Auth:         "abc:def",
	UsedPortsRaw: []string{"10-20"},
}

var plusLog = logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

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
	config := &chserver.Config{
		Server: defaultValidMinServerConfig,
		PlusConfig: &rportplus.PlusConfig{
			PluginPath: "./invalid/path",
		},
	}

	fs := &mockFileSystem{
		ShouldNotExist: true,
	}

	_, err := rportplus.NewPlusManager(config.PlusConfig, plusLog, fs)
	assert.EqualError(t, err, `plugin not found at path "./invalid/path"`)
}

func TestShouldNotErrorWhenCorrectPluginPath(t *testing.T) {
	config := &chserver.Config{
		Server: defaultValidMinServerConfig,
		PlusConfig: &rportplus.PlusConfig{
			PluginPath: defaultPluginPath,
		},
	}

	fs := &mockFileSystem{}
	_, err := rportplus.NewPlusManager(config.PlusConfig, plusLog, fs)

	assert.NoError(t, err)
	assert.Equal(t, config.PlusConfig.PluginPath, fs.CheckedPath)
}
