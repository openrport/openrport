package clienttunnel

import (
	"fmt"
	"net"
	"strings"
)

type TunnelACL struct {
	AllowedIPs []net.IPNet
}

func (a *TunnelACL) AddACL(aclStr string) {
	lh, _ := parseIPNet(aclStr)
	a.AllowedIPs = append(a.AllowedIPs, *lh)
}

// CheckAccess returns true if connection from specified address is allowed
func (a TunnelACL) CheckAccess(ip net.IP) bool {
	if len(a.AllowedIPs) == 0 {
		return true
	}
	for _, allowed := range a.AllowedIPs {
		if allowed.Contains(ip) {
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
		ipNet, err := parseIPNet(strVal)
		if err != nil {
			return nil, err
		}

		acl.AllowedIPs = append(acl.AllowedIPs, *ipNet)
	}
	return acl, nil
}

func parseIPNet(strVal string) (*net.IPNet, error) {
	var ip net.IP
	var ipNet *net.IPNet
	var err error
	if strings.ContainsRune(strVal, '/') {
		ip, ipNet, err = net.ParseCIDR(strVal)
		if err != nil {
			return nil, err
		}
	} else {
		ip = net.ParseIP(strVal)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP addr: %s", strVal)
		}
	}

	if ip.Equal(net.IPv4zero) || ip.Equal(net.IPv6zero) {
		return nil, fmt.Errorf("%s would allow access to everyone. If that's what you want, do not set the ACL", ip)
	}

	if ipNet == nil {
		// if range is not specified, specify mask for one addr (/32 or /128)
		ipMask := net.IPv4Mask(255, 255, 255, 255)
		if ip.To4() == nil {
			ipMask = net.CIDRMask(128, 128)
		}
		ipNet = &net.IPNet{IP: ip, Mask: ipMask}
	}

	return ipNet, nil
}
