package chserver

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

// Tunnel represents active remote proxy connection
type Tunnel struct {
	chshare.Remote
	*chshare.Logger `json:"-"`

	ID string `json:"id"`

	sshConn                   ssh.Conn
	connectionIDAutoIncrement int
	stopFn                    func()
	wg                        sync.WaitGroup
}

func NewTunnel(logger *chshare.Logger, ssh ssh.Conn, id string, remote *chshare.Remote) *Tunnel {
	return &Tunnel{
		Logger:  logger.Fork("tunnel#%s:%s", id, remote),
		Remote:  *remote,
		ID:      id,
		sshConn: ssh,
	}
}

func (t *Tunnel) Start(ctx context.Context) error {
	l, err := net.Listen("tcp4", t.LocalHost+":"+t.LocalPort)
	if err != nil {
		return fmt.Errorf("%s: %s", t.Logger.Prefix(), err)
	}

	ctx, t.stopFn = context.WithCancel(ctx)
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
	t.wg.Add(1)
	defer func() {
		t.wg.Done()
	}()

	t.Infof("Listening")
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			l.Close()
			t.Infof("Closed")
		case <-done:
		}
	}()
	for {
		src, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				//listener closed
			default:
				t.Infof("Accept error: %s", err)
			}
			close(done)
			return
		}
		go t.accept(src)
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
		l.Infof("Stream error: %s", err)
		return
	}
	go ssh.DiscardRequests(reqs)
	//then pipe
	s, r := chshare.Pipe(src, dst)
	l.Debugf("Close (sent %s received %s)", sizestr.ToString(s), sizestr.ToString(r))
}
