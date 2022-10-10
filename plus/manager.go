package rportplus

import (
	"errors"
	"fmt"
	"plugin"
	"sync"

	"github.com/cloudradar-monitoring/rport/plus/capabilities/oauth"
	"github.com/cloudradar-monitoring/rport/plus/capabilities/status"
	"github.com/cloudradar-monitoring/rport/plus/loader"
	"github.com/cloudradar-monitoring/rport/plus/validator"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

const (
	PlusOAuthCapability  = "plus-oauth"
	PlusStatusCapability = "plus-status"
)

var (
	ErrPlusNotAvailable       = errors.New("rport-plus not enabled/available")
	ErrCapabilityNotAvailable = func(capName string) error { return fmt.Errorf("rport-plus capability (%s) not available", capName) }
)

// PlusConfig contains the overall config for the rport-plus plugin. note that
// each capability should have it's own config section in the config file.
type PlusConfig struct {
	PluginPath string `mapstructure:"plugin_path"`
}

// Capability is used to track loaded info about a plugin capability. See the
// corresponding individual plugin Capability structs.
type Capability interface {
	GetInitFuncName() (name string)
	InitProvider(sym plugin.Symbol)
	GetConfigValidator() (v validator.Validator)
}

// Manager defines the plus manager behavior
type Manager interface {
	InitPlusManager(cfg *PlusConfig, logger *logger.Logger)

	// General registry type funcs
	RegisterCapability(capName string, newCap Capability) (cap Capability, err error)
	IsEnabledCapability(capName string) (isEnabled bool)

	// Access specific capabilities
	GetOAuthCapabilityEx() (capEx oauth.CapabilityEx)
	GetStatusCapabilityEx() (capEx status.CapabilityEx)

	// Access config validation
	GetConfigValidator(capName string) (v validator.Validator)
}

// ManagerProvider contains a map of all available capabilities and the overall
// plugin config. The manager is thread safe for reads but not initialization.
type ManagerProvider struct {
	Config *PlusConfig
	logger *logger.Logger

	mu   sync.RWMutex
	caps map[string]Capability
}

// NewPlusManager checks the plugin exists at the specified path, allocates a new
// plus manager and initializes it
func NewPlusManager(cfg *PlusConfig, logger *logger.Logger, filesAPI files.FileAPI) (pm Manager, err error) {
	if filesAPI != nil {
		exists, err := filesAPI.Exist(cfg.PluginPath)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("plugin not found at path \"%s\"", cfg.PluginPath)
		}
	}

	pm = &ManagerProvider{}
	pm.InitPlusManager(cfg, logger)
	return pm, nil
}

// InitPlusManager initializes a plus manager
func (pm *ManagerProvider) InitPlusManager(cfg *PlusConfig, logger *logger.Logger) {
	pm.Config = cfg
	pm.caps = make(map[string]Capability, 0)
	pm.logger = logger
}

// RegisterCapability adds a new plugin capability component, including loading
// the relevant init func for the capability from the plugin and initializing
// the capability (via InitProvider)
func (pm *ManagerProvider) RegisterCapability(capName string, newCap Capability) (cap Capability, err error) {
	if pm.Config == nil {
		return nil, ErrPlusNotAvailable
	}

	pm.setCap(capName, newCap)

	initFuncName := newCap.GetInitFuncName()
	if initFuncName != "" {
		// an init func name indicates that the provider should be initialized using the plugin
		initFn, err := loader.LoadSymbol(pm.Config.PluginPath, newCap.GetInitFuncName())
		if err != nil {
			return nil, err
		}
		newCap.InitProvider(initFn)
	} else {
		// empty init func name indicates that the provider can be initialized locally
		newCap.InitProvider(nil)
	}

	return newCap, err
}

// IsEnabledCapability returns whether the specified capability is enabled
func (pm *ManagerProvider) IsEnabledCapability(capName string) (isEnabled bool) {
	// at the moment, if present it is enabled
	capEntry := pm.getCap(capName)
	return capEntry != nil
}

// GetOAuthCapability returns a cast version of the OAuth capability
func (pm *ManagerProvider) GetOAuthCapabilityEx() (capEx oauth.CapabilityEx) {
	capEntry := pm.getCap(PlusOAuthCapability)
	if capEntry != nil {
		cap, ok := capEntry.(*oauth.Capability)
		if !ok {
			// TODO: consider returning an error here
			return nil
		}
		capEx = cap.GetOAuthCapabilityEx()
		return capEx
	}

	return nil
}

// GetStatusCapabilityEx returns a cast version of the Plus Status capability
func (pm *ManagerProvider) GetStatusCapabilityEx() (capEx status.CapabilityEx) {
	capEntry := pm.getCap(PlusStatusCapability)
	if capEntry != nil {
		cap, ok := capEntry.(*status.Capability)
		if !ok {
			// TODO: consider returning an error here
			return nil
		}
		capEx = cap.GetStatusCapabilityEx()
		return capEx
	}

	return nil
}

// GetConfigValidator gets a validator interface that can be invoked to validate
// the capability config
func (pm *ManagerProvider) GetConfigValidator(capName string) (v validator.Validator) {
	capEntry := pm.getCap(capName)
	if capEntry != nil {
		v = capEntry.GetConfigValidator()
	}
	return v
}

// setCap sets the capability in the capability map
func (pm *ManagerProvider) setCap(capName string, cap Capability) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.caps[capName] = cap
}

// getCap sets the raw capability in the capability map without casting
func (pm *ManagerProvider) getCap(capName string) (cap Capability) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.caps[capName]
}

// GetTotalCapabilities is currently only used for testing
func (pm *ManagerProvider) GetTotalCapabilities() (total int) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return len(pm.caps)
}
