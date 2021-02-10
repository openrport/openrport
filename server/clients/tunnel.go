package clients

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

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
	stopFn                    func()
	wg                        sync.WaitGroup
	acl                       *TunnelACL // parsed Remote.ACL field
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

func (t *Tunnel) Terminate() {
	if t.stopFn == nil {
		return
	}

	t.stopFn()
	t.wg.Wait()
	t.Infof("stopped")
	t.stopFn = nil
}

func (t *Tunnel) listen(ctx context.Context, l net.Listener) {
	defer func() {
		t.wg.Done()
	}()

	t.Infof("Listening")

	// background goroutine to close the listener when Done channel is closed
	go func() {
		<-ctx.Done()
		if err := l.Close(); err != nil {
			t.Errorf("Failed to close: %v", err)
			return
		}
		t.Infof("Closed")
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
			continue
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
			t.accept(conn)
			t.wg.Done()
		}()
	}
}

func (t *Tunnel) accept(src io.ReadWriteCloser) {
	defer src.Close()
	t.connectionIDAutoIncrement++
	cid := t.connectionIDAutoIncrement
	l := t.Fork("conn#%d", cid)
	l.Debugf("Open")
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
