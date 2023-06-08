package rportplus

import (
	"context"
	"errors"
	"fmt"
	"plugin"
	"sync"

	alertingcap "github.com/realvnc-labs/rport/plus/capabilities/alerting"
	"github.com/realvnc-labs/rport/plus/capabilities/extendedpermission"
	licensecap "github.com/realvnc-labs/rport/plus/capabilities/license"
	"github.com/realvnc-labs/rport/plus/capabilities/oauth"
	"github.com/realvnc-labs/rport/plus/capabilities/status"
	"github.com/realvnc-labs/rport/plus/license"
	"github.com/realvnc-labs/rport/plus/loader"
	"github.com/realvnc-labs/rport/plus/validator"
	"github.com/realvnc-labs/rport/share/files"
	"github.com/realvnc-labs/rport/share/logger"
)

const (
	PlusOAuthCapability              = "plus-oauth"
	PlusStatusCapability             = "plus-status"
	PlusLicenseCapability            = "plus-license"
	PlusExtendedPermissionCapability = "plus-extendedpermission"
	PlusAlertingCapability           = "plus-alerting"
)

var (
	ErrPlusNotAvailable       = errors.New("rport-plus not enabled/available")
	ErrCapabilityNotAvailable = func(capName string) error { return fmt.Errorf("rport-plus capability (%s) not available", capName) }
)

// Capability is used to track loaded info about a plugin capability. See the
// corresponding individual plugin Capability structs.
type Capability interface {
	GetInitFuncName() (name string)
	InitProvider(sym plugin.Symbol)
	GetConfigValidator() (v validator.Validator)
}

// Manager defines the plus manager behavior
type Manager interface {
	InitPlusManager(cfg *PlusConfig, pluginLoader loader.Loader, logger *logger.Logger)

	// General registry type funcs
	RegisterCapability(capName string, newCap Capability) (cap Capability, err error)
	IsEnabledCapability(capName string) (isEnabled bool)

	// Access specific capabilities
	GetOAuthCapabilityEx() (capEx oauth.CapabilityEx)
	GetStatusCapabilityEx() (capEx status.CapabilityEx)
	GetExtendedPermissionCapabilityEx() (capEx extendedpermission.CapabilityEx)
	GetLicenseCapabilityEx() (capEx licensecap.CapabilityEx)
	GetAlertingCapabilityEx() (capEx alertingcap.CapabilityEx)

	// Access config validation
	GetConfigValidator(capName string) (v validator.Validator)
}

// ManagerProvider contains a map of all available capabilities and the overall
// plugin config. The manager is thread safe for reads but not initialization.
type ManagerProvider struct {
	Config       *PlusConfig
	pluginLoader loader.Loader
	logger       *logger.Logger

	mu   sync.RWMutex
	caps map[string]Capability
}

// NewPlusManager checks the plugin exists at the specified path, allocates a new
// plus manager and initializes it
func NewPlusManager(ctx context.Context, cfg *PlusConfig, pluginLoader loader.Loader, l *logger.Logger, filesAPI files.FileAPI) (pm Manager, err error) {
	if pluginLoader == nil {
		pluginLoader = loader.New()
	}

	if filesAPI != nil {
		pluginPath := cfg.PluginConfig.PluginPath
		exists, err := filesAPI.Exist(pluginPath)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("plugin not found at path \"%s\"", pluginPath)
		}

		startFn, err := pluginLoader.LoadSymbol(pluginPath, "StartPluginEx")
		if err != nil {
			return nil, err
		}

		if startFn != nil {
			err = startFn.(func(ctx context.Context, cfg *license.Config, l *logger.Logger) (err error))(ctx, cfg.LicenseConfig, l)
			if err != nil {
				return nil, err
			}
		}
	}

	pm = &ManagerProvider{}
	pm.InitPlusManager(cfg, pluginLoader, l)
	return pm, nil
}

// InitPlusManager initializes a plus manager
func (pm *ManagerProvider) InitPlusManager(cfg *PlusConfig, pluginLoader loader.Loader, logger *logger.Logger) {
	pm.Config = cfg
	pm.pluginLoader = pluginLoader
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
		pluginPath := pm.Config.PluginConfig.PluginPath
		initFn, err := pm.pluginLoader.LoadSymbol(pluginPath, newCap.GetInitFuncName())
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
			return nil
		}
		capEx = cap.GetStatusCapabilityEx()
		return capEx
	}

	return nil
}

// GetExtendedPermissionCapabilityEx returns a cast version of the Plus Extended Permission capability
func (pm *ManagerProvider) GetExtendedPermissionCapabilityEx() (capEx extendedpermission.CapabilityEx) {
	capEntry := pm.getCap(PlusExtendedPermissionCapability)
	if capEntry != nil {
		cap, ok := capEntry.(*extendedpermission.Capability)
		if !ok {
			return nil
		}
		capEx = cap.GetExtendedPermissionCapabilityEx()
		return capEx
	}

	return nil
}

// GetLicenseCapabilityEx returns a cast version of the Plus License capability
func (pm *ManagerProvider) GetLicenseCapabilityEx() (capEx licensecap.CapabilityEx) {
	capEntry := pm.getCap(PlusLicenseCapability)
	if capEntry != nil {
		cap, ok := capEntry.(*licensecap.Capability)
		if !ok {
			return nil
		}
		capEx = cap.GetLicenseCapabilityEx()
		return capEx
	}

	return nil
}

// GetAlertingCapabilityEx returns a cast version of the Plus Alerting capability
func (pm *ManagerProvider) GetAlertingCapabilityEx() (capEx alertingcap.CapabilityEx) {
	capEntry := pm.getCap(PlusAlertingCapability)
	if capEntry != nil {
		cap, ok := capEntry.(*alertingcap.Capability)
		if !ok {
			return nil
		}
		capEx = cap.GetAlertingCapabilityEx()
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
