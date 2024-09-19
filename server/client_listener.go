package chserver

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/sftp"

	"github.com/gorilla/websocket"
	"github.com/jpillora/requestlog"
	"golang.org/x/crypto/ssh"

	rportplus "github.com/openrport/openrport/plus"
	alertingcap "github.com/openrport/openrport/plus/capabilities/alerting"
	"github.com/openrport/openrport/plus/capabilities/alerting/transformers"
	"github.com/openrport/openrport/server/api/middleware"
	"github.com/openrport/openrport/server/auditlog"
	"github.com/openrport/openrport/server/chconfig"
	"github.com/openrport/openrport/server/clients"
	"github.com/openrport/openrport/server/clients/clientdata"
	chshare "github.com/openrport/openrport/share"
	"github.com/openrport/openrport/share/comm"
	"github.com/openrport/openrport/share/logger"
	"github.com/openrport/openrport/share/models"
	"github.com/openrport/openrport/share/security"
)

const (
	ConnectionRequestTimeOut = 5 * 60 * time.Second

	ClientRequestsLog     = "requests"
	ClientPingsLog        = "ping"
	ClientMeasurementsLog = "measurements"
)

var (
	ClientRequestsLogEnabled = true
	ClientPingsLogEnabled    = false
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type ClientListener struct {
	logger  *logger.Logger
	server  *Server
	ctx     context.Context
	stopped atomic.Bool

	connStats         chshare.ConnStats
	httpServer        *chshare.HTTPServer
	reverseProxy      *httputil.ReverseProxy
	sshConfig         *ssh.ServerConfig
	requestLogOptions *requestlog.Options
	bannedClientAuths *security.BanList
	bannedIPs         *security.MaxBadAttemptsBanList

	clientIndexAutoIncrement int32

	// semaphore used to limit concurrent pending SSH connections
	inprogressSSHHandshakes chan struct{}

	mu sync.RWMutex
}

func (cl *ClientListener) log() (l *logger.Logger) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.logger
}

func (cl *ClientListener) getCtx() (ctx context.Context) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.ctx
}

func (cl *ClientListener) getClientService() (cs clients.ClientService) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.server.clientService
}

func NewClientListener(server *Server, privateKey ssh.Signer) (*ClientListener, error) {
	config := server.config

	// semaphore to limit number of active pending SSH connections
	inprogressSSHHandshakes := make(chan struct{}, config.Server.MaxConcurrentSSHConnectionHandshakes)

	clog := logger.NewLogger("client-listener", config.Logging.LogOutput, config.Logging.LogLevel)
	cl := &ClientListener{
		server:                  server,
		httpServer:              chshare.NewHTTPServer(int(config.Server.MaxRequestBytesClient), clog),
		requestLogOptions:       config.InitRequestLogOptions(),
		bannedClientAuths:       security.NewBanList(time.Duration(config.Server.ClientLoginWait) * time.Second),
		inprogressSSHHandshakes: inprogressSSHHandshakes,
		logger:                  clog,
	}

	if config.Server.MaxFailedLogin > 0 && config.Server.BanTime > 0 {
		cl.bannedIPs = security.NewMaxBadAttemptsBanList(
			config.Server.MaxFailedLogin,
			time.Duration(config.Server.BanTime)*time.Second,
			cl.logger,
		)
	}

	// create ssh config
	cl.sshConfig = &ssh.ServerConfig{
		ServerVersion:    "SSH-" + chshare.ProtocolVersion + "-server",
		PasswordCallback: cl.authUser,
	}

	cl.sshConfig.AddHostKey(privateKey)

	// setup reverse proxy
	if config.Server.Proxy != "" {
		u, err := url.Parse(config.Server.Proxy)
		if err != nil {
			return nil, err
		}
		if u.Host == "" {
			return nil, fmt.Errorf("missing protocol: %s", u)
		}
		cl.reverseProxy = httputil.NewSingleHostReverseProxy(u)
		// always use proxy host
		cl.reverseProxy.Director = func(r *http.Request) {
			r.URL.Scheme = u.Scheme
			r.URL.Host = u.Host
			r.Host = u.Host
		}
	}

	return cl, nil
}

// authUser is responsible for validating the ssh user / password combination
func (cl *ClientListener) authUser(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	clientAuthID := c.User()

	if cl.bannedClientAuths.IsBanned(clientAuthID) {
		cl.log().Infof("Failed login attempt for client auth id %q, forcing to wait for %vs (%s)",
			clientAuthID,
			cl.server.config.Server.ClientLoginWait,
			cl.getIP(c.RemoteAddr()),
		)
		return nil, ErrTooManyRequests
	}

	clientAuth, err := cl.server.clientAuthProvider.Get(clientAuthID)
	if err != nil {
		return nil, err
	}

	ip := cl.getIP(c.RemoteAddr())
	// constant time compare is used for security reasons
	if clientAuth == nil || subtle.ConstantTimeCompare([]byte(clientAuth.Password), password) != 1 {
		cl.log().Debugf("Login failed for client auth id: %s", clientAuthID)
		cl.bannedClientAuths.Add(clientAuthID)
		if cl.bannedIPs != nil {
			cl.bannedIPs.AddBadAttempt(ip)
		}
		return nil, fmt.Errorf("invalid authentication for client auth id: %s", clientAuthID)
	}

	if cl.bannedIPs != nil {
		cl.bannedIPs.AddSuccessAttempt(ip)
	}
	return nil, nil
}

func (cl *ClientListener) getIP(addr net.Addr) string {
	addrStr := addr.String()
	host, _, err := net.SplitHostPort(addrStr)
	if err != nil {
		cl.log().Errorf("failed to split host port for %q: %v", addr, err)
		return addrStr
	}
	return host
}

func (cl *ClientListener) Start(ctx context.Context, listenAddr string) error {
	clLogger := cl.log()
	clLogger.Debugf("Client listener starting...")

	// save the server ctx for use when handling client ssh connections etc
	cl.ctx = ctx

	if cl.reverseProxy != nil {
		clLogger.Infof("Reverse proxy enabled")
	}
	clLogger.Infof("Listening on %s...", listenAddr)

	h := http.Handler(middleware.MaxBytes(http.HandlerFunc(cl.handleClient), cl.server.config.Server.MaxRequestBytesClient))
	if cl.bannedIPs != nil {
		h = security.RejectBannedIPs(cl.bannedIPs)(h)
	}
	h = requestlog.WrapWith(h, *cl.requestLogOptions)

	return cl.httpServer.GoListenAndServe(ctx, listenAddr, h)
}

// Wait waits for the http server to close
func (cl *ClientListener) Wait() error {
	cl.log().Debugf("client listener waiting...")
	err := cl.httpServer.Wait()
	cl.log().Debugf("client listener stopping...")
	cl.stopped.Store(true)
	return err
}

// Close forcibly closes the http server
func (cl *ClientListener) Close() error {
	err := cl.httpServer.Close()
	cl.stopped.Store(true)
	cl.log().Debugf("client listener stopped")
	return err
}

func (cl *ClientListener) handleClient(w http.ResponseWriter, r *http.Request) {
	cl.log().Debugf("Incoming client connection...")
	// websockets upgrade AND has rport prefix
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))
	protocol := r.Header.Get("Sec-WebSocket-Protocol")
	if upgrade == "websocket" && strings.HasPrefix(protocol, "rport-") {
		if protocol == chshare.ProtocolVersion {
			cl.handleWebsocket(w, r)
			return
		}
		// print into server logs and silently fall-through
		cl.log().Infof("ignored client connection using protocol '%s', expected '%s'",
			protocol, chshare.ProtocolVersion)
	}
	// proxy target was provided
	if cl.reverseProxy != nil {
		cl.reverseProxy.ServeHTTP(w, r)
		return
	}

	w.WriteHeader(404)
	_, _ = w.Write([]byte{})
}

func (cl *ClientListener) nextClientIndex() int32 {
	return atomic.AddInt32(&cl.clientIndexAutoIncrement, 1)
}

func (cl *ClientListener) acceptSSHConnection(w http.ResponseWriter, req *http.Request) (sshConn *ssh.ServerConn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request,
	clog *logger.DynamicLogger, err error) {

	// throttle concurrent connections
	// add to pending connections. will block if the chan is full
	cl.inprogressSSHHandshakes <- struct{}{}
	defer func() {
		// on handshake finished, remove from pending connections, which will allow another connection to take place
		<-cl.inprogressSSHHandshakes
	}()

	clog = logger.ForkToDynamicLogger(cl.log(), fmt.Sprintf("client#%d", cl.nextClientIndex()), true, false)
	clog.SetControl(ClientRequestsLog, ClientRequestsLogEnabled)
	clog.SetControl(ClientPingsLog, ClientPingsLogEnabled)

	clog.Debugf("Handling inbound web socket connection...")
	ts := time.Now()

	wsConn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		clog.Debugf("Failed to upgrade (%s)", err)
		return nil, nil, nil, nil, err
	}
	conn := chshare.NewWebSocketConn(wsConn)
	// perform SSH handshake on net.Conn
	clog.Debugf("SSH Handshaking...")
	sshConn, chans, reqs, err = ssh.NewServerConn(conn, cl.sshConfig)
	if err != nil {
		if strings.Contains(err.Error(), "unexpected EOF") {
			clog.Debugf("Failed to handshake (client closed connection? - %s) from %s", err, conn.RemoteAddr().String())
		} else {
			clog.Debugf("Failed to handshake (%s) from %s", err, conn.RemoteAddr().String())
		}
		return nil, nil, nil, nil, err
	}
	clog.Debugf("SSH Handshake finished after %s", time.Since(ts))

	return sshConn, chans, reqs, clog, err
}

func (cl *ClientListener) receiveClientConnectionRequest(sshConn *ssh.ServerConn, reqs <-chan *ssh.Request, clog *logger.DynamicLogger) (connRequest *chshare.ConnectionRequest, r *ssh.Request, err error) {
	pendingRequestTimer := time.NewTimer(ConnectionRequestTimeOut)

	select {
	case r = <-reqs:
		pendingRequestTimer.Stop()

	case <-cl.getCtx().Done():
		pendingRequestTimer.Stop()
		return nil, nil, cl.ctx.Err()

	case <-pendingRequestTimer.C:
		errMsg := fmt.Sprintf("connection request timeout exceeded %0.2f sec", ConnectionRequestTimeOut.Seconds())
		clog.Debugf(errMsg)
		closeErr := sshConn.Close()
		if closeErr != nil {
			clog.Debugf("error on SSH connection close: %s", closeErr)
		}
		return nil, nil, errors.New(errMsg)
	}

	// it seems that nil requests are possible, so return an error
	if r == nil {
		return nil, nil, errors.New("received nil request from client")
	}

	if r.Type != "new_connection" {
		return nil, nil, errors.New("expecting connection request")
	}

	if len(r.Payload) > int(cl.server.config.Server.MaxRequestBytesClient) {
		return nil, nil, fmt.Errorf("request data exceeds the limit of %d bytes, actual size: %d", cl.server.config.Server.MaxRequestBytesClient, len(r.Payload))
	}

	connRequest, err = chshare.DecodeConnectionRequest(r.Payload)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid connection request: %s", err)
	}

	return connRequest, r, nil
}

// handleWebsocket is responsible for handling the websocket connection
func (cl *ClientListener) handleWebsocket(w http.ResponseWriter, req *http.Request) {
	// keep the time from the initial client connection attempt
	ts1 := time.Now()

	sshConn, chans, reqs, clientLog, err := cl.acceptSSHConnection(w, req)
	if err != nil {
		return
	}

	// verify configuration
	clientLog.Debugf("Verifying configuration...")

	// first request to be received must be a connection request
	connRequest, r, err := cl.receiveClientConnectionRequest(sshConn, reqs, clientLog)
	if err != nil {
		cl.replyConnectionError(r, err)
		return
	}

	clientLog.Debugf("client version: %s", connRequest.Version)

	checkVersions(clientLog, connRequest.Version)

	// get the current client auth id
	clientAuthID := sshConn.User()

	clientID, err := cl.getClientID(connRequest.ID, cl.server.config, clientAuthID)
	if err != nil {
		cl.replyConnectionError(r, fmt.Errorf("could not get clientID: %s", err))
		return
	}

	ctx, cancel := context.WithCancel(cl.getCtx())
	defer cancel()

	client, err := cl.getClientService().StartClient(ctx, clientAuthID, clientID, sshConn, cl.server.config.Server.AuthMultiuseCreds, connRequest, clientLog.GetLogger())
	if err != nil {
		cl.replyConnectionError(r, err)
		return
	}
	clientLog.Debugf("Client service started for %s (%s) within %s", client.GetID(), client.GetName(), time.Since(ts1))

	ts2 := time.Now()

	cl.replyConnectionSuccess(r, connRequest.Remotes)
	cl.sendCapabilities(sshConn)
	// Now the client is fully connected and ready to create tunnels and execute command and scripts

	clientBanner := client.Banner()
	clientLog.Debugf("opened %s within %s", clientBanner, time.Since(ts2))

	// now run handler for other client requests and connections
	go cl.handleSSHRequests(clientLog, clientID, reqs)
	go cl.handleSSHChannels(clientLog.GetLogger(), chans)

	// wait until we're disconnected from the client
	if err = sshConn.Wait(); err != nil {
		clientLog.Debugf("sshConn.Wait() error: %s", err)
	}
	clientLog.Debugf("close %s", clientBanner)

	err = cl.getClientService().Terminate(client)
	if err != nil {
		cl.log().Errorf("could not terminate client: %s", err)
	}
}

// checkVersions print if client and server versions dont match.
func checkVersions(log *logger.DynamicLogger, clientVersion string) {
	if clientVersion == chshare.BuildVersion {
		return
	}

	v := clientVersion
	if v == "" {
		v = "<unknown>"
	}

	log.Infof("Client version (%s) differs from server version (%s)", v, chshare.BuildVersion)
}

func (cl *ClientListener) getClientID(reqID string, config *chconfig.Config, clientAuthID string) (string, error) {
	if reqID != "" {
		return reqID, nil
	}

	// use client auth id as client id if proper configs are set
	if !config.Server.AuthMultiuseCreds && config.Server.EquateClientauthidClientid {
		return clientAuthID, nil
	}

	return clientdata.NewClientID()
}

func (cl *ClientListener) replyConnectionSuccess(r *ssh.Request, remotes []*models.Remote) {
	replyPayload, err := json.Marshal(remotes)
	if err != nil {
		cl.log().Errorf("can't encode success reply payload")
		cl.replyConnectionError(r, err)
		return
	}

	err = r.Reply(true, replyPayload)
	if err != nil {
		cl.log().Errorf("error during connection success reply: %w", err)
	}
}

func (cl *ClientListener) replyConnectionError(r *ssh.Request, err error) {
	if r == nil {
		cl.log().Errorf("failed to send connection reply with error due to nil request")
		return
	}
	if err == nil {
		cl.log().Debugf("sending connection reply with nil error: %s", r.Type)
		err = r.Reply(false, nil)
		if err != nil {
			cl.log().Errorf("error during connection nil error reply: %w", err)
		}
		return
	}
	err = r.Reply(false, []byte(err.Error()))
	if err != nil {
		cl.log().Errorf("error during connection error reply: %w", err)
	}
}

func (cl *ClientListener) handleSSHRequests(clientLog *logger.DynamicLogger, clientID string, reqs <-chan *ssh.Request) {
	clientService := cl.getClientService()

	for r := range reqs {
		if cl.stopped.Load() {
			clientLog.Debugf("client listener has been stopped. stopping requests handler for %s", clientID)
			break
		}

		if len(r.Payload) > int(cl.server.config.Server.MaxRequestBytesClient) {
			clientLog.Errorf("%s:request data exceeds the limit of %d bytes, actual size: %d", comm.RequestTypeSaveMeasurement, cl.server.config.Server.MaxRequestBytesClient, len(r.Payload))
			continue
		}

		clientLog.NDebugf(ClientRequestsLog, "received request: %s from %s", r.Type, clientID)

		// TODO: (rs): these case handlers should be refactored into individual handling fns
		switch r.Type {

		// we shouldn't be receiving this. it means the client didn't receive the server's reply
		// to the previous connection request, so ask the client to reconnect.
		case "new_connection":
			clientLog.Debugf("received connection request on existing connection. asking the client to reconnect.")
			// IMPORTANT: the client is checking for the word "reconnect" in reply errors
			cl.replyConnectionError(r, errors.New("unexpected connection request. please reconnect"))
			client, err := clientService.GetRepo().GetActiveByID(clientID)
			if err != nil {
				clientLog.Debugf("unable to get client: %v", err)
				continue
			}
			if client == nil {
				clientLog.Debugf("client not found: %v", err)
				continue
			}
			clientLog.Debugf("terminating client due for reconnect")
			err = clientService.Terminate(client)
			if err != nil {
				clientLog.Debugf("failed to terminate client due for reconnect: %v", err)
			}
			continue

		case comm.RequestTypePing:
			var ts time.Time
			if ClientPingsLogEnabled {
				clientLog.NDebugf(ClientPingsLog, "ping received from: %s", clientID)
				ts = time.Now().UTC()
			}
			// ts := time.Now()
			_ = r.Reply(true, nil)
			err := clientService.SetLastHeartbeat(clientID, time.Now())
			if err != nil {
				clientLog.Errorf("Failed to save heartbeat: %s", err)
				continue
			}
			if ClientPingsLogEnabled {
				clientLog.NDebugf(ClientPingsLog, "ping for: %s done in %s", clientID, time.Since(ts))
			}

		case comm.RequestTypeCmdResult:
			clientLog.Debugf("saving command result from: %s", clientID)
			var ts time.Time
			if ClientRequestsLogEnabled {
				ts = time.Now().UTC()
			}

			job, err := cl.saveCmdResult(r.Payload)
			if err != nil {
				clientLog.Errorf("Failed to save cmd result: %s", err)
				continue
			}
			clientLog.Debugf("%s, Command result saved successfully.", job.LogPrefix())

			var auditLogEntry *auditlog.Entry
			if job.IsScript {
				auditLogEntry = cl.server.auditLog.Entry(auditlog.ApplicationClientScript, auditlog.ActionExecuteDone)
			} else {
				auditLogEntry = cl.server.auditLog.Entry(auditlog.ApplicationClientCommand, auditlog.ActionExecuteDone)
			}
			if job.MultiJobID != nil {
				auditLogEntry.WithID(*job.MultiJobID)
			} else {
				auditLogEntry.WithID(job.JID)
			}
			auditLogEntry.
				WithResponse(job).
				WithClientID(clientID).
				Save()

			if job.MultiJobID != nil {
				done := cl.server.jobsDoneChannel.Get(*job.MultiJobID)
				if done != nil {
					// to avoid blocking the exec - send job result in a new goroutine
					go func(done2 chan *models.Job, job2 *models.Job) {
						done2 <- job2
					}(done, job)
				}
			}
			if ClientRequestsLogEnabled {
				clientLog.NDebugf(ClientRequestsLog, "%s: command results request completed at %s in %s", clientID, time.Now().UTC(), time.Since(ts))
			}

		case comm.RequestTypeUpdatesStatus:
			clientLog.Debugf("setting updates status from: %s", clientID)
			var ts time.Time
			if ClientRequestsLogEnabled {
				ts = time.Now().UTC()
			}
			updatesStatus := &models.UpdatesStatus{}
			err := json.Unmarshal(r.Payload, updatesStatus)
			if err != nil {
				clientLog.Errorf("Failed to unmarshal updates status: %s", err)
				continue
			}
			err = clientService.SetUpdatesStatus(clientID, updatesStatus)
			if err != nil {
				clientLog.Errorf("Failed to save updates status: %s", err)
				continue
			}
			if ClientRequestsLogEnabled {
				clientLog.NDebugf(ClientRequestsLog, "%s: updates updated at %s in %s", clientID, time.Now().UTC(), time.Since(ts))
			}

		case comm.RequestTypeInventory:
			clientLog.Debugf("setting inventory from %s", clientID)
			var ts time.Time
			if ClientRequestsLogEnabled {
				ts = time.Now().UTC()
			}
			inventory := &models.Inventory{}
			err := json.Unmarshal(r.Payload, inventory)
			if err != nil {
				clientLog.Errorf("Failed to unmarshal inventory: %s", err)
				continue
			}
			err = clientService.SetInventory(clientID, inventory)
			if err != nil {
				clientLog.Errorf("Failed to save inventory: %s", err)
				continue
			}
			if ClientRequestsLogEnabled {
				clientLog.NDebugf(ClientRequestsLog, "%s: inventory updated at %s in %s", clientID, time.Now().UTC(), time.Since(ts))
			}

		case comm.RequestTypeSaveMeasurement:
			// if server monitoring is disabled then do not save measurements even if received
			if !cl.server.config.Monitoring.Enabled {
				clientLog.Errorf("Received measurement when monitoring disabled. Measurement not saved.")
				continue
			}

			measurement := models.Measurement{}
			err := json.Unmarshal(r.Payload, &measurement)
			if err != nil {
				clientLog.Errorf("Failed to unmarshal save_measurement: %s", err)
				continue
			}

			measurement.ClientID = clientID
			measurement.Timestamp = time.Now().UTC()

			cl.server.monitoringQueue.Notify(measurement)

			if rportplus.IsPlusEnabled(cl.server.config.PlusConfig) {
				alertingCap := cl.server.plusManager.GetAlertingCapabilityEx()
				if alertingCap != nil {
					cl.sendMeasurementToAlertingService(alertingCap, &measurement, clientLog)
				}
			}
		case comm.RequestTypeIPAddresses:
			clientLog.Debugf("IP addresses update received from: %s, payload: %s", clientID, r.Payload)
			IPAddresses := &models.IPAddresses{}
			err := json.Unmarshal(r.Payload, IPAddresses)
			if err != nil {
				clientLog.Errorf("Failed to unmarshal IP addresses: %s", err)
				continue
			}
			IPAddresses.UpdatedAt = time.Now().UTC()
			err = clientService.SetIPAddresses(clientID, IPAddresses)
			if err != nil {
				clientLog.Errorf("Failed to save IPAddresses status: %s", err)
				continue
			}
		default:
			clientLog.Debugf("Unknown request: %s", r.Type)
		}
	}

	clientLog.Debugf("Client listener for %s stopped", clientID)
}

func (cl *ClientListener) sendMeasurementToAlertingService(
	alertingCap alertingcap.CapabilityEx,
	measurement *models.Measurement,
	clientLog *logger.DynamicLogger) {

	m, err := transformers.TransformRportMeasurementToMeasure(measurement)
	if err != nil {
		clientLog.Debugf("Failed to transform measurement: %v", err)
		return
	}

	as := alertingCap.GetService()

	err = as.PutMeasurement(m)
	if err != nil {
		clientLog.Debugf("Failed to send measurement to the alerting service: %v", err)
		return
	}
}

func (cl *ClientListener) saveCmdResult(respBytes []byte) (*models.Job, error) {
	resp := models.Job{}
	err := json.Unmarshal(respBytes, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cmd result request: %s", err)
	}

	var wsJID string
	if resp.MultiJobID != nil {
		wsJID = *resp.MultiJobID
	} else {
		wsJID = resp.JID
	}
	ws := cl.server.uiJobWebSockets.Get(wsJID)
	if ws != nil {
		err := ws.WriteMessage(websocket.TextMessage, respBytes)
		if err != nil {
			cl.log().Errorf("%s, failed to write message to UI Web Socket: %v", resp.LogPrefix(), err)
			// proceed further
		}
	} else {
		cl.log().Debugf("%s, WS conn not found when saving command result. No active listeners connected", resp.LogPrefix())
	}

	err = cl.server.jobProvider.SaveJob(&resp)
	if err != nil {
		return nil, fmt.Errorf("failed to save job result: %s", err)
	}

	return &resp, nil
}

func (cl *ClientListener) handleSSHChannels(clientLog *logger.Logger, chans <-chan ssh.NewChannel) {
	for ch := range chans {
		ch := ch
		extraData := string(ch.ExtraData())
		stream, reqs, err := ch.Accept()
		if err != nil {
			clientLog.Debugf("Failed to accept stream: %s", err)
			continue
		}

		go func() {
			for req := range reqs {
				cl.handleReq(req, clientLog)
			}
		}()

		switch ch.ChannelType() {
		case "session":
			cl.handleSessionChannel(stream, clientLog)
		case models.ChannelStdout, models.ChannelStderr:
			go func() {
				err := cl.handleOutputChannel(ch.ChannelType(), ch.ExtraData(), clientLog, stream)
				if err != nil {
					clientLog.Errorf("Error handling output channel %s: %v", ch.ChannelType(), err)
				}
			}()
		default:
			// handle stream type
			connID := cl.connStats.New()
			go chshare.HandleTCPStream(clientLog.Fork("conn#%d", connID), &cl.connStats, stream, extraData)
		}
	}
}

type outputChannelData struct {
	JID        string            `json:"jid"`
	ClientID   string            `json:"client_id"`
	ClientName string            `json:"client_name"`
	Result     *models.JobResult `json:"result"`
}

func (cl *ClientListener) handleOutputChannel(typ string, jobData []byte, clientLog *logger.Logger, stream io.Reader) error {
	job := models.Job{}
	err := json.Unmarshal(jobData, &job)
	if err != nil {
		return err
	}

	var wsJID string
	if job.MultiJobID != nil {
		wsJID = *job.MultiJobID
	} else {
		wsJID = job.JID
	}

	ws := cl.server.uiJobWebSockets.Get(wsJID)

	ocd := outputChannelData{
		JID:        job.JID,
		ClientID:   job.ClientID,
		ClientName: job.ClientName,
	}

	data := make([]byte, 4096)
	for {
		n, err := stream.Read(data)
		if err != nil {
			if err == io.EOF {
				clientLog.Debugf("Output channel %s for %s stop: %v", typ, wsJID, err)
				break
			}
			return err
		}

		if ws != nil {
			switch typ {
			case models.ChannelStdout:
				ocd.Result = &models.JobResult{
					StdOut: string(data[:n]),
				}
			case models.ChannelStderr:
				ocd.Result = &models.JobResult{
					StdErr: string(data[:n]),
				}
			}
			err := ws.WriteNonFinalJSON(ocd)
			if err != nil {
				clientLog.Errorf("Failed to write message to UI Web Socket: %v", err)
				// proceed further
			}
		} else {
			clientLog.Debugf("WS conn not found handling output channel. No active listeners connected")
		}
	}
	return nil
}

func (cl *ClientListener) handleReq(req *ssh.Request, clientLog *logger.Logger) {
	ok := false
	switch req.Type {
	// https://datatracker.ietf.org/doc/html/draft-ietf-secsh-filexfer-02#section-2
	case "subsystem":
		if string(req.Payload[4:]) == "sftp" {
			ok = true
		}
	}
	if req.WantReply {
		err := req.Reply(ok, nil)
		if err != nil {
			clientLog.Errorf("Failed to send ssh reply: %v", err)
		}
	}
}

func (cl *ClientListener) handleSessionChannel(stream ssh.Channel, clientLog *logger.Logger) {
	server, err := sftp.NewServer(
		stream,
		sftp.ReadOnly(),
	)
	if err != nil {
		clientLog.Debugf("Failed to create sftp server: %s", err)
		return
	}

	if err := server.Serve(); err == io.EOF {
		e := server.Close()
		if e != nil {
			clientLog.Errorf("failed to close sftp server: %v", e)
		}
		clientLog.Debugf("sftp client exited session.")
	} else if err != nil {
		clientLog.Errorf("sftp server completed with error: %v", err)
	}
}

func (cl *ClientListener) sendCapabilities(conn *ssh.ServerConn) {
	payload, err := json.Marshal(cl.server.capabilities)
	if err != nil {
		cl.log().Errorf("can't encode capabilities payload")
		return
	}

	if _, _, err = conn.SendRequest(comm.RequestTypePutCapabilities, false, payload); err != nil {
		cl.log().Errorf("can't send capabilities: %v", err)
	}
}
