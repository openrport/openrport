package chclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"

	chshare "github.com/cloudradar-monitoring/rport/share"

	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

//Config represents a client configuration
type Config struct {
	shared           *chshare.Config
	Fingerprint      string
	Auth             string
	KeepAlive        time.Duration
	MaxRetryCount    int
	MaxRetryInterval time.Duration
	Server           string
	Proxy            string
	ID               string
	Name             string
	Tags             tags
	Remotes          []string
	Headers          http.Header
	DialContext      func(ctx context.Context, network, addr string) (net.Conn, error)
	LogOutput        *os.File
	LogLevel         chshare.LogLevel
}

type tags []string

func (t *tags) String() string {
	return strings.Join(*t, ",")
}

func (t *tags) Set(value string) error {
	*t = append(*t, value)
	return nil
}

//Client represents a client instance
type Client struct {
	*chshare.Logger
	config    *Config
	sshConfig *ssh.ClientConfig
	sshConn   ssh.Conn
	proxyURL  *url.URL
	server    string
	running   bool
	runningc  chan error
	connStats chshare.ConnStats
}

//NewClient creates a new client instance
func NewClient(config *Config) (*Client, error) {
	//apply default scheme
	if !strings.HasPrefix(config.Server, "http") {
		config.Server = "http://" + config.Server
	}
	if config.MaxRetryInterval < time.Second {
		config.MaxRetryInterval = 5 * time.Minute
	}
	u, err := url.Parse(config.Server)
	if err != nil {
		return nil, err
	}
	//apply default port
	if !regexp.MustCompile(`:\d+$`).MatchString(u.Host) {
		if u.Scheme == "https" || u.Scheme == "wss" {
			u.Host = u.Host + ":443"
		} else {
			u.Host = u.Host + ":80"
		}
	}
	//swap to websockets scheme
	u.Scheme = strings.Replace(u.Scheme, "http", "ws", 1)
	shared := &chshare.Config{}
	for _, s := range config.Remotes {
		var r *chshare.Remote
		r, err = chshare.DecodeRemote(s)
		if err != nil {
			return nil, fmt.Errorf("Failed to decode remote '%s': %s", s, err)
		}
		shared.Remotes = append(shared.Remotes, r)
	}
	config.shared = shared
	client := &Client{
		Logger:   chshare.NewLogger("client", config.LogOutput, config.LogLevel),
		config:   config,
		server:   u.String(),
		running:  true,
		runningc: make(chan error, 1),
	}

	if p := config.Proxy; p != "" {
		client.proxyURL, err = url.Parse(p)
		if err != nil {
			return nil, fmt.Errorf("Invalid proxy URL (%s)", err)
		}
	}

	user, pass := chshare.ParseAuth(config.Auth)

	client.sshConfig = &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		ClientVersion:   "SSH-" + chshare.ProtocolVersion + "-client",
		HostKeyCallback: client.verifyServer,
		Timeout:         30 * time.Second,
	}

	return client, nil
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
	expect := c.config.Fingerprint
	got := chshare.FingerprintKey(key)
	if expect != "" && !strings.HasPrefix(got, expect) {
		return fmt.Errorf("Invalid fingerprint (%s)", got)
	}
	//overwrite with complete fingerprint
	c.Infof("Fingerprint %s", got)
	return nil
}

//Start client and does not block
func (c *Client) Start(ctx context.Context) error {
	via := ""
	if c.proxyURL != nil {
		via = " via " + c.proxyURL.String()
	}

	c.Infof("Connecting to %s%s\n", c.server, via)
	//optional keepalive loop
	if c.config.KeepAlive > 0 {
		go c.keepAliveLoop()
	}
	//connection loop
	go c.connectionLoop()
	return nil
}

func (c *Client) keepAliveLoop() {
	for c.running {
		time.Sleep(c.config.KeepAlive)
		if c.sshConn != nil {
			_, _, _ = c.sshConn.SendRequest("ping", true, nil)
		}
	}
}

func (c *Client) connectionLoop() {
	//connection loop!
	var connerr error
	b := &backoff.Backoff{Max: c.config.MaxRetryInterval}
	for c.running {
		if connerr != nil {
			attempt := int(b.Attempt())
			d := b.Duration()
			c.showConnectionError(connerr, attempt)
			//give up?
			if c.config.MaxRetryCount >= 0 && attempt >= c.config.MaxRetryCount {
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
			NetDialContext:   c.config.DialContext,
		}
		//optionally proxy
		if c.proxyURL != nil {
			if strings.HasPrefix(c.proxyURL.Scheme, "socks") {
				// SOCKS5 proxy
				if c.proxyURL.Scheme != "socks" && c.proxyURL.Scheme != "socks5h" {
					c.Errorf(
						"unsupported socks proxy type: %s:// (only socks5h:// or socks:// is supported)",
						c.proxyURL.Scheme)
					break
				}
				var auth *proxy.Auth
				if c.proxyURL.User != nil {
					pass, _ := c.proxyURL.User.Password()
					auth = &proxy.Auth{
						User:     c.proxyURL.User.Username(),
						Password: pass,
					}
				}
				socksDialer, err := proxy.SOCKS5("tcp", c.proxyURL.Host, auth, proxy.Direct)
				if err != nil {
					connerr = err
					continue
				}
				d.NetDial = socksDialer.Dial
			} else {
				// CONNECT proxy
				d.Proxy = func(*http.Request) (*url.URL, error) {
					return c.proxyURL, nil
				}
			}
		}
		wsConn, _, err := d.Dial(c.server, c.config.Headers)
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
		c.config.shared.Version = chshare.BuildVersion
		c.config.shared.ID = c.config.ID
		c.config.shared.Name = c.config.Name
		c.config.shared.Tags = c.config.Tags
		c.config.shared.OS, _ = chshare.Uname()
		c.config.shared.Hostname, _ = os.Hostname()
		ipv4, ipv6, _ := localIPAddresses()
		c.config.shared.IPv4 = ipv4
		c.config.shared.IPv6 = ipv6
		conf, _ := chshare.EncodeConfig(c.config.shared)
		c.Debugf("Sending config")
		t0 := time.Now()
		configReplyOk, configReply, err := sshConn.SendRequest("config", true, conf)
		if err != nil {
			c.Errorf("Config verification failed")
			break
		}
		if !configReplyOk {
			c.Errorf(string(configReply))
			break
		}
		var remotes []*chshare.Remote
		err = json.Unmarshal(configReply, &remotes)
		if err != nil {
			err = fmt.Errorf("can't decode config reply payload: %s", err)
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
		go ssh.DiscardRequests(reqs)
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

func (c *Client) showConnectionError(connerr error, attempt int) {
	maxAttempt := c.config.MaxRetryCount
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
