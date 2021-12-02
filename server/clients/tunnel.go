package clients

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jpillora/sizestr"
	"golang.org/x/crypto/ssh"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
)

// TODO(m-terel): Refactor to use separate models for representation and business logic.
// Tunnel represents active remote proxy connection
type Tunnel struct {
	models.Remote
	*logger.Logger `json:"-"`

	ID string `json:"id"`

	sshConn                   ssh.Conn
	connectionIDAutoIncrement int
	connCount                 int32
	connCloseChan             chan bool
	stopFn                    func()
	wg                        sync.WaitGroup // TODO: verify whether wait group is needed here
	acl                       *TunnelACL     // parsed Remote.ACL field
	Proxy                     *TunnelProxy   `json:"-"`
}

func NewTunnel(logger *logger.Logger, ssh ssh.Conn, id string, remote *models.Remote, acl *TunnelACL) *Tunnel {
	return &Tunnel{
		Logger:  logger.Fork("tunnel#%s:%s", id, remote),
		Remote:  *remote,
		ID:      id,
		sshConn: ssh,
		acl:     acl,
	}
}

func (t *Tunnel) Start(ctx context.Context) (autoCloseChan chan bool, err error) {
	// TODO(m-terel): consider to use ListenTCP
	l, err := net.Listen("tcp4", t.LocalHost+":"+t.LocalPort)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", t.Logger.Prefix(), err)
	}

	ctx, t.stopFn = context.WithCancel(ctx)
	if t.IdleTimeoutMinutes > 0 {
		t.connCloseChan = make(chan bool)
		autoCloseChan = t.getAutoCloseChan(ctx)
	}
	t.wg.Add(1)
	go t.listen(ctx, l)
	return
}

func (t *Tunnel) Terminate(force bool) error {
	n := atomic.LoadInt32(&t.connCount)
	if !force && n > 0 {
		return fmt.Errorf("tunnel has %d active connection(s)", n)
	}
	if t.stopFn == nil {
		return nil
	}

	t.stopFn()
	t.wg.Wait()
	t.Infof("stopped")
	t.stopFn = nil
	return nil
}

func (t *Tunnel) listen(ctx context.Context, l net.Listener) {
	defer func() {
		t.wg.Done()
	}()

	t.Infof("Listening")

	// background goroutine to close the listener when context is canceled
	go func() {
		<-ctx.Done()
		if err := l.Close(); err != nil {
			t.Errorf("Failed to close listener: %v", err)
			return
		}
		t.Debugf("Listener closed")
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			// If Done channel was closed then listener was closed by the background goroutine. It causes
			// Accept to return an err. Check the Done channel to see whether it should continue or quit.
			select {
			case <-ctx.Done():
				//listener closed
			default:
				t.Errorf("Failed to accept connection: %v", err)
			}
			return
		}

		if t.acl != nil {
			tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr)
			if !ok {
				t.Errorf("Unsupported remote address type. Expected net.TCPAddr. %v", conn.RemoteAddr())
				conn.Close()
				continue
			}

			if !t.acl.CheckAccess(tcpAddr) {
				t.Debugf("Access rejected. Remote addr: %s", tcpAddr)
				conn.Close()
				continue
			}
		}

		t.wg.Add(1)
		go func() {
			t.accept(ctx, conn)
			t.wg.Done()
			if t.connCloseChan != nil {
				// just track when connection was closed, because connection creation is covered by connection counter
				t.connCloseChan <- true // TODO: on context close do not wait
			}
		}()
	}
}

// TODO: consider to create a separate background task to terminate all inactive tunnels based on some deadline/lastActivity time
func (t *Tunnel) getAutoCloseChan(ctx context.Context) chan bool {
	autoCloseChan := make(chan bool)
	idleTimeout := time.Duration(t.IdleTimeoutMinutes) * time.Minute
	go func() {
		for {
			select {
			case <-ctx.Done():
				// close if the ctx was canceled
				return
			case <-time.After(idleTimeout):
				// track time after the last activity,
				// if it reaches the timeout and there are no active connections - terminate the tunnel
				if atomic.LoadInt32(&t.connCount) > 0 {
					continue
				}
				t.Infof("Terminating... inactivity period is reached: %d minute(s)", t.IdleTimeoutMinutes)
				_ = t.Terminate(true)
				close(autoCloseChan)
				return
			case <-t.connCloseChan:
				// if there was some activity - continue to restart the inactivity tracking
				continue
			}
		}
	}()
	return autoCloseChan
}

func (t *Tunnel) accept(ctx context.Context, src io.ReadWriteCloser) {
	defer src.Close()
	t.connectionIDAutoIncrement++
	atomic.AddInt32(&t.connCount, 1)
	defer atomic.AddInt32(&t.connCount, -1)

	cid := t.connectionIDAutoIncrement
	l := t.Fork("conn#%d", cid)
	l.Debugf("Open")

	done := make(chan bool)
	// link ctx to conn
	go func() {
		select {
		case <-ctx.Done():
			if src.Close() == nil {
				l.Debugf("closed")
			}
		case <-done:
			// do nothing
		}
	}()

	if t.sshConn == nil {
		l.Debugf("No remote connection")
		return
	}
	//ssh request for tcp connection for this proxy's remote
	dst, reqs, err := t.sshConn.OpenChannel("rport", []byte(t.Remote.Remote()))
	if err != nil {
		l.Errorf("Stream error: %s", err)
		return
	}
	go ssh.DiscardRequests(reqs)
	//then pipe
	s, r := chshare.Pipe(src, dst)
	l.Debugf("Close (sent %s received %s)", sizestr.ToString(s), sizestr.ToString(r))
	close(done)
}
