package chserver

import (
	"context"
	"errors"
	"fmt"

	rportplus "github.com/realvnc-labs/rport/plus"
	alertingcap "github.com/realvnc-labs/rport/plus/capabilities/alerting"
	"github.com/realvnc-labs/rport/plus/capabilities/extendedpermission"
	licensecap "github.com/realvnc-labs/rport/plus/capabilities/license"
	"github.com/realvnc-labs/rport/plus/capabilities/oauth"
	"github.com/realvnc-labs/rport/plus/capabilities/status"
	"github.com/realvnc-labs/rport/plus/license"
	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/realvnc-labs/rport/share/files"
	"github.com/realvnc-labs/rport/share/logger"
)

var (
	ErrPlusNotEnabled           = errors.New("rport-plus not enabled")
	ErrPlusLicenseNotConfigured = errors.New("rport-plus license not configured")
)

// EnablePlusIfLicensed will initialize a new plus manager and request registration of the desired
// capabilities
func EnablePlusIfLicensed(ctx context.Context, cfg *chconfig.Config, filesAPI files.FileAPI) (plusManager rportplus.Manager, err error) {
	logger := logger.NewLogger("rport-plus", cfg.Logging.LogOutput, cfg.Logging.LogLevel)

	if !rportplus.IsPlusEnabled(cfg.PlusConfig) {
		logger.Infof("not enabled")
		return nil, ErrPlusNotEnabled
	}

	// If plus is enabled then ensure license checking is disabled until release 1.0
	if rportplus.IsPlusEnabled(cfg.PlusConfig) {
		if !rportplus.HasLicenseConfig(cfg.PlusConfig) {
			cfg.PlusConfig.LicenseConfig = &license.Config{
				CheckingEnabled: false,
			}
		} else {
			cfg.PlusConfig.LicenseConfig.CheckingEnabled = true
		}
	}

	if !rportplus.HasLicenseConfig(cfg.PlusConfig) {
		logger.Errorf(ErrPlusLicenseNotConfigured.Error())
		return nil, ErrPlusLicenseNotConfigured
	}

	// Use the DataDir from the server config for the Rport Plus plugin license config
	dataDir := cfg.Server.DataDir
	cfg.PlusConfig.LicenseConfig.DataDir = dataDir

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
