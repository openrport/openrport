package comm

import (
	"time"

	"golang.org/x/crypto/ssh"
)

func PingConnectionWithTimeout(conn ssh.Conn, timeout time.Duration) (ok bool, response []byte, rtt time.Duration, err error) {
	timerStart := time.Now()
	ok, response, err = SendRequestWithTimeout(conn, RequestTypePing, true, nil, timeout)
	return ok, response, time.Since(timerStart), err
}
