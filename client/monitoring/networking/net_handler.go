package networking

import (
	"github.com/cloudradar-monitoring/rport/share/clientconfig"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type NetHandler struct {
	config     *clientconfig.MonitoringConfig
	netWatcher *CAgentNetWatcher
}

func NewNetHandler(config *clientconfig.MonitoringConfig) *NetHandler {
	nh := &NetHandler{config: config}
	if config.LanCard != nil || config.WanCard != nil {
		nh.netWatcher = NewCAgentNetWatcher()
	}
	return nh
}

func (nh *NetHandler) GetNets() (*models.NetBytes, *models.NetBytes, error) {
	if nh.netWatcher == nil {
		return nil, nil, nil //no net data required
	}

	var mm map[string]interface{}
	mm, err := nh.netWatcher.Results()
	if err != nil {
		return nil, nil, err
	}

	var netLan *models.NetBytes
	if nh.config.LanCard != nil {
		netLan = getNetBytesFromMap(mm, nh.config.LanCard.Name)
	}

	var netWan *models.NetBytes
	if nh.config.WanCard != nil {
		netWan = getNetBytesFromMap(mm, nh.config.WanCard.Name)
	}

	return netLan, netWan, err
}

func getNetBytesFromMap(mm map[string]interface{}, interfaceName string) *models.NetBytes {
	netBytes := &models.NetBytes{}

	bytesIn := mm[inBytesName+"."+interfaceName]
	if bIn, ok := bytesIn.(int); ok {
		netBytes.In = bIn
	}

	bytesOut := mm[outBytesName+"."+interfaceName]
	if bOut, ok := bytesOut.(int); ok {
		netBytes.Out = bOut
	}

	return netBytes
}
