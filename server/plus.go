package chserver

import (
	"context"
	"errors"
	"fmt"

	rportplus "github.com/openrport/openrport/plus"
	alertingcap "github.com/openrport/openrport/plus/capabilities/alerting"
	"github.com/openrport/openrport/plus/capabilities/extendedpermission"
	licensecap "github.com/openrport/openrport/plus/capabilities/license"
	"github.com/openrport/openrport/plus/capabilities/oauth"
	"github.com/openrport/openrport/plus/capabilities/status"
	"github.com/openrport/openrport/plus/license"
	"github.com/openrport/openrport/server/chconfig"
	"github.com/openrport/openrport/share/files"
	"github.com/openrport/openrport/share/logger"
)

var (
	ErrPlusNotEnabled           = errors.New("rport-plus not enabled")
	ErrPlusLicenseNotConfigured = errors.New("rport-plus license not configured")
)

// EnablePlusIfAvailable will initialize a new plus manager and request registration of the desired
// capabilities
func EnablePlusIfAvailable(ctx context.Context, cfg *chconfig.Config, filesAPI files.FileAPI) (plusManager *rportplus.ManagerProvider, err error) {
	logger := logger.NewLogger("rport-plus", cfg.Logging.LogOutput, cfg.Logging.LogLevel)

	if !rportplus.IsPlusEnabled(cfg.PlusConfig) {
		logger.Infof("not enabled")
		return nil, ErrPlusNotEnabled
	}

	if rportplus.HasLicenseConfig(cfg.PlusConfig) {
		dataDir := cfg.Server.DataDir
		cfg.PlusConfig.LicenseConfig.DataDir = dataDir
	} else {
		cfg.PlusConfig.LicenseConfig = &license.Config{}
	}

	plusManager, err = rportplus.NewPlusManager(ctx, &cfg.PlusConfig, nil, logger, filesAPI)
	if err != nil {
		return nil, err
	}
	logger.Infof("plus manager initialized")

	err = RegisterPlusCapabilities(plusManager, cfg, logger)
	if err != nil {
		return nil, err
	}
	return plusManager, nil
}

// RegisterPluginCapabilitities registers the rport-plus additional capabilities.
// All plus capabilities must be added here.
func RegisterPlusCapabilities(plusManager rportplus.Manager, cfg *chconfig.Config, logger *logger.Logger) (err error) {
	if rportplus.IsPlusOAuthEnabled(cfg.PlusConfig) {
		_, err := plusManager.RegisterCapability(rportplus.PlusOAuthCapability, &oauth.Capability{
			Config: cfg.PlusConfig.OAuthConfig,
			Logger: logger,
		})
		if err != nil {
			return fmt.Errorf("unable to register oauth plugin capability: %w", err)
		}

		// now validate the registered capability using the capability itself
		v := plusManager.GetConfigValidator(rportplus.PlusOAuthCapability)
		if v != nil {
			err = v.ValidateConfig()
			if err != nil {
				return fmt.Errorf("invalid oauth configuration: %w", err)
			}
		}

		logger.Infof("oauth capability registered")
	}

	// always register the plus status capability
	_, err = plusManager.RegisterCapability(rportplus.PlusStatusCapability, &status.Capability{
		Config: nil,
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("unable to register plus status capability: %w", err)
	}
	logger.Infof("plus status capability registered")

	// always register the plus license capability
	_, err = plusManager.RegisterCapability(rportplus.PlusLicenseCapability, &licensecap.Capability{
		Config: nil,
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("unable to register plus license capability: %w", err)
	}
	logger.Infof("plus license capability registered")

	// register the plus alerting capability
	_, err = plusManager.RegisterCapability(rportplus.PlusAlertingCapability, &alertingcap.Capability{
		Config: &alertingcap.Config{
			MaxWorkers:    maxAlertingWorkers,
			AlertsLogPath: cfg.Server.DataDir,
		},
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("unable to register plus alerting capability: %w", err)
	}
	logger.Infof("plus alerting capability registered")

	// always register the plus extended permission capability
	_, err = plusManager.RegisterCapability(rportplus.PlusExtendedPermissionCapability, &extendedpermission.Capability{
		Config: nil,
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("unable to register plus extended permission capability: %w", err)
	}
	logger.Infof("plus extended permission capability registered")

	return nil
}
