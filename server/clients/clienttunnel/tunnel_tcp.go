package clienttunnel

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

type tunnelTCP struct {
	// Declare 64-bit integer before 32-bit for alignment when compiling Go on 32-bit ARM platforms
	lastConnClose             int64          // time stored as int64 so it can be used with atomic
	*logger.Logger
	models.Remote
	sshConn ssh.Conn
	acl     *TunnelACL // parsed Remote.ACL field

	stopFn                    func()
	connectionIDAutoIncrement int
	connCount                 int32
	wg                        sync.WaitGroup // TODO: verify whether wait group is needed here
}

func newTunnelTCP(logger *logger.Logger, ssh ssh.Conn, remote models.Remote, acl *TunnelACL) *tunnelTCP {
	return &tunnelTCP{
		Logger:  logger,
		Remote:  remote,
		sshConn: ssh,
		acl:     acl,
	}
}

func (t *tunnelTCP) Start(ctx context.Context) error {
	// TODO(m-terel): consider to use ListenTCP
	l, err := net.Listen("tcp4", t.Local())
	if err != nil {
		return fmt.Errorf("%s: %s", t.Logger.Prefix(), err)
	}

	ctx, t.stopFn = context.WithCancel(ctx)
	t.wg.Add(1)
	go t.listen(ctx, l)
	return nil
}

func (t *tunnelTCP) Terminate(force bool) error {
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

func (t *tunnelTCP) listen(ctx context.Context, l net.Listener) {
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

			if !t.acl.CheckAccess(tcpAddr.IP) {
				t.Debugf("Access rejected. Remote addr: %s", tcpAddr)
				conn.Close()
				continue
			}
		}

		t.wg.Add(1)
		go func() {
			t.accept(ctx, conn)
			t.wg.Done()
			atomic.StoreInt64(&t.lastConnClose, time.Now().Unix())
		}()
	}
}

func (t *tunnelTCP) LastActive() time.Time {
	if atomic.LoadInt32(&t.connCount) > 0 {
		return time.Now()
	}
	return time.Unix(atomic.LoadInt64(&t.lastConnClose), 0)
}

func (t *tunnelTCP) accept(ctx context.Context, src io.ReadWriteCloser) {
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
		l.Errorf("Could not establish TCP tunnel: %v", err)
		return
	}
	go ssh.DiscardRequests(reqs)
	//then pipe
	s, r := chshare.Pipe(src, dst)
	l.Debugf("Close (sent %s received %s)", sizestr.ToString(s), sizestr.ToString(r))
	close(done)
}
