package comm

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

func PingConnectionWithTimeout(conn ssh.Conn, timeout time.Duration) (bool, []byte, time.Duration, error) {
	var (
		ok         bool
		response   []byte
		err        error
		timerStart = time.Now()
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan bool, 1)
	go func() {
		ok, response, err = conn.SendRequest(RequestTypePing, true, nil)
		select {
		default:
			ch <- true
		case <-ctx.Done():
			return
		}
	}()
	select {
	case <-ch:
		return ok, response, time.Since(timerStart), err
	case <-time.After(timeout):
		return false, nil, 0, fmt.Errorf("timeout %s exceeded", timeout)
	}
}
