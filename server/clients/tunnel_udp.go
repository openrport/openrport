package clients

import (
	"context"
	"io"
	"net"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type tunnelUDP struct {
	*logger.Logger
	models.Remote
	sshConn ssh.Conn
	acl     *TunnelACL // parsed Remote.ACL field

	conn    *net.UDPConn
	channel *comm.UDPChannel
	done    chan struct{}
	cancel  func()
}

func newTunnelUDP(logger *logger.Logger, ssh ssh.Conn, remote models.Remote, acl *TunnelACL) *tunnelUDP {
	return &tunnelUDP{
		Logger:  logger,
		Remote:  remote,
		sshConn: ssh,
		acl:     acl,
		done:    make(chan struct{}),
	}
}

func (t *tunnelUDP) Start(ctx context.Context) (chan bool, error) {
	remoteAddr := t.Remote.Remote() + "/udp"
	sshChan, reqs, err := t.sshConn.OpenChannel("rport", []byte(remoteAddr))
	if err != nil {
		return nil, err
	}
	go ssh.DiscardRequests(reqs)

	return t.start(ctx, sshChan)
}

func (t *tunnelUDP) start(ctx context.Context, sshChan io.ReadWriter) (chan bool, error) {
	a, err := net.ResolveUDPAddr("udp", t.Local())
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", a)
	if err != nil {
		return nil, err
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

	return nil, nil
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

		err := t.conn.SetReadDeadline(time.Now().Add(time.Second))
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
