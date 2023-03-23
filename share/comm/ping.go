package comm

import (
	"context"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/realvnc-labs/rport/share/logger"
)

func PingConnectionWithTimeout(ctx context.Context, conn ssh.Conn, timeout time.Duration, l *logger.Logger) (ok bool, response []byte, rtt time.Duration, err error) {
	timerStart := time.Now()
	ok, response, err = SendRequestWithTimeout(ctx, conn, RequestTypePing, true, nil, timeout, l)
	return ok, response, time.Since(timerStart), err
}
