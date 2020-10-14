package chclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
)

//Client represents a client instance
type Client struct {
	*chshare.Logger
	connReq *chshare.ConnectionRequest

	Config         *Config
	sshConfig      *ssh.ClientConfig
	sshConn        ssh.Conn
	running        bool
	runningc       chan error
	connStats      chshare.ConnStats
	cmdExec        CmdExecutor
	curCmdPID      *int
	curCmdPIDMutex sync.Mutex
}

//NewClient creates a new client instance
func NewClient(config *Config) *Client {
	connectionReq := &chshare.ConnectionRequest{
		Version: chshare.BuildVersion,
		ID:      config.Client.ID,
		Name:    config.Client.Name,
		Tags:    config.Client.Tags,
		Remotes: config.Client.remotes,
	}
	connectionReq.OS, _ = chshare.Uname()
	connectionReq.Hostname, _ = os.Hostname()

	client := &Client{
		Logger:   chshare.NewLogger("client", config.Logging.LogOutput, config.Logging.LogLevel),
		Config:   config,
		connReq:  connectionReq,
		running:  true,
		runningc: make(chan error, 1),
		cmdExec:  NewCmdExecutor(),
	}

	user, pass := chshare.ParseAuth(config.Client.Auth)

	client.sshConfig = &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		ClientVersion:   "SSH-" + chshare.ProtocolVersion + "-client",
		HostKeyCallback: client.verifyServer,
		Timeout:         30 * time.Second,
	}

	return client
}

//Run starts client and blocks while connected
func (c *Client) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := c.Start(ctx); err != nil {
		return err
	}
	return c.Wait()
}

func (c *Client) verifyServer(hostname string, remote net.Addr, key ssh.PublicKey) error {
	got := chshare.FingerprintKey(key)
	if c.Config.Client.Fingerprint != "" && !strings.HasPrefix(got, c.Config.Client.Fingerprint) {
		return fmt.Errorf("Invalid fingerprint (%s)", got)
	}
	//overwrite with complete fingerprint
	c.Infof("Fingerprint %s", got)
	return nil
}

//Start client and does not block
func (c *Client) Start(ctx context.Context) error {
	via := ""
	if c.Config.Client.proxyURL != nil {
		via = " via " + c.Config.Client.proxyURL.String()
	}

	c.Infof("Connecting to %s%s\n", c.Config.Client.Server, via)
	//optional keepalive loop
	if c.Config.Connection.KeepAlive > 0 {
		go c.keepAliveLoop()
	}
	//connection loop
	go c.connectionLoop(ctx)
	return nil
}

func (c *Client) keepAliveLoop() {
	for c.running {
		time.Sleep(c.Config.Connection.KeepAlive)
		if c.sshConn != nil {
			_, _, _ = c.sshConn.SendRequest(comm.RequestTypePing, true, nil)
		}
	}
}

func (c *Client) connectionLoop(ctx context.Context) {
	//connection loop!
	var connerr error
	b := &backoff.Backoff{Max: c.Config.Connection.MaxRetryInterval}
	for c.running {
		if connerr != nil {
			attempt := int(b.Attempt())
			d := b.Duration()
			c.showConnectionError(connerr, attempt)
			//give up?
			if c.Config.Connection.MaxRetryCount >= 0 && attempt >= c.Config.Connection.MaxRetryCount {
				break
			}
			c.Errorf("Retrying in %s...", d)
			connerr = nil
			chshare.SleepSignal(d)
		}
		d := websocket.Dialer{
			ReadBufferSize:   1024,
			WriteBufferSize:  1024,
			HandshakeTimeout: 45 * time.Second,
			Subprotocols:     []string{chshare.ProtocolVersion},
		}
		//optionally proxy
		if c.Config.Client.proxyURL != nil {
			if strings.HasPrefix(c.Config.Client.proxyURL.Scheme, "socks") {
				// SOCKS5 proxy
				if c.Config.Client.proxyURL.Scheme != "socks" && c.Config.Client.proxyURL.Scheme != "socks5h" {
					c.Errorf(
						"unsupported socks proxy type: %s:// (only socks5h:// or socks:// is supported)",
						c.Config.Client.proxyURL.Scheme)
					break
				}
				var auth *proxy.Auth
				if c.Config.Client.proxyURL.User != nil {
					pass, _ := c.Config.Client.proxyURL.User.Password()
					auth = &proxy.Auth{
						User:     c.Config.Client.proxyURL.User.Username(),
						Password: pass,
					}
				}
				socksDialer, err := proxy.SOCKS5("tcp", c.Config.Client.proxyURL.Host, auth, proxy.Direct)
				if err != nil {
					connerr = err
					continue
				}
				d.NetDial = socksDialer.Dial
			} else {
				// CONNECT proxy
				d.Proxy = func(*http.Request) (*url.URL, error) {
					return c.Config.Client.proxyURL, nil
				}
			}
		}
		wsConn, _, err := d.Dial(c.Config.Client.Server, c.Config.Connection.Headers())
		if err != nil {
			connerr = err
			continue
		}
		conn := chshare.NewWebSocketConn(wsConn)
		// perform SSH handshake on net.Conn
		c.Debugf("Handshaking...")
		sshConn, chans, reqs, err := ssh.NewClientConn(conn, "", c.sshConfig)
		if err != nil {
			if strings.Contains(err.Error(), "unable to authenticate") {
				c.Errorf("Authentication failed")
				c.Debugf(err.Error())
				connerr = err
				continue
			}
			c.Errorf(err.Error())
			break
		}
		ipv4, ipv6, _ := localIPAddresses()
		c.connReq.IPv4 = ipv4
		c.connReq.IPv6 = ipv6
		req, _ := chshare.EncodeConnectionRequest(c.connReq)
		c.Debugf("Sending connection request")
		t0 := time.Now()
		replyOk, respBytes, err := sshConn.SendRequest("new_connection", true, req)
		if err != nil {
			c.Errorf("connection request verification failed")
			break
		}
		if !replyOk {
			msg := string(respBytes)
			c.Errorf(msg)

			// if replied with client credentials already used - retry
			if strings.Contains(msg, "client is already connected:") {
				connerr = errors.New(msg)
				if closeErr := sshConn.Close(); closeErr != nil {
					c.Errorf(closeErr.Error())
				}
				continue
			}

			break
		}
		var remotes []*chshare.Remote
		err = json.Unmarshal(respBytes, &remotes)
		if err != nil {
			err = fmt.Errorf("can't decode reply payload: %s", err)
			c.Errorf(err.Error())
			break
		}
		c.Infof("Connected (Latency %s)", time.Since(t0))
		for _, r := range remotes {
			c.Infof("new tunnel: %s", r.String())
		}
		//connected
		b.Reset()
		c.sshConn = sshConn
		go c.handleSSHRequests(ctx, reqs)
		go c.connectStreams(chans)
		err = sshConn.Wait()
		//disconnected
		c.sshConn = nil
		if err != nil && err != io.EOF {
			connerr = err
			continue
		}
		c.Infof("Disconnected\n")
	}
	close(c.runningc)
}

func (c *Client) handleSSHRequests(ctx context.Context, reqs <-chan *ssh.Request) {
	for r := range reqs {
		var err error
		var resp interface{}
		switch r.Type {
		case comm.RequestTypeCheckPort:
			resp, err = checkPort(r.Payload)
		case comm.RequestTypeRunCmd:
			resp, err = c.HandleRunCmdRequest(ctx, r.Payload)
		default:
			c.Debugf("Unknown request: %q", r.Type)
			continue
		}

		if err != nil {
			c.Errorf("Failed to handle %q request: %v", r.Type, err)
			comm.ReplyError(c.Logger, r, err)
			continue
		}

		comm.ReplySuccessJSON(c.Logger, r, resp)
	}
}

func checkPort(payload []byte) (*comm.CheckPortResponse, error) {
	req, err := comm.DecodeCheckPortRequest(payload)
	if err != nil {
		return nil, err
	}

	open, checkErr := IsPortOpen(req.HostPort, req.Timeout)
	var errMsg string
	if checkErr != nil {
		errMsg = checkErr.Error()
	}
	return &comm.CheckPortResponse{
		Open:   open,
		ErrMsg: errMsg,
	}, nil
}

func (c *Client) showConnectionError(connerr error, attempt int) {
	maxAttempt := c.Config.Connection.MaxRetryCount
	//show error and attempt counts
	msg := fmt.Sprintf("Connection error: %s", connerr)
	if attempt > 0 {
		msg += fmt.Sprintf(" (Attempt: %d", attempt)
		if maxAttempt > 0 {
			msg += fmt.Sprintf("/%d", maxAttempt)
		}
		msg += ")"
	}
	c.Errorf(msg)
}

//Wait blocks while the client is running.
//Can only be called once.
func (c *Client) Wait() error {
	return <-c.runningc
}

//Close manually stops the client
func (c *Client) Close() error {
	c.running = false
	if c.sshConn == nil {
		return nil
	}
	return c.sshConn.Close()
}

func (c *Client) connectStreams(chans <-chan ssh.NewChannel) {
	for ch := range chans {
		remote := string(ch.ExtraData())
		stream, reqs, err := ch.Accept()
		if err != nil {
			c.Debugf("Failed to accept stream: %s", err)
			continue
		}
		go ssh.DiscardRequests(reqs)
		l := c.Logger.Fork("conn#%d", c.connStats.New())
		go chshare.HandleTCPStream(l, &c.connStats, stream, remote)
	}
}

// returns all local ipv4, ipv6 addresses
func localIPAddresses() ([]string, []string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}

	ipv4 := []string{}
	ipv6 := []string{}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.IsLoopback() {
				continue
			}
			if ip.To4() != nil {
				ipv4 = append(ipv4, ip.String())
			} else if ip.To16() != nil {
				ipv6 = append(ipv6, ip.String())
			}
		}
	}
	return ipv4, ipv6, nil
}

func (c *Client) getCurCmdPID() *int {
	c.curCmdPIDMutex.Lock()
	defer c.curCmdPIDMutex.Unlock()
	return c.curCmdPID
}

func (c *Client) setCurCmdPID(pid *int) {
	c.curCmdPIDMutex.Lock()
	defer c.curCmdPIDMutex.Unlock()
	c.curCmdPID = pid
}
