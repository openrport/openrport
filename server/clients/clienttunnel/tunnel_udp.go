package clienttunnel

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/openrport/openrport/share/comm"
	"github.com/openrport/openrport/share/logger"
	"github.com/openrport/openrport/share/models"
)

var udpReadTimeout = time.Second

type tunnelUDP struct {
	*logger.Logger
	models.Remote
	sshConn     ssh.Conn
	acl         atomic.Pointer[TunnelACL] // parsed Remote.ACL field
	idleTimeout time.Duration

	conn    *net.UDPConn
	channel *comm.UDPChannel
	done    chan struct{}
	cancel  func()

	mtx        sync.Mutex
	lastActive time.Time
}

func newTunnelUDP(logger *logger.Logger, ssh ssh.Conn, remote models.Remote, acl *TunnelACL) *tunnelUDP {
	t := &tunnelUDP{
		Logger:      logger,
		Remote:      remote,
		sshConn:     ssh,
		done:        make(chan struct{}),
		lastActive:  time.Now(),
		idleTimeout: time.Duration(remote.IdleTimeoutMinutes) * time.Minute,
	}
	t.SetACL(acl)
	return t
}

func (t *tunnelUDP) Start(ctx context.Context) error {
	t.Logger.Debugf("Starting udp tunnel...")
	remoteAddr := t.Remote.Remote() + "/udp"
	sshChan, reqs, err := t.sshConn.OpenChannel("rport", []byte(remoteAddr))
	if err != nil {
		return err
	}
	go ssh.DiscardRequests(reqs)

	return t.start(ctx, sshChan)
}

func (t *tunnelUDP) start(ctx context.Context, sshChan io.ReadWriter) error {
	a, err := net.ResolveUDPAddr("udp", t.Local())
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", a)
	if err != nil {
		return err
	}
	t.conn = conn

	ctx, t.cancel = context.WithCancel(ctx)

	t.channel = comm.NewUDPChannel(sshChan)

	go func() {
		err := t.runInbound(ctx)
		if err != nil {
			t.Errorf("Error receiving UDP: %v", err)
		}
	}()
	go func() {
		err := t.runOutbound(ctx)
		if err != nil {
			t.Errorf("Error sending UDP: %v", err)
		}
	}()

	return nil
}

func (t *tunnelUDP) runInbound(ctx context.Context) error {
	defer t.conn.Close()
	defer close(t.done)

	const maxMTU = 9012
	buff := make([]byte, maxMTU)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		err := t.conn.SetReadDeadline(time.Now().Add(udpReadTimeout))
		if err != nil {
			return err
		}

		n, sourceAddr, err := t.conn.ReadFromUDP(buff)
		if e, ok := err.(net.Error); ok && (e.Timeout() || e.Temporary()) {
			continue
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		t.setLastActive()

		acl := t.acl.Load()
		if acl != nil {
			if !acl.CheckAccess(sourceAddr.IP) {
				t.Debugf("Access rejected. Remote addr: %s", sourceAddr)
				continue
			}
		}

		err = t.channel.Encode(sourceAddr, buff[:n])
		if err != nil {
			return err
		}
	}
}

func (t *tunnelUDP) runOutbound(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		addr, data, err := t.channel.Decode()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		t.setLastActive()

		_, err = t.conn.WriteToUDP(data, addr)
		if err != nil {
			return err
		}
	}
}

func (t *tunnelUDP) Terminate(force bool) error {
	t.cancel()
	<-t.done

	return nil
}

func (t *tunnelUDP) LastActive() time.Time {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	return t.lastActive
}

func (t *tunnelUDP) setLastActive() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	t.lastActive = time.Now()
}

func (t *tunnelUDP) SetACL(acl *TunnelACL) {
	t.acl.Store(acl)
}
