package capabilities

import (
	"github.com/openrport/openrport/server/chconfig"
	chshare "github.com/openrport/openrport/share"
	"github.com/openrport/openrport/share/models"
)

func NewServerCapabilities(cfg *chconfig.MonitoringConfig) *models.Capabilities {
	caps := models.Capabilities{
		ServerVersion:      chshare.BuildVersion,
		MonitoringVersion:  chshare.MonitoringVersion,
		IPAddressesVersion: chshare.IPAddressesVersion,
	}

	if !cfg.Enabled {
		caps.MonitoringVersion = 0
	}
	return &caps
}
