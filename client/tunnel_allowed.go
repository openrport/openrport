package chclient

import (
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Used to override in tests
var lookupIP = net.LookupIP

func TunnelIsAllowed(tunnelAllowed []string, remote string) (bool, error) {
	if len(tunnelAllowed) == 0 {
		return true, nil
	}

	host, remotePort, err := net.SplitHostPort(remote)
	if err != nil {
		return false, err
	}
	ips, err := lookupIP(host)
	if err != nil {
		return false, err
	}

iploop:
	for _, ip := range ips {
		for _, ta := range tunnelAllowed {
			ipnet, port, err := ParseTunnelAllowed(ta)
			if err != nil {
				return false, err
			}

			if ipnet != nil {
				if !ipnet.Contains(ip) {
					continue
				}
			}

			if port != "" {
				if port != remotePort {
					continue
				}
			}
			continue iploop
		}
		return false, nil
	}

	return true, nil
}

func ParseTunnelAllowed(input string) (*net.IPNet, string, error) {
	var err error

	parts := strings.Split(input, ":")
	if len(parts) < 1 || len(parts) > 2 {
		return nil, "", errors.Errorf("invalid value: %q", input)
	}

	portStr := ""
	if len(parts) == 2 {
		portStr = parts[1]
	}

	var ipnet *net.IPNet
	if parts[0] != "" {
		_, ipnet, err = net.ParseCIDR(parts[0])
		if err != nil {
			ip := net.ParseIP(parts[0])
			if ip == nil {
				if len(parts) == 1 {
					portStr = parts[0]
				} else {
					return nil, "", errors.Errorf("invalid ip range: %q", parts[0])
				}
			} else {
				ipnet = &net.IPNet{
					IP:   ip,
					Mask: net.CIDRMask(32, 32),
				}
			}
		}
	}

	if portStr == "" && ipnet == nil {
		return nil, "", errors.Errorf("empty value not allowed: %q", input)
	}

	if portStr != "" {
		if _, err := strconv.Atoi(portStr); err != nil {
			return nil, "", errors.Errorf("invalid port: %q", portStr)
		}
	}

	return ipnet, portStr, nil
}
