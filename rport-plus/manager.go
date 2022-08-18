package rportplus

import (
	"errors"
	"fmt"
	"plugin"
	"sync"

	"github.com/cloudradar-monitoring/rport/rport-plus/capabilities/oauth"
	"github.com/cloudradar-monitoring/rport/rport-plus/capabilities/version"
	"github.com/cloudradar-monitoring/rport/rport-plus/loader"
	"github.com/cloudradar-monitoring/rport/rport-plus/validator"
	"github.com/cloudradar-monitoring/rport/share/files"
)

// use var to allow a mock capability string to be patched in
var (
	PlusOAuthCapability   = "oauth_provider"
	PlusVersionCapability = "plugin_version"
)

var (
	ErrPlusNotAvailable       = errors.New("rport-plus not enabled/available")
	ErrCapabilityNotAvailable = func(capName string) error { return fmt.Errorf("rport-plus capability (%s) not available", capName) }
)

// PlugConfig contains the overall config for the rport-plus plugin. note that
// each capability should have it's own config section in the config file.
type PlusConfig struct {
	PluginPath string `mapstructure:"plugin_path"`
}

// Capability is used to track loaded info about a plugin capability. See the
// corresponding individual plugin Capability structs.
type Capability interface {
	GetInitFuncName() (name string)
	SetProvider(sym plugin.Symbol)
	GetConfigValidator() (v validator.Validator)
}

// Manager defines the plus manager behavior
type Manager interface {
	InitPlusManager(cfg *PlusConfig, fileAPI files.FileAPI)
	RegisterCapability(capName string, newCap Capability) (cap Capability, err error)
	SetCapability(capName string, cap Capability)
	HasCapabilityEnabled(capName string) (isEnabled bool)
	GetOAuthCapabilityEx() (capEx oauth.CapabilityEx)
	GetVersionCapabilityEx() (capEx version.CapabilityEx)
	GetConfigValidator(capName string) (v validator.Validator)
	GetTotalCapabilities() (total int)
}

// ManagerProvider contains a map of all available capabilities and the overall
// plugin config. The manager is thread safe for reads but not initialization.
type ManagerProvider struct {
	Config *PlusConfig

	caps map[string]Capability

	mu sync.RWMutex
}

// NewPlusManager checks the plugin exists at the specified path, allocates a new
// plus manager and initializes it
func NewPlusManager(cfg *PlusConfig, filesAPI files.FileAPI) (pm Manager, err error) {
	if filesAPI != nil {
		exists, err := filesAPI.Exist(cfg.PluginPath)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("plugin not found at path \"%s\"", cfg.PluginPath)
		}
	}

	pmp := &ManagerProvider{}
	pmp.InitPlusManager(cfg, filesAPI)
	return pmp, nil
}

// InitPlusManager initializes a plus manager
func (pm *ManagerProvider) InitPlusManager(cfg *PlusConfig, fileAPI files.FileAPI) {
	pm.Config = cfg
	pm.caps = make(map[string]Capability, 0)
}

// RegisterCapability adds a new plugin capability component, including loading
// the relevant func symbols for capability from the plugin and calling the
// capability config validation
func (pm *ManagerProvider) RegisterCapability(capName string, newCap Capability) (cap Capability, err error) {
	if pm.Config == nil {
		return nil, ErrPlusNotAvailable
	}

	pm.SetCapability(capName, newCap)

	sym, err := pm.LoadInitFunc(pm.Config.PluginPath, newCap.GetInitFuncName())
	if err != nil {
		return nil, err
	}

	newCap.SetProvider(sym)

	return newCap, err
}

func (pm *ManagerProvider) LoadInitFunc(pluginPath string, capName string) (sym plugin.Symbol, err error) {
	sym, _ = loader.LoadSymbol(pluginPath, capName)
	if err != nil {
		return nil, err
	}
	return sym, nil
}

// SetCapability sets the capability in the capability map
func (pm *ManagerProvider) SetCapability(capName string, cap Capability) {
	pm.caps[capName] = cap
}

// HasCapabilityEnabled returns whether the specified capability is present/enabled
// or not
func (pm *ManagerProvider) HasCapabilityEnabled(capName string) (isEnabled bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// at the moment, if present it is enabled
	cap := pm.caps[capName]
	return cap != nil
}

// GetOAuthCapability return a cast version of the OAuth capability
func (pm *ManagerProvider) GetOAuthCapabilityEx() (capEx oauth.CapabilityEx) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	capEntry := pm.caps[PlusOAuthCapability]
	if capEntry != nil {
		cap := capEntry.(*oauth.Capability)
		capEx = cap.GetOAuthCapabilityEx()
		return capEx
	}

	return nil
}

// GetVersionCapability returns a cast version of the Version capability
func (pm *ManagerProvider) GetVersionCapabilityEx() (capEx version.CapabilityEx) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	capEntry := pm.caps[PlusVersionCapability]
	if capEntry != nil {
		cap := capEntry.(*version.Capability)
		capEx = cap.GetVersionCapabilityEx()
		return capEx
	}

	return nil
}

// GetConfigValidator gets a validator interface that can be invoked to
// validate the capability config
func (pm *ManagerProvider) GetConfigValidator(capName string) (v validator.Validator) {
	capEntry := pm.caps[capName]
	if capEntry != nil {
		v = capEntry.GetConfigValidator()
	}
	return v
}

// GetTotalCapabilities is currently primarily used for testing
func (pm *ManagerProvider) GetTotalCapabilities() (total int) {
	return len(pm.caps)
}
