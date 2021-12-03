// +build windows

package networking

import (
	"net"

	"github.com/pkg/errors"
	utilnet "github.com/shirou/gopsutil/net"

	"github.com/cloudradar-monitoring/cagent/pkg/winapi"
)

type windowsLinkSpeedProvider struct {
	// interface name -> bytes per second
	cache         map[string]float64
	isInitialized bool
}

func newLinkSpeedProvider() linkSpeedProvider {
	return &windowsLinkSpeedProvider{
		cache: make(map[string]float64),
	}
}

func (p *windowsLinkSpeedProvider) init() error {
	interfacesInfo, err := winapi.GetAdaptersAddresses()
	if err != nil {
		return errors.Wrap(err, "GetAdaptersAddresses failed")
	}

	for _, interfaceInfo := range interfacesInfo {
		p.cache[interfaceInfo.GetInterfaceName()] = float64(interfaceInfo.ReceiveLinkSpeed) / 8
	}

	p.isInitialized = true
	return nil
}

func (p *windowsLinkSpeedProvider) GetMaxAvailableLinkSpeed(ifName string) (float64, error) {
	if !p.isInitialized {
		err := p.init()
		if err != nil {
			return 0, err
		}
	}

	result, exists := p.cache[ifName]
	if !exists {
		return 0, errors.New("no speed information found")
	}
	return result, nil
}

// the functions overwrites the gopsutil/net IOCounters() implementation to support higher values of counters
func getNetworkIOCounters() ([]utilnet.IOCountersStat, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var ret []utilnet.IOCountersStat

	for _, ifi := range ifs {
		c := utilnet.IOCountersStat{
			Name: ifi.Name,
		}

		row := winapi.MibIfRow2{InterfaceIndex: uint32(ifi.Index)}
		err := winapi.GetIfEntry2(&row)
		if err != nil {
			return nil, err
		}
		c.BytesSent = row.OutOctets
		c.BytesRecv = row.InOctets
		c.PacketsSent = row.OutUcastPkts
		c.PacketsRecv = row.InUcastPkts
		c.Errin = row.InErrors
		c.Errout = row.OutErrors
		c.Dropin = row.InDiscards
		c.Dropout = row.OutDiscards

		ret = append(ret, c)
	}
	return ret, nil
}
