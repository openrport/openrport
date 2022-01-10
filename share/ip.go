package chshare

import (
	"net"
	"net/http"
	"strings"
)

func RemoteIP(r *http.Request) string {
	ips := strings.Split(r.Header.Get("X-Forwarded-For"), ",")

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ips = append(ips, r.RemoteAddr)
	} else {
		ips = append(ips, ip)
	}

	publicIP, ok := firstValidIP(ips, false)
	if ok {
		return publicIP
	}

	privateIP, ok := firstValidIP(ips, true)
	if ok {
		return privateIP
	}

	return ips[0]
}

func firstValidIP(ips []string, allowPrivate bool) (string, bool) {
	for _, ipStr := range ips {
		ip := net.ParseIP(strings.TrimSpace(ipStr))
		if ip == nil {
			continue
		}
		if allowPrivate {
			return ip.String(), true
		}
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || isPrivate(ip) {
			continue
		}
		return ip.String(), true
	}
	return "", false
}

// This implementation is copied from go's `IP.IsPrivate`, it can be removed when we upgrade to go 17.
// isPrivate reports whether ip is a private address, according to
// RFC 1918 (IPv4 addresses) and RFC 4193 (IPv6 addresses).
func isPrivate(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		// Following RFC 1918, Section 3. Private Address Space which says:
		//   The Internet Assigned Numbers Authority (IANA) has reserved the
		//   following three blocks of the IP address space for private internets:
		//     10.0.0.0        -   10.255.255.255  (10/8 prefix)
		//     172.16.0.0      -   172.31.255.255  (172.16/12 prefix)
		//     192.168.0.0     -   192.168.255.255 (192.168/16 prefix)
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1]&0xf0 == 16) ||
			(ip4[0] == 192 && ip4[1] == 168)
	}
	// Following RFC 4193, Section 8. IANA Considerations which says:
	//   The IANA has assigned the FC00::/7 prefix to "Unique Local Unicast".
	return len(ip) == net.IPv6len && ip[0]&0xfe == 0xfc
}
