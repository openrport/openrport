// +build !windows

package networking

import (
	"context"
	"time"

	utilnet "github.com/shirou/gopsutil/net"
)

const netGetCountersTimeout = time.Second * 10

func getNetworkIOCounters() ([]utilnet.IOCountersStat, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), netGetCountersTimeout)
	defer cancelFn()
	return utilnet.IOCountersWithContext(ctx, true)
}
