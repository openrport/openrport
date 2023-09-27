package networking

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openrport/openrport/share/clientconfig"
	"github.com/openrport/openrport/share/models"
)

func TestNewNetHandler(t *testing.T) {
	netHandler := NewNetHandler(&clientconfig.MonitoringConfig{
		Enabled:                       true,
		Interval:                      0,
		FSTypeInclude:                 nil,
		FSPathExclude:                 nil,
		FSPathExcludeRecurse:          false,
		FSIdentifyMountpointsByDevice: false,
		PMEnabled:                     false,
		PMKerneltasksEnabled:          false,
		PMMaxNumberProcesses:          0,
		NetLan:                        []string{"wlp2s0", "10"},
		NetWan:                        nil,
		LanCard: &models.NetworkCard{
			Name:     "wlp2s0",
			MaxSpeed: 10,
		},
		WanCard: nil,
	})

	assert.NotNil(t, netHandler.netWatcher)
}

func TestNewNetHandlerNoNetworkMonitoring(t *testing.T) {
	netHandler := NewNetHandler(&clientconfig.MonitoringConfig{
		Enabled:                       true,
		Interval:                      0,
		FSTypeInclude:                 nil,
		FSPathExclude:                 nil,
		FSPathExcludeRecurse:          false,
		FSIdentifyMountpointsByDevice: false,
		PMEnabled:                     false,
		PMKerneltasksEnabled:          false,
		PMMaxNumberProcesses:          0,
		NetLan:                        []string{},
		NetWan:                        nil,
		LanCard:                       nil,
		WanCard:                       nil,
	})

	assert.Nil(t, netHandler.netWatcher)
}
