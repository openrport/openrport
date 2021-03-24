package clients

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/jpillora/sizestr"
	"golang.org/x/crypto/ssh"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

// TODO(m-terel): Refactor to use separate models for representation and business logic.
// Tunnel represents active remote proxy connection
type Tunnel struct {
	chshare.Remote
	*chshare.Logger `json:"-"`

	ID string `json:"id"`

	sshConn                   ssh.Conn
	connectionIDAutoIncrement int
	connCount                 int32
	stopFn                    func()
	wg                        sync.WaitGroup // TODO: verify whether wait group is needed here
	acl                       *TunnelACL     // parsed Remote.ACL field
}

func NewTunnel(logger *chshare.Logger, ssh ssh.Conn, id string, remote *chshare.Remote, acl *TunnelACL) *Tunnel {
	return &Tunnel{
		Logger:  logger.Fork("tunnel#%s:%s", id, remote),
		Remote:  *remote,
		ID:      id,
		sshConn: ssh,
		acl:     acl,
	}
}

func (t *Tunnel) Start(ctx context.Context) error {
	// TODO(m-terel): consider to use ListenTCP
	l, err := net.Listen("tcp4", t.LocalHost+":"+t.LocalPort)
	if err != nil {
		return fmt.Errorf("%s: %s", t.Logger.Prefix(), err)
	}

	ctx, t.stopFn = context.WithCancel(ctx)
	t.wg.Add(1)
	go t.listen(ctx, l)
	return nil
}

func (t *Tunnel) Terminate(force bool) error {
	n := atomic.LoadInt32(&t.connCount)
	if !force && n > 0 {
		return fmt.Errorf("tunnel is still active: it has %d active connection(s)", n)
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
				return
			default:
				t.Errorf("Failed to accept connection: %v", err)
			}
			continue // TODO: return?
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
		}()
	}
}

func (t *Tunnel) accept(ctx context.Context, src io.ReadWriteCloser) {
	defer src.Close()
	t.connectionIDAutoIncrement++
	atomic.AddInt32(&t.connCount, 1)
	defer atomic.AddInt32(&t.connCount, -1)

	cid := t.connectionIDAutoIncrement
	l := t.Fork("conn#%d", cid)
	l.Debugf("Open")

	// link ctx to conn
	go func() {
		<-ctx.Done()
		if src.Close() == nil {
			l.Debugf("closed")
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
}
