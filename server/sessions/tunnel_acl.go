package sessions

import (
	"fmt"
	"net"
	"strings"
)

type TunnelACL struct {
	AllowedIPs []net.IPNet
}

// CheckAccess returns true if connection from specified address is allowed
func (a TunnelACL) CheckAccess(addr *net.TCPAddr) bool {
	if len(a.AllowedIPs) == 0 {
		return true
	}
	for _, allowed := range a.AllowedIPs {
		if allowed.Contains(addr.IP) {
			return true
		}
	}
	return false
}

func ParseTunnelACL(str string) (*TunnelACL, error) {
	if str == "" {
		return nil, nil
	}

	acl := &TunnelACL{
		AllowedIPs: make([]net.IPNet, 0),
	}
	values := strings.Split(str, ",")
	for _, strVal := range values {
		var ip net.IP
		var ipNet *net.IPNet
		var err error
		if strings.ContainsRune(strVal, '/') {
			ip, ipNet, err = net.ParseCIDR(strVal)
			if err != nil {
				return nil, fmt.Errorf("invalid addr range %s: %s", strVal, err)
			}
		} else {
			ip = net.ParseIP(strVal)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP addr: %s", strVal)
			}
		}

		if ip.To4() == nil {
			return nil, fmt.Errorf("%s is not IPv4 address", strVal)
		}

		if ipNet == nil {
			// if range is not specified, specify mask for one addr (/32)
			ipMask := net.IPv4Mask(255, 255, 255, 255)
			ipNet = &net.IPNet{IP: ip, Mask: ipMask}
		}

		acl.AllowedIPs = append(acl.AllowedIPs, *ipNet)
	}
	return acl, nil
}
