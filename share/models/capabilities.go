package models

import chshare "github.com/cloudradar-monitoring/rport/share"

type Capabilities struct {
	ServerVersion     string
	MonitoringVersion int
}

func NewCapabilities() *Capabilities {
	return &Capabilities{
		ServerVersion:     chshare.BuildVersion,
		MonitoringVersion: chshare.MonitoringVersion,
	}
}
