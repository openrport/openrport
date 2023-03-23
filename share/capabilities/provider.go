package capabilities

import (
	"github.com/realvnc-labs/rport/server/chconfig"
	chshare "github.com/realvnc-labs/rport/share"
	"github.com/realvnc-labs/rport/share/models"
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
