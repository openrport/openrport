package networking_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/client/monitoring/networking"
)

func TestNetWatcher_Results(t *testing.T) {
	watcher := networking.NewWatcher(networking.NetWatcherConfig{
		NetInterfaceExclude:             nil,
		NetInterfaceExcludeRegex:        []string{"^vnet(.*)$", "^virbr(.*)$", "^vmnet(.*)$", "^vEthernet(.*)$", "^docker(.*)$"},
		NetInterfaceExcludeDisconnected: true,
		NetInterfaceExcludeLoopback:     true,
		NetMetrics:                      []string{"in_B_per_s", "out_B_per_s"},
		NetInterfaceMaxSpeed:            1000 * 1000 * 1,
	})
	assert.NotNil(t, watcher)

	m, err := watcher.Results()
	assert.Nil(t, err)
	assert.NotNil(t, m)
	time.Sleep(5 * time.Second)
	m, err = watcher.Results()
	assert.Nil(t, err)
	assert.NotNil(t, m)
	fmt.Println(m.ToJSON())
}
