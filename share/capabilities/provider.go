package capabilities

import (
	"github.com/cloudradar-monitoring/rport/server/chconfig"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/models"
)

func NewServerCapabilities(cfg *chconfig.MonitoringConfig) *models.Capabilities {
	caps := models.Capabilities{
		ServerVersion:     chshare.BuildVersion,
		MonitoringVersion: chshare.MonitoringVersion,
	}

	if !cfg.Enabled {
		caps.MonitoringVersion = 0
	}
	return &caps
}
