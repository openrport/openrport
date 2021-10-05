package system

import (
	"net"
	"time"
)

// IsPortOpen returns (true, nil) if a given TCP port is open on a given host.
// (false, err) - otherwise or if timeout is reached. 'hostPort' is in format <host>:<port>
func IsPortOpen(hostPort string, timeout time.Duration) (bool, error) {
	conn, err := net.DialTimeout("tcp", hostPort, timeout)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	return true, nil
}
