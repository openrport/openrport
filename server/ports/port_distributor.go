package ports

import (
	"fmt"
	"math"

	mapset "github.com/deckarep/golang-set"
	"github.com/shirou/gopsutil/net"
)

type PortDistributor struct {
	allowedPorts mapset.Set
	portsPool    mapset.Set
}

func NewPortDistributor(excludedPorts mapset.Set) *PortDistributor {
	return &PortDistributor{
		allowedPorts: setFromRange(1, math.MaxUint16).Difference(excludedPorts),
	}
}

func (d *PortDistributor) GetRandomPort() (int, error) {
	if d.portsPool == nil {
		err := d.Refresh()
		if err != nil {
			return 0, err
		}
	}

	port := d.portsPool.Pop()
	if port == nil {
		return 0, fmt.Errorf("no ports available")
	}
	return port.(int), nil
}

func (d *PortDistributor) Refresh() error {
	busyPorts, err := listBusyPorts()
	if err != nil {
		return err
	}

	d.portsPool = d.allowedPorts.Difference(busyPorts)
	return nil
}

func listBusyPorts() (mapset.Set, error) {
	result := mapset.NewThreadUnsafeSet()
	connections, err := net.Connections("all")
	if err != nil {
		return nil, err
	}

	for _, c := range connections {
		isActive := c.Status == "LISTEN" || c.Status == "NONE" || c.Status == ""
		if isActive && c.Laddr.Port != 0 {
			result.Add(int(c.Laddr.Port))
		}
	}

	return result, nil
}
