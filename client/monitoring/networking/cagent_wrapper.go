package networking

import "github.com/cloudradar-monitoring/cagent/pkg/monitoring/networking"

type CAgentNetWatcher struct {
	*networking.NetWatcher
}

const inBytesName = "in_B_per_s"
const outBytesName = "out_B_per_s"

func NewCAgentNetWatcher() *CAgentNetWatcher {
	nwConfig := networking.NetWatcherConfig{
		NetInterfaceExclude:             nil,
		NetInterfaceExcludeRegex:        []string{"^vnet(.*)$", "^virbr(.*)$", "^vmnet(.*)$", "^vEthernet(.*)$", "^docker(.*)$"},
		NetInterfaceExcludeDisconnected: true,
		NetInterfaceExcludeLoopback:     true,
		NetMetrics:                      []string{inBytesName, outBytesName},
		NetInterfaceMaxSpeed:            1000 * 1000 * 1,
	}
	netWatcher := networking.NewWatcher(nwConfig)

	return &CAgentNetWatcher{netWatcher}
}
