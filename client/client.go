package chclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/openrport/openrport/client/inventory"
	ipAddresses "github.com/openrport/openrport/client/ip_addresses"

	"github.com/openrport/openrport/share/random"

	"github.com/denisbrodbeck/machineid"
	"github.com/pkg/errors"

	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"
	"github.com/shirou/gopsutil/v3/host"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"

	"github.com/openrport/openrport/client/monitoring"
	"github.com/openrport/openrport/client/system"
	"github.com/openrport/openrport/client/updates"
	chshare "github.com/openrport/openrport/share"
	"github.com/openrport/openrport/share/comm"
	"github.com/openrport/openrport/share/files"
	"github.com/openrport/openrport/share/logger"
	"github.com/openrport/openrport/share/models"
)

const DialTimeout = 5 * 60 * time.Second
const AuthTimeout = 30 * time.Second
const MinConnectionBackoffWaitTime = 5 * time.Second
const MaxConnectionBackoffWaitTime = 10 * 60 * time.Second
const ServerReconnectRequestBackoffTime = 3 * 60 * time.Second
const InitialConnectionRequestSendDelayJitterMilliseconds = 10000
const SendRequestTimeout = 30 * time.Second
const MinSendRequestRetryWaitTime = 1 * time.Second
const BackoffOnServerTimeoutMaxDuration = 1 * time.Second
const MaxKeepAliveJitterMilliseconds = 5000

// Client represents a client instance
type Client struct {
	*logger.Logger

	SessionID          string
	configHolder       *ClientConfigHolder
	sshConfig          *ssh.ClientConfig
	sshConnection      ssh.Conn
	running            bool
	runningc           chan error
	connStats          chshare.ConnStats
	cmdExec            system.CmdExecutor
	systemInfo         system.SysInfo
	updates            *updates.Updates
	inventory          *inventory.Inventory
	monitor            *monitoring.Monitor
	ipAddressesFetcher *ipAddresses.Fetcher
	serverCapabilities *models.Capabilities
	filesAPI           files.FileAPI
	watchdog           *Watchdog

	mu sync.RWMutex
}

type sshClientConnection struct {
	Connection ssh.Conn
	Channels   <-chan ssh.NewChannel
	Requests   <-chan *ssh.Request
}

// NewClient creates a new client instance
func NewClient(config *ClientConfigHolder, filesAPI files.FileAPI) (*Client, error) {
	// Generate a session id that will not change while the client is running
	// This allows the server to resume sessions.
	sessionID, err := random.UUID4()
	if err != nil {
		return nil, fmt.Errorf("failed to create initial session id: %s", err)
	}

	cmdExec := system.NewCmdExecutor(logger.NewLogger("cmd executor", config.Logging.LogOutput, config.Logging.LogLevel))
	logger := logger.NewLogger("client", config.Logging.LogOutput, config.Logging.LogLevel)

	watchdog, err := NewWatchdog(config.Connection.WatchdogIntegration, config.Client.DataDir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create watchdog: %s", err)
	}

	systemInfo := system.NewSystemInfo(cmdExec)
	client := &Client{
		SessionID:          sessionID,
		Logger:             logger,
		configHolder:       config,
		running:            true,
		runningc:           make(chan error, 1),
		cmdExec:            cmdExec,
		systemInfo:         systemInfo,
		updates:            updates.New(logger, config.Client.UpdatesInterval),
		inventory:          inventory.New(logger, config.Client.InventoryInterval),
		monitor:            monitoring.NewMonitor(logger, config.Monitoring, systemInfo),
		ipAddressesFetcher: ipAddresses.NewFetcher(logger, config.Client.IPAPIURL, config.Client.IPRefreshMin),
		filesAPI:           filesAPI,
		watchdog:           watchdog,
	}

	client.sshConfig = &ssh.ClientConfig{
		User:            config.Client.AuthUser,
		Auth:            []ssh.AuthMethod{ssh.Password(config.Client.AuthPass)},
		ClientVersion:   "SSH-" + chshare.ProtocolVersion + "-client",
		HostKeyCallback: client.verifyServer,
		Timeout:         AuthTimeout,
	}

	logger.Infof("NewFetcher client instance with sessionID %s", sessionID)
	return client, nil
}

// Run starts client and blocks while connected
func (c *Client) Run(ctx context.Context) (err error) {
	if err = c.Start(ctx); err != nil {
		return err
	}

	err = c.Wait(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (c *Client) verifyServer(hostname string, remote net.Addr, key ssh.PublicKey) error {
	got := chshare.FingerprintKey(key)
	if c.configHolder.Client.Fingerprint != "" && !strings.HasPrefix(got, c.configHolder.Client.Fingerprint) {
		return fmt.Errorf("invalid fingerprint (%s)", got)
	}
	//overwrite with complete fingerprint
	c.Infof("Server's full fingerprint %s", got)
	return nil
}

// Start client and do not block
func (c *Client) Start(ctx context.Context) error {
	//optional keepalive loop
	if c.configHolder.Connection.KeepAlive > 0 {
		c.Infof("Keepalive job (client to server ping) started with interval %s", c.configHolder.Connection.KeepAlive)
		go c.keepAliveLoop(ctx)
	}

	//connection loop
	go c.connectionLoop(ctx, true)

	c.updates.Start(ctx)
	c.inventory.Start(ctx)

	return nil
}

func (c *Client) getConn() (sshConnection ssh.Conn) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sshConnection
}

func (c *Client) setConn(sshConnection ssh.Conn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sshConnection = sshConnection
}

func printMemStats(c *Client) {
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)

	c.Debugf("mem usage summary: liveobjects=%d, heapObjects=%d, heapAlloc=%d, numGC=%d, lastGC=%s",
		rtm.Mallocs-rtm.Frees,
		rtm.HeapObjects,
		rtm.HeapAlloc,
		rtm.NumGC,
		time.UnixMilli(int64(rtm.LastGC/1_000_000)),
	)
}

func (c *Client) keepAliveLoop(ctx context.Context) {
	for c.isRunning() {
		printMemStats(c)

		time.Sleep(c.configHolder.Connection.KeepAlive + (time.Duration(rand.Intn(MaxKeepAliveJitterMilliseconds)))*time.Millisecond)

		conn := c.getConn()

		if conn != nil {

			res, err := comm.WithRetry(func() (res *sendResponse, err error) {
				ok, _, rtt, err := comm.PingConnectionWithTimeout(ctx, conn, c.configHolder.Connection.KeepAliveTimeout, c.Logger)
				return &sendResponse{
					replyOk:   ok,
					rtt:       rtt,
					respBytes: nil,
				}, err
			}, canRetryFn, MinSendRequestRetryWaitTime, "ping", c.Logger)

			if err != nil || !res.replyOk {
				c.Errorf("Failed to send keepalive (client to server ping): %s", err)
				conn.Close()
			} else {
				msg := fmt.Sprintf("ping to %s succeeded within %s", conn.RemoteAddr(), res.rtt)
				c.Debugf(msg)
				c.watchdog.Ping(WatchdogStateConnected, msg)
			}
		}
	}

	c.Logger.Debugf("keepAliveLoop finished")
}

func (c *Client) connectionLoop(ctx context.Context, withInitialSendRequestDelay bool) {
	//connection loop!
	var connerr error
	switchbackChan := make(chan *sshClientConnection, 1)
	backoff := &backoff.Backoff{
		Min:    MinConnectionBackoffWaitTime + time.Duration(rand.Intn(60)),
		Max:    MaxConnectionBackoffWaitTime,
		Jitter: true,
	}

	for c.isRunning() {
		if connerr != nil {
			stopRetrying := c.handleConnectionError(backoff, connerr)
			if stopRetrying {
				break
			}
			connerr = nil
		}

		c.Logger.Debugf("conn loop attempt = %d", int(backoff.Attempt())+1)

		// make the connection attempt
		var sshClientConn *sshClientConnection
		var isPrimary bool
		select {
		// When switchback to main server succeeds we get connection on the channel, otherwise try to connect
		case sshClientConn = <-switchbackChan:
			isPrimary = true
		case <-ctx.Done():
			connerr = ctx.Err()
			continue
		default:
			var err error
			sshClientConn, isPrimary, err = c.connectToMainOrFallback()
			if err != nil {
				connerr = err // Setting a connerr causes the loop to sleep and try again later
				continue
			}
		}

		go c.handleSSHRequests(ctx, sshClientConn)
		go c.connectStreams(sshClientConn.Channels)

		switchbackCtx, cancelSwitchback := context.WithCancel(ctx)
		if !isPrimary {
			go c.handleServerSwitchBack(switchbackCtx, switchbackChan, sshClientConn)
		}

		if withInitialSendRequestDelay {
			delay := time.Duration(rand.Intn(InitialConnectionRequestSendDelayJitterMilliseconds)) * time.Millisecond
			c.Logger.Debugf("waiting for %d milliseconds before sending connection request", delay/time.Millisecond)
			time.Sleep(delay)
		}

		err := c.sendConnectionRequest(ctx, sshClientConn.Connection, MinSendRequestRetryWaitTime)
		if err != nil {
			// Connection request has failed then the connection will be closed and we try again
			cancelSwitchback()
			connerr = err
			continue
		}

		// Connection request has succeeded
		backoff.Reset()

		// Hand over the open SSH connection to the client
		c.setConn(sshClientConn.Connection)

		// Hand over the open SSH connection to subsystems running their own go routines
		c.updates.SetConn(sshClientConn.Connection)
		c.ipAddressesFetcher.SetConn(sshClientConn.Connection)
		c.monitor.SetConn(sshClientConn.Connection)
		c.inventory.SetConn(sshClientConn.Connection)

		// watch for shutting down due to ctx.Done
		go func() {
			<-ctx.Done()
			_ = c.CloseConnection()
			c.Logger.Infof("connection closed by ctx.Done")
		}()

		// now wait with the client handling SSH Requests and Channel Connections
		err = sshClientConn.Connection.Wait()

		c.Logger.Infof("connection wait stopped")

		c.setConn(nil)
		c.monitor.Stop()
		c.updates.Stop()
		c.ipAddressesFetcher.Stop()
		c.inventory.Stop()
		cancelSwitchback()

		// use of closed network connection happens when switchback closes the connection, ignore the error
		if err != nil && err != io.EOF && !strings.HasSuffix(err.Error(), "use of closed network connection") {
			connerr = err
		}

		c.Infof("Disconnected\n")
	}

	close(c.runningc)

	c.Infof("connectionLoop finished")
}

func (c *Client) handleConnectionError(backoff *backoff.Backoff, connerr error) (stopRetrying bool) {
	attempt := int(backoff.Attempt())

	c.showConnectionError(connerr, attempt)

	// check if the user has set a max retry limit
	if c.configHolder.Connection.MaxRetryCount >= 0 && attempt >= c.configHolder.Connection.MaxRetryCount {
		c.Errorf("connection error: max retries exceeded")
		return true // if so, stop trying
	}

	var d = backoff.Duration()
	if _, ok := connerr.(comm.TimeoutError); ok {
		c.Debugf("reseting backoff timer")
		// Timeout means the server isn't offline, so reset the backoff and use an initial short retry duration
		backoff.Reset()
		rand.Seed(time.Now().UnixNano())
		d = time.Duration(rand.Intn(int(backoff.Attempt()))) * BackoffOnServerTimeoutMaxDuration
	}
	msg := fmt.Sprintf("Retrying in %s...", d)
	c.Infof(msg)
	c.watchdog.Ping(WatchdogStateReconnecting, msg)
	chshare.SleepSignal(d)

	return false
}

func (c *Client) showConnectionError(connerr error, attempt int) {
	if errors.Is(connerr, context.Canceled) {
		c.Infof("connection error: context canceled")
		return
	}
	maxAttempt := c.configHolder.Connection.MaxRetryCount
	//show error and attempt counts
	msg := fmt.Sprintf("connection error: %s", connerr)
	if attempt > 0 {
		maxAttemptStr := fmt.Sprint(maxAttempt)
		if maxAttempt < 0 {
			maxAttemptStr = "infinite"
		}
		msg += fmt.Sprintf(" (attempt: %d of %s)", attempt, maxAttemptStr)
	}
	c.Errorf(msg)
	if strings.Contains(msg, "previous session was not properly closed") {
		c.Infof("Server will clean up orphaned sessions within its {check_clients_connection_interval} automatically.")
	}
}

func (c *Client) handleServerSwitchBack(switchbackCtx context.Context, switchbackChan chan *sshClientConnection, sshClientConn *sshClientConnection) {
	for {
		switchbackTimer := time.NewTimer(c.configHolder.Client.ServerSwitchbackInterval)
		select {
		case <-switchbackCtx.Done():
			switchbackTimer.Stop()
			return
		case <-switchbackTimer.C:
			switchbackConn, err := c.connect(c.configHolder.Client.Server)
			if err != nil {
				c.Errorf("Switchback failed: %v", err.Error())
				continue
			}
			c.Infof("Connected to main server, switching back.")
			switchbackChan <- switchbackConn
			sshClientConn.Connection.Close()
			return
		}
	}
}

func (c *Client) connectToMainOrFallback() (conn *sshClientConnection, isPrimary bool, err error) {
	servers := append([]string{c.configHolder.Client.Server}, c.configHolder.Client.FallbackServers...)
	for i, server := range servers {
		conn, err = c.connect(server)
		if err != nil {
			continue // Try the next server in the list
		}
		return conn, i == 0, nil
	}
	return nil, false, err
}

func (c *Client) connect(server string) (*sshClientConnection, error) {
	via := ""
	if c.configHolder.Client.ProxyURL != nil {
		via = " via " + c.configHolder.Client.ProxyURL.String()
	}
	c.Infof("Trying to connect to %s%s ...\n", server, via)
	c.Infof("Will wait up to %0.2f seconds for the server to respond", DialTimeout.Seconds())
	d, netDialer, err := c.setupDialer()
	if err != nil {
		return nil, err
	}

	//optionally proxy
	if c.configHolder.Client.ProxyURL != nil {
		err := c.addDialerProxySupport(d, netDialer)
		if err != nil {
			return nil, err
		}
	}

	wsConn, _, err := d.Dial(server, c.configHolder.Connection.HTTPHeaders)
	if err != nil {
		return nil, ConnectionErrorHints(server, c.Logger, err)
	}

	conn := chshare.NewWebSocketConn(wsConn)

	// perform SSH handshake on net.Conn
	c.Debugf("Handshaking...")
	sshClientConn, chans, reqs, err := ssh.NewClientConn(conn, "", c.sshConfig)
	if err != nil {
		if strings.Contains(err.Error(), "unable to authenticate") {
			c.Errorf("Authentication failed")
			return nil, err
		}
		return nil, err
	}

	return &sshClientConnection{
		Connection: sshClientConn,
		Requests:   reqs,
		Channels:   chans,
	}, nil
}

func (c *Client) setupDialer() (d *websocket.Dialer, netDialer *net.Dialer, err error) {
	netDialer = &net.Dialer{}
	d = &websocket.Dialer{
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: DialTimeout,
		Subprotocols:     []string{chshare.ProtocolVersion},
		NetDialContext:   netDialer.DialContext,
	}
	if c.configHolder.Client.BindInterface != "" {
		laddr, err := c.localAddrForInterface(c.configHolder.Client.BindInterface)
		if err != nil {
			return nil, nil, err
		}
		netDialer.LocalAddr = laddr
	}

	return d, netDialer, err
}

func (c *Client) addDialerProxySupport(d *websocket.Dialer, netDialer *net.Dialer) (err error) {
	if strings.HasPrefix(c.configHolder.Client.ProxyURL.Scheme, "socks") {
		// SOCKS5 proxy
		if c.configHolder.Client.ProxyURL.Scheme != "socks" && c.configHolder.Client.ProxyURL.Scheme != "socks5h" {
			return fmt.Errorf(
				"unsupported socks proxy type: %s:// (only socks5h:// or socks:// is supported)",
				c.configHolder.Client.ProxyURL.Scheme)
		}
		var auth *proxy.Auth
		if c.configHolder.Client.ProxyURL.User != nil {
			pass, _ := c.configHolder.Client.ProxyURL.User.Password()
			auth = &proxy.Auth{
				User:     c.configHolder.Client.ProxyURL.User.Username(),
				Password: pass,
			}
		}
		socksDialer, err := proxy.SOCKS5("tcp", c.configHolder.Client.ProxyURL.Host, auth, netDialer)
		if err != nil {
			return err
		}
		d.NetDialContext = socksDialer.(proxy.ContextDialer).DialContext
	} else {
		// CONNECT proxy
		d.Proxy = func(*http.Request) (*url.URL, error) {
			return c.configHolder.Client.ProxyURL, nil
		}
	}

	return nil
}

type sendResponse struct {
	replyOk   bool
	respBytes []byte
	rtt       time.Duration
}

func canRetryFn(err error) (can bool) {
	// if a timeout err, retry on the existing connection
	return strings.Contains(err.Error(), "timeout")
}

func (c *Client) sendConnectionRequest(ctx context.Context, sshConn ssh.Conn, minRetryWaitDuration time.Duration) error {
	connReq, err := c.connectionRequest(ctx)
	if err != nil {
		return err
	}

	req, err := chshare.EncodeConnectionRequest(connReq)
	if err != nil {
		return fmt.Errorf("could not encode connection request: %v", err)
	}

	c.Infof("Sending connection request.")
	c.Debugf("Sending connection request with client details %s", string(req))
	t0 := time.Now()
	res, err := comm.WithRetry(func() (res *sendResponse, err error) {
		replyOk, respBytes, err := comm.SendRequestWithTimeout(ctx, sshConn, "new_connection", true, req, SendRequestTimeout, c.Logger)
		return &sendResponse{
			replyOk:   replyOk,
			respBytes: respBytes,
		}, err
	}, canRetryFn, minRetryWaitDuration, "Connection Request", c.Logger)

	if err != nil {
		c.Errorf("connection request err = %v", err)
		c.Errorf("closing sshConn")
		if closeErr := sshConn.Close(); closeErr != nil {
			c.Errorf("Failed to close connection: %s", closeErr)
		}
		reconnect := strings.Contains(err.Error(), "reconnect")
		if reconnect {
			reconnectDelay := ServerReconnectRequestBackoffTime + (time.Duration(rand.Intn(30)) * time.Second)
			c.Debugf("reconnect requested. waiting %d seconds before retrying.", reconnectDelay/time.Second)
			// this probably means the server is too busy for us. wait quite a while
			// before returning to the conn loop.
			time.Sleep(reconnectDelay)
		}
		return err
	}

	c.Debugf("Connection request has been answered within %s.", time.Since(t0))
	if !res.replyOk {
		msg := string(res.respBytes)

		// if replied with client credentials already used - retry
		if strings.Contains(msg, "client is already connected:") {
			if closeErr := sshConn.Close(); closeErr != nil {
				c.Errorf(closeErr.Error())
			}
			return errors.New("client is already connected or previous session was not properly closed")
		}

		return errors.New(msg)
	}

	var remotes []*models.Remote
	err = json.Unmarshal(res.respBytes, &remotes)
	if err != nil {
		return fmt.Errorf("can't decode reply payload: %s", err)
	}

	msg := fmt.Sprintf("Connected to %s within %s", sshConn.RemoteAddr().String(), time.Since(t0))
	c.watchdog.Ping(WatchdogStateConnected, msg)
	c.Infof(msg)

	for _, r := range remotes {
		c.Infof("New tunnel: %s", r.String())

		serverStr := r.Local()
		if r.HTTPProxy {
			serverStr = "https://" + serverStr
		}

		c.Infof("Local port %s has become available on %s", r.Remote(), serverStr)
	}

	return nil
}

// afterPutCapabilities is the place to do things dependent on server capabilities
func (c *Client) afterPutCapabilities(ctx context.Context) {
	if c.serverCapabilities.MonitoringVersion > 0 {
		c.monitor.Start(ctx)
	} else {
		c.Debugf("Server has no monitoring capability, measurement not started")
	}

	if c.serverCapabilities.IPAddressesVersion > 0 {
		c.ipAddressesFetcher.Start(ctx)
	} else {
		c.Debugf("Server has no Fetcher capability, fetching not started")
	}
}

func (c *Client) handlePutCapabilitiesRequest(ctx context.Context, payload []byte) {
	caps := &models.Capabilities{}
	if err := json.Unmarshal(payload, caps); err != nil {
		c.Errorf("failed to decode %T: %v", caps, err)
		return
	}
	c.Debugf("Server has capabilities: %s", string(payload))
	c.serverCapabilities = caps
	c.afterPutCapabilities(ctx)
}

func (c *Client) handleSSHRequests(ctx context.Context, sshClientConn *sshClientConnection) {
	c.Logger.Debugf("handleSSHRequests started")

	for r := range sshClientConn.Requests {
		c.Logger.Debugf("handling request: %s", r.Type)
		c.Logger.Debugf("payload: %v", string(r.Payload))
		var err error
		var resp interface{}
		switch r.Type {

		case comm.RequestTypeUpdateClientAttributes:
			resp, err = c.updateAttributes(r.Payload)
		case comm.RequestTypeCheckPort:
			resp, err = checkPort(r.Payload)
			// fall through for err and resp handling
		case comm.RequestTypeRunCmd:
			resp, err = c.HandleRunCmdRequest(ctx, r.Payload)
			// fall through for err and resp handling
		case comm.RequestTypeRefreshUpdatesStatus:
			c.updates.Refresh()
			// fall through to reply success with empty resp
		case comm.RequestTypePutCapabilities:
			c.handlePutCapabilitiesRequest(ctx, r.Payload)
			// fall through to reply success with empty resp
		case comm.RequestTypeUpload:
			uploadManager := NewSSHUploadManager(
				c.Logger,
				c.filesAPI,
				c.configHolder,
				sshClientConn.Connection,
				system.SysUserProvider{},
			)
			resp, err = uploadManager.HandleUploadRequest(r.Payload)
			// fall through for err and resp handling
		case comm.RequestTypeCheckTunnelAllowed:
			resp, err = c.checkTunnelAllowed(r.Payload)
			// fall through for err and resp handling
		case comm.RequestTypePing:
			// use empty reply (and NOT empty resp with success reply)
			_ = r.Reply(true, nil)
			continue
		default:
			c.Debugf("Unknown request: %q", r.Type)
			comm.ReplyError(c.Logger, r, errors.New("unknown request"))
			continue
		}

		if err != nil {
			c.Errorf("Failed to handle %q request: %v", r.Type, err)
			comm.ReplyError(c.Logger, r, err)
			continue
		}

		comm.ReplySuccessJSON(c.Logger, r, resp)
	}

	c.Logger.Debugf("handleSSHRequests finished")
}

func checkPort(payload []byte) (*comm.CheckPortResponse, error) {
	req, err := comm.DecodeCheckPortRequest(payload)
	if err != nil {
		return nil, err
	}

	open, checkErr := system.IsPortOpen(req.HostPort, req.Timeout)
	var errMsg string
	if checkErr != nil {
		errMsg = checkErr.Error()
	}
	return &comm.CheckPortResponse{
		Open:   open,
		ErrMsg: errMsg,
	}, nil
}

func (c *Client) checkTunnelAllowed(payload []byte) (*comm.CheckTunnelAllowedResponse, error) {
	var req comm.CheckTunnelAllowedRequest
	err := json.Unmarshal(payload, &req)
	if err != nil {
		return nil, err
	}

	allowed, err := TunnelIsAllowed(c.configHolder.Client.TunnelAllowed, req.Remote)
	if err != nil {
		return nil, err
	}
	if !allowed {
		c.Errorf(`Tunnel to %q not allowed based on "tunnel_allowed" config: %v`, req.Remote, c.configHolder.Client.TunnelAllowed)
	}

	return &comm.CheckTunnelAllowedResponse{
		IsAllowed: allowed,
	}, nil
}

// Wait blocks while the client is running.
// Can only be called once.
func (c *Client) Wait(ctx context.Context) (err error) {
	select {
	case <-c.runningc:
	case <-ctx.Done():
		c.Logger.Debugf("context canceled during client wait")
		err = ctx.Err()
	}
	return err
}

// Close manually stops the client
func (c *Client) Close() error {
	c.stopRunning()
	c.watchdog.Close()
	return c.CloseConnection()
}

func (c *Client) CloseConnection() error {
	sshConn := c.getConn()
	if sshConn == nil {
		return nil
	}
	return sshConn.Close()
}

func (c *Client) isRunning() (isRunning bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

func (c *Client) stopRunning() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
}

func (c *Client) connectStreams(chans <-chan ssh.NewChannel) {
	c.Logger.Debugf("connectStreams started")
	for ch := range chans {
		remote := string(ch.ExtraData())
		protocol := models.ProtocolTCP
		c.Debugf("handling connect stream: remote=%s, protocol=%s", remote, protocol)
		parts := strings.SplitN(remote, "/", 2)
		if len(parts) == 2 {
			remote = parts[0]
			protocol = parts[1]
		}

		allowed, err := TunnelIsAllowed(c.configHolder.Client.TunnelAllowed, remote)
		if err != nil {
			c.Errorf("Could not check if remote is allowed: %v", err)
		}
		if !allowed {
			c.Errorf(`Rejecting stream to %q based on "tunnel_allowed" config: %v`, remote, c.configHolder.Client.TunnelAllowed)
			err := ch.Reject(ssh.Prohibited, `not allowed with "tunnel_allowed" config`)
			if err != nil {
				c.Errorf("Failed to reject stream: %v", err)
			}
			continue
		}

		stream, reqs, err := ch.Accept()
		if err != nil {
			c.Debugf("Failed to accept stream: %s", err)
			continue
		}
		go ssh.DiscardRequests(reqs)

		switch protocol {
		case models.ProtocolTCP:
			l := c.Logger.Fork("tcp conn#%d", c.connStats.New())
			go chshare.HandleTCPStream(l, &c.connStats, stream, remote)
		case models.ProtocolUDP:
			go func() {
				err := newUDPHandler(c.Logger.Fork("udp#%s", remote), remote).Handle(stream)
				if err != nil {
					c.Errorf("Error with UDP: %v", err)
				}
			}()
		default:
			c.Errorf("Unsupported protocol %v for tunnel %v", protocol, remote)
			stream.Close()
		}
	}
	c.Logger.Debugf("connectStreams finished")
}

// returns all local ipv4, ipv6 addresses
func (c *Client) localIPAddresses() ([]string, []string, error) {
	ipv4 := []string{}
	ipv6 := []string{}

	addrs, err := c.systemInfo.InterfaceAddrs()
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
	return ipv4, ipv6, nil
}

func (c *Client) connectionRequest(ctx context.Context) (*chshare.ConnectionRequest, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	connReq := &chshare.ConnectionRequest{
		ID:                     c.configHolder.Client.ID,
		Name:                   c.configHolder.Client.Name,
		SessionID:              c.SessionID,
		Tags:                   c.configHolder.Client.Tags,
		Labels:                 c.configHolder.Client.Labels,
		Remotes:                c.configHolder.Client.Tunnels,
		OS:                     system.UnknownValue,
		OSArch:                 c.systemInfo.GoArch(),
		OSKernel:               system.UnknownValue,
		OSFamily:               system.UnknownValue,
		OSVersion:              system.UnknownValue,
		OSVirtualizationRole:   system.UnknownValue,
		OSVirtualizationSystem: system.UnknownValue,
		Version:                chshare.BuildVersion,
		Hostname:               system.UnknownValue,
		CPUFamily:              system.UnknownValue,
		CPUModel:               system.UnknownValue,
		CPUModelName:           system.UnknownValue,
		CPUVendor:              system.UnknownValue,
		ClientConfiguration:    c.configHolder.Config,
	}

	var err error
	if connReq.ID == "" && c.configHolder.Client.UseSystemID {
		connReq.ID, err = machineid.ID()
		if err != nil {
			return nil, fmt.Errorf("could not use system id as client id: try to set client.id manually or disable client.use_system_id. Error: %w", err)
		}
	}

	if connReq.Name == "" && c.configHolder.Client.UseHostname {
		connReq.Name, err = c.systemInfo.Hostname()
		if err != nil {
			return nil, fmt.Errorf("could not use system hostname as client name: try to set client.name manually or disable client.use_hostname. Error: %w", err)
		}
	}

	info, err := c.systemInfo.HostInfo(ctx)
	if err != nil {
		c.Logger.Errorf("Could not get os information: %v", err)
	} else {
		connReq.OSKernel = info.OS
		connReq.OSFamily = info.PlatformFamily
	}

	os, err := c.getOS(ctx, info)
	if err != nil {
		c.Logger.Errorf("Could not get os name: %v", err)
	} else {
		connReq.OS = os
	}

	connReq.OSFullName = c.getOSFullName(info)
	if info != nil && info.PlatformVersion != "" {
		connReq.OSVersion = info.PlatformVersion
	}

	oSVirtualizationSystem, oSVirtualizationRole, err := c.systemInfo.VirtualizationInfo(ctx)
	if err != nil {
		c.Logger.Errorf("Could not get OS Virtualization Info: %v", err)
	} else {
		connReq.OSVirtualizationSystem = oSVirtualizationSystem
		connReq.OSVirtualizationRole = oSVirtualizationRole
	}

	connReq.IPv4, connReq.IPv6, err = c.localIPAddresses()
	if err != nil {
		c.Logger.Errorf("Could not get local ips: %v", err)
	}

	hostname, err := c.systemInfo.Hostname()
	if err != nil {
		c.Logger.Errorf("Could not get hostname: %v", err)
	} else {
		connReq.Hostname = hostname
	}

	cpuInfo, err := c.systemInfo.CPUInfo(ctx)

	if err != nil {
		c.Logger.Errorf("Could not get cpu information: %v", err)
	}

	if len(cpuInfo.CPUs) > 0 {
		connReq.CPUFamily = cpuInfo.CPUs[0].Family
		connReq.CPUModel = cpuInfo.CPUs[0].Model
		connReq.CPUModelName = cpuInfo.CPUs[0].ModelName
		connReq.CPUVendor = cpuInfo.CPUs[0].VendorID
	}
	connReq.NumCPUs = cpuInfo.NumCores

	memoryInfo, err := c.systemInfo.MemoryStats(ctx)
	if err != nil {
		c.Logger.Errorf("Could not get memory information: %v", err)
	} else if memoryInfo != nil {
		connReq.MemoryTotal = memoryInfo.Total
	}

	connReq.Timezone = c.getTimezone()

	return connReq, nil
}

func (c *Client) getOS(ctx context.Context, info *host.InfoStat) (string, error) {
	if info == nil {
		return system.UnknownValue, nil
	} else if info.OS == "windows" {
		return info.Platform + " " + info.PlatformVersion + " " + info.PlatformFamily, nil
	}
	return c.systemInfo.Uname(ctx)
}

func (c *Client) getOSFullName(infoStat *host.InfoStat) string {
	if infoStat == nil {
		return system.UnknownValue
	}

	return fmt.Sprintf("%s %s", strings.Title(strings.ToLower(infoStat.Platform)), infoStat.PlatformVersion)
}

func (c *Client) getTimezone() string {
	return c.systemInfo.SystemTime().Format("MST (UTC-07:00)")
}

func (c *Client) localAddrForInterface(ifaceName string) (net.Addr, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to find %s", ifaceName)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get address for %s", ifaceName)
	}
	var selected net.IP
	for _, a := range addrs {
		ip, _, err := net.ParseCIDR(a.String())
		if err != nil {
			return nil, err
		}
		if ip.IsUnspecified() {
			continue
		}
		selected = ip
		break
	}
	if selected == nil {
		return nil, errors.Errorf("no address found for %s", ifaceName)
	}
	laddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:0", selected))
	if err != nil {
		return nil, errors.Wrapf(err, "could not resolve tcp address for %s", ifaceName)
	}

	c.Infof("Connecting using %s (%s)", iface.Name, selected)

	return laddr, nil
}

func (c *Client) updateAttributes(payload []byte) (any, error) {
	c.mu.RLock()
	attributesFilePath := c.configHolder.Client.AttributesFilePath
	c.mu.RUnlock()

	if attributesFilePath == "" {
		return nil, fmt.Errorf("attributes file path not set")
	}

	configHolder := &models.Attributes{}
	err := json.Unmarshal(payload, configHolder)
	if err != nil {
		return nil, fmt.Errorf("payload unreadable: %v", err)
	}

	data, err := json.Marshal(configHolder)
	if err != nil {
		return nil, fmt.Errorf("can't serialize attributes: %v", err)
	}

	err = os.WriteFile(attributesFilePath, data, 0600)
	if err != nil {
		return nil, fmt.Errorf("can't write attributes to file: %v", err)
	}

	c.mu.Lock()
	c.configHolder.Client.Tags = configHolder.Tags
	c.configHolder.Client.Labels = configHolder.Labels
	c.mu.Unlock()

	return struct {
		Status string `json:"status"`
	}{Status: "OK"}, nil
}
