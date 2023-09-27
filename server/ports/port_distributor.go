package ports

import (
	"fmt"
	"sync"

	mapset "github.com/deckarep/golang-set"
	"github.com/shirou/gopsutil/v3/net"

	"github.com/openrport/openrport/share/models"
)

type PortDistributor struct {
	allowedPorts mapset.Set

	portsPools map[string]mapset.Set
	mu         sync.RWMutex
}

func NewPortDistributor(allowedPorts mapset.Set) *PortDistributor {
	return &PortDistributor{
		allowedPorts: allowedPorts,
		portsPools:   make(map[string]mapset.Set),
	}
}

// NewPortDistributorForTests is used only for unit-testing.
func NewPortDistributorForTests(allowedPorts, tcpPortsPool, udpPortsPool mapset.Set) *PortDistributor {
	return &PortDistributor{
		allowedPorts: allowedPorts,
		portsPools: map[string]mapset.Set{
			models.ProtocolTCP: tcpPortsPool,
			models.ProtocolUDP: udpPortsPool,
		},
	}
}

func (d *PortDistributor) GetRandomPort(protocol string) (int, error) {
	subProtocols := []string{protocol}
	if protocol == models.ProtocolTCPUDP {
		subProtocols = []string{models.ProtocolTCP, models.ProtocolUDP}
	}
	for _, p := range subProtocols {
		pool := d.getPoolFromMap(p)
		if pool == nil {
			err := d.refresh(p)
			if err != nil {
				return 0, err
			}
		}
	}

	port := d.getPool(protocol).Pop()
	if port == nil {
		return 0, fmt.Errorf("no ports available")
	}

	// Make sure port is removed from all pools for tcp+udp protocol
	for _, p := range subProtocols {
		pool := d.getPoolFromMap(p)
		pool.Remove(port)
	}

	return port.(int), nil
}

func (d *PortDistributor) IsPortAllowed(port int) bool {
	return d.allowedPorts.Contains(port)
}

func (d *PortDistributor) IsPortBusy(protocol string, port int) bool {
	return !d.getPool(protocol).Contains(port)
}

func (d *PortDistributor) getPool(protocol string) mapset.Set {
	pool := d.getPoolFromMap(protocol)
	if protocol == models.ProtocolTCPUDP {
		pool = d.portsPools[models.ProtocolTCP].Intersect(d.portsPools[models.ProtocolUDP])
	}
	return pool
}

func (d *PortDistributor) Refresh() error {
	err := d.refresh(models.ProtocolTCP)
	if err != nil {
		return err
	}
	err = d.refresh(models.ProtocolUDP)
	if err != nil {
		return err
	}
	return nil
}

func (d *PortDistributor) refresh(protocol string) error {
	busyPorts, err := ListBusyPorts(protocol)
	if err != nil {
		return err
	}

	pool := d.allowedPorts.Difference(busyPorts)
	d.setPool(protocol, pool)

	return nil
}

func ListBusyPorts(protocol string) (mapset.Set, error) {
	result := mapset.NewSet()
	connections, err := net.Connections(protocol)
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

func (d *PortDistributor) getPoolFromMap(protocol string) (pool mapset.Set) {
	d.mu.RLock()
	pool = d.portsPools[protocol]
	d.mu.RUnlock()
	return pool
}

func (d *PortDistributor) setPool(protocol string, pool mapset.Set) {
	d.mu.Lock()
	d.portsPools[protocol] = pool
	d.mu.Unlock()
}
