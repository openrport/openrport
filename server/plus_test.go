package chserver

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rportplus "github.com/cloudradar-monitoring/rport/plus"
	"github.com/cloudradar-monitoring/rport/plus/capabilities/oauth"
	"github.com/cloudradar-monitoring/rport/plus/validator"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

const (
	defaultPluginPath = "../rport-plus/rport-plus.so"
)

type mockValidator struct{}

func (m *mockValidator) ValidateConfig() (err error) {
	return nil
}

type plusManagerMock struct {
	CapabilityCount int
	Caps            map[string]rportplus.Capability

	rportplus.ManagerProvider
}

func (pm *plusManagerMock) RegisterCapability(capName string, newCap rportplus.Capability) (cap rportplus.Capability, err error) {
	pm.CapabilityCount++
	if pm.Caps == nil {
		pm.Caps = make(map[string]rportplus.Capability, 0)
	}
	pm.Caps[capName] = newCap
	return newCap, nil
}

func (pm *plusManagerMock) GetConfigValidator(capName string) (v validator.Validator) {
	return &mockValidator{}
}

// Checks that the expected plugins are loaded using using mock interfaces.
// Does not require a working plugin.
func TestShouldRegisterPlusCapabilities(t *testing.T) {
	plusLog := logger.NewLogger("rport-plus", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	config := &Config{
		Server: defaultValidMinServerConfig,
		PlusConfig: &rportplus.PlusConfig{
			PluginPath: defaultPluginPath,
		},
		OAuthConfig: &oauth.Config{
			Provider: oauth.GitHubOAuthProvider,
		},
	}

	plus := &plusManagerMock{}
	plus.InitPlusManager(config.PlusConfig, plusLog)
	require.NotNil(t, plus)

	// register the capabilities with the plus manager partial mock. the purpose
	// of the test is to check whether the expected capabilities are being
	// requested, not to test the plugin manager.
	err := RegisterPlusCapabilities(plus, config, testLog)
	assert.NoError(t, err)

	// this check will flag when additional capabilities have been registered but the test
	// not updated
	assert.Equal(t, 2, plus.CapabilityCount)

	// additional capabilities should be checked here to see that the server has
	// registered them
	assert.NotNil(t, plus.Caps[rportplus.PlusOAuthCapability])
	assert.NotNil(t, plus.Caps[rportplus.PlusStatusCapability])
}
