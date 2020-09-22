package ports

import (
	"net"
	"time"
)

const DefaultDialTimeout = 3 * time.Second

// IsPortOpen returns (true, nil) if a given TCP port is open on a given host, otherwise - (false, err).
func IsPortOpen(host, port string) (bool, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), DefaultDialTimeout)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	return true, err
}
