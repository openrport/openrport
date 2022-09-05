package ports

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"
	"github.com/shirou/gopsutil/net"

	"github.com/cloudradar-monitoring/rport/share/models"
)

type PortDistributor struct {
	allowedPorts mapset.Set

	portsPools map[string]mapset.Set
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
	checkProtocols := []string{protocol}
	if protocol == models.ProtocolTCPUDP {
		checkProtocols = []string{models.ProtocolTCP, models.ProtocolUDP}
	}
	for _, p := range checkProtocols {
		if d.portsPools[p] == nil {
			err := d.refresh(p)
			if err != nil {
				return 0, err
			}
		}
	}

	pool := d.portsPools[protocol]
	if protocol == models.ProtocolTCPUDP {
		pool = d.portsPools[models.ProtocolTCP].Intersect(d.portsPools[models.ProtocolUDP])
	}

	port := pool.Pop()
	if port == nil {
		return 0, fmt.Errorf("no ports available")
	}

	return port.(int), nil
}

func (d *PortDistributor) IsPortAllowed(port int) bool {
	return d.allowedPorts.Contains(port)
}

func (d *PortDistributor) IsPortBusy(protocol string, port int) bool {
	return !d.portsPools[protocol].Contains(port)
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

	d.portsPools[protocol] = d.allowedPorts.Difference(busyPorts)
	return nil
}

func ListBusyPorts(protocol string) (mapset.Set, error) {
	result := mapset.NewThreadUnsafeSet()
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
