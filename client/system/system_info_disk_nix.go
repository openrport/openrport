// +build !windows

package system

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/disk"
)

func DiskUsage(ctx context.Context) error {
	usageStat, err := disk.UsageWithContext(ctx, "/")
	if err != nil {
		return err
	}

	fmt.Println("UsageStat:", usageStat)
	return nil
}
