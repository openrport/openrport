package chserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rportplus "github.com/cloudradar-monitoring/rport/rport-plus"
	"github.com/cloudradar-monitoring/rport/rport-plus/capabilities/oauth"
	"github.com/cloudradar-monitoring/rport/rport-plus/validator"
	"github.com/cloudradar-monitoring/rport/share/files"
)

const (
	defaultPluginPath = "../../rport-plus/plugin.so"
)

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
	config := &Config{
		Server:     defaultValidMinServerConfig,
		PlusConfig: &rportplus.PlusConfig{},
	}

	fs := &MockFileSystem{
		ShouldNotExist: true,
	}
	_, err := NewServer(config, &ServerOpts{FilesAPI: fs})
	assert.EqualError(t, err, "plugin not found at path \"\"")
}

// Requires a working plugin
func TestShouldLoadPluginWhenPath(t *testing.T) {
	config := &Config{
		Server: defaultValidMinServerConfig,
		PlusConfig: &rportplus.PlusConfig{
			PluginPath: defaultPluginPath,
		},
	}

	fs := &MockFileSystem{}
	_, err := NewServer(config, &ServerOpts{FilesAPI: fs})

	assert.NoError(t, err)
	assert.Equal(t, config.PlusConfig.PluginPath, fs.ExistPath)
}

type MockValidator struct{}

func (m *MockValidator) ValidateConfig() (err error) {
	return nil
}

type PlusManagerMock struct {
	rportplus.ManagerProvider
}

func (pm *PlusManagerMock) RegisterCapability(capName string, newCap rportplus.Capability) (cap rportplus.Capability, err error) {
	pm.SetCapability(capName, newCap)
	return newCap, nil
}

func (pm *PlusManagerMock) GetConfigValidator(capName string) (v validator.Validator) {
	return &MockValidator{}
}

// Does not require a working plugin. Checks that the server tries to load the expected
// plugins using mock interfaces.
func TestShouldEnablePlusIfLicensed(t *testing.T) {
	config := &Config{
		Server: defaultValidMinServerConfig,
		PlusConfig: &rportplus.PlusConfig{
			PluginPath: defaultPluginPath,
		},
		OAuthConfig: &oauth.Config{
			Provider: oauth.GitHubOAuthProvider,
		},
	}

	fs := &MockFileSystem{}

	pm := &PlusManagerMock{}
	pm.InitPlusManager(config.PlusConfig, fs)

	// create a new server with the plus manager partial mock. this will allow us to bypass
	// loading of the actual plugin and focus on the server behavior testing.
	s, err := NewServer(config, &ServerOpts{
		FilesAPI:    fs,
		PlusManager: pm,
	})
	require.NoError(t, err)

	err = s.EnablePlusIfLicensed()
	assert.NoError(t, err)

	plus := s.plusManager
	require.NotNil(t, plus)

	// this check will flag when additional capabilities have been registered but the test
	// not updated
	count := plus.GetTotalCapabilities()
	assert.Equal(t, 2, count)

	// additional capabilities should be checked here to see that the server is trying to load them
	assert.True(t, plus.HasCapabilityEnabled(rportplus.PlusOAuthCapability))
	assert.True(t, plus.HasCapabilityEnabled(rportplus.PlusVersionCapability))
}
