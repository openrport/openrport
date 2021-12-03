package networking

import (
	"fmt"
	"net"

	utilnet "github.com/shirou/gopsutil/net"
	"github.com/sirupsen/logrus"

	"github.com/cloudradar-monitoring/rport/client/monitoring/helper"
)

type linkSpeedProvider interface {
	// GetMaxAvailableLinkSpeed returns link speed in bytes per second
	GetMaxAvailableLinkSpeed(ifName string) (float64, error)
}

func IPAddresses() (helper.MeasurementsMap, error) {
	var addresses []string

	// Fetch all interfaces
	interfaces, err := utilnet.Interfaces()
	if err != nil {
		return nil, err
	}

	// Check all interfaces for their addresses
INFLOOP:
	for _, inf := range interfaces {
		for _, flag := range inf.Flags {
			// Ignore loopback addresses
			if flag == "loopback" {
				continue INFLOOP
			}
		}

		// Append all addresses to our slice
		for _, addr := range inf.Addrs {
			addresses = append(addresses, addr.Addr)
		}
	}

	result := make(helper.MeasurementsMap)
	v4Count := uint32(1)
	v6Count := uint32(1)

	for _, address := range addresses {
		ipAddr, _, err := net.ParseCIDR(address)
		if err != nil {
			logrus.Warnf("Failed to parse IP address %s: %s", address, err)
			continue
		}
		// Check if ip is v4
		if v4 := ipAddr.To4(); v4 != nil {
			result[fmt.Sprintf("ipv4.%d", v4Count)] = v4.String()
			v4Count++
			continue
		}
		// Check if ip is v6
		if v6 := ipAddr.To16(); v6 != nil {
			result[fmt.Sprintf("ipv6.%d", v6Count)] = v6.String()
			v6Count++
			continue
		}

		logrus.Warnf("Could not determine if IP is v4 or v6: %s", address)
	}

	return result, nil
}

func isInterfaceLoobpack(netIf *utilnet.InterfaceStat) bool {
	for _, flag := range netIf.Flags {
		if flag == "loopback" {
			return true
		}
	}

	return false
}

func isInterfaceDown(netIf *utilnet.InterfaceStat) bool {
	for _, flag := range netIf.Flags {
		if flag == "up" {
			return false
		}
	}

	return true
}
