package fs

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/disk"
)

const fsInfoRequestTimeout = time.Second * 10

func getFsPartitionUsageInfo(mountPoint string) (*disk.UsageStat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fsInfoRequestTimeout)
	defer cancel()
	return disk.UsageWithContext(ctx, mountPoint)
}
