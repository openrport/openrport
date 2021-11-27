package capabilities

import (
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/models"
)

func NewServerCapabilities() *models.Capabilities {
	return &models.Capabilities{
		ServerVersion:     chshare.BuildVersion,
		MonitoringVersion: chshare.MonitoringVersion,
	}
}
