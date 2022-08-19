package chserver

import (
	"fmt"

	rportplus "github.com/cloudradar-monitoring/rport/rport-plus"
	"github.com/cloudradar-monitoring/rport/rport-plus/capabilities/oauth"
	"github.com/cloudradar-monitoring/rport/rport-plus/capabilities/version"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

// EnablePlusIfLicensed will initialize a new plus manager and request registration of the desired
// capabilities
func EnablePlusIfLicensed(cfg *Config, filesAPI files.FileAPI) (plusManager rportplus.Manager, err error) {
	if cfg.PlusConfig != nil {
		logger := logger.NewLogger("rport-plus", cfg.Logging.LogOutput, cfg.Logging.LogLevel)
		plusManager, err = rportplus.NewPlusManager(cfg.PlusConfig, logger, filesAPI)
		if err != nil {
			return nil, err
		}
		logger.Infof("plus manager initialized")

		err = RegisterPlusCapabilities(plusManager, cfg, logger)
		if err != nil {
			return nil, err
		}
	}
	//  else {
	// 	logger.Infof("report-plus not enabled")
	// }
	return plusManager, nil
}

// RegisterPluginCapabilitities registers the rport-plus additional capabilities.
// All plus capabilities must be added here.
// TODO: no need to be part of the server component
func RegisterPlusCapabilities(plusManager rportplus.Manager, cfg *Config, logger *logger.Logger) (err error) {
	if cfg.PlusOAuthEnabled() {

		_, err := plusManager.RegisterCapability(rportplus.PlusOAuthCapability, &oauth.Capability{
			Config: cfg.OAuthConfig,
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

	// always register the version capability
	_, err = plusManager.RegisterCapability(rportplus.PlusVersionCapability, &version.Capability{
		Config: nil,
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("unable to register version plugin capability: %w", err)
	}
	logger.Infof("version capability registered")

	return nil
}
