package chserver

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jpillora/requestlog"
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/api/middleware"
	"github.com/cloudradar-monitoring/rport/server/clients"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/security"
)

type ClientListener struct {
	*chshare.Logger
	*Server

	connStats         chshare.ConnStats
	httpServer        *chshare.HTTPServer
	reverseProxy      *httputil.ReverseProxy
	sshConfig         *ssh.ServerConfig
	requestLogOptions *requestlog.Options
	bannedClientAuths *security.BanList
	bannedIPs         *security.MaxBadAttemptsBanList

	clientIndexAutoIncrement int32
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func NewClientListener(server *Server, privateKey ssh.Signer) (*ClientListener, error) {
	config := server.config
	cl := &ClientListener{
		Server:            server,
		httpServer:        chshare.NewHTTPServer(int(config.Server.MaxRequestBytes)),
		Logger:            chshare.NewLogger("client-listener", config.Logging.LogOutput, config.Logging.LogLevel),
		requestLogOptions: config.InitRequestLogOptions(),
		bannedClientAuths: security.NewBanList(time.Duration(config.Server.ClientLoginWait) * time.Second),
	}

	if config.Server.MaxFailedLogin > 0 && config.Server.BanTime > 0 {
		cl.bannedIPs = security.NewMaxBadAttemptsBanList(
			config.Server.MaxFailedLogin,
			time.Duration(config.Server.BanTime)*time.Second,
			cl.Logger,
		)
	}

	//create ssh config
	cl.sshConfig = &ssh.ServerConfig{
		ServerVersion:    "SSH-" + chshare.ProtocolVersion + "-server",
		PasswordCallback: cl.authUser,
	}
	cl.sshConfig.AddHostKey(privateKey)
	//setup reverse proxy
	if config.Server.Proxy != "" {
		u, err := url.Parse(config.Server.Proxy)
		if err != nil {
			return nil, err
		}
		if u.Host == "" {
			return nil, fmt.Errorf("missing protocol: %s", u)
		}
		cl.reverseProxy = httputil.NewSingleHostReverseProxy(u)
		//always use proxy host
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
		cl.Infof("Failed login attempt for client auth id %q, forcing to wait for %vs (%s)",
			clientAuthID,
			cl.config.Server.ClientLoginWait,
			cl.getIP(c.RemoteAddr()),
		)
		return nil, ErrTooManyRequests
	}

	clientAuth, err := cl.clientAuthProvider.Get(clientAuthID)
	if err != nil {
		return nil, err
	}

	ip := cl.getIP(c.RemoteAddr())
	// constant time compare is used for security reasons
	if clientAuth == nil || subtle.ConstantTimeCompare([]byte(clientAuth.Password), password) != 1 {
		cl.Debugf("Login failed for client auth id: %s", clientAuthID)
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
		cl.Errorf("failed to split host port for %q: %v", addr, err)
		return addrStr
	}
	return host
}

func (cl *ClientListener) Start(listenAddr string) error {
	if cl.reverseProxy != nil {
		cl.Infof("Reverse proxy enabled")
	}
	cl.Infof("Listening on %s...", listenAddr)

	h := http.Handler(middleware.MaxBytes(http.HandlerFunc(cl.handleClient), cl.config.Server.MaxRequestBytesClient))
	if cl.bannedIPs != nil {
		h = http.Handler(security.RejectBannedIPs(h, cl.bannedIPs))
	}
	h = requestlog.WrapWith(h, *cl.requestLogOptions)
	return cl.httpServer.GoListenAndServe(listenAddr, h)
}

// Wait waits for the http server to close
func (cl *ClientListener) Wait() error {
	return cl.httpServer.Wait()
}

// Close forcibly closes the http server
func (cl *ClientListener) Close() error {
	return cl.httpServer.Close()
}

func (cl *ClientListener) handleClient(w http.ResponseWriter, r *http.Request) {
	//websockets upgrade AND has rport prefix
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))
	protocol := r.Header.Get("Sec-WebSocket-Protocol")
	if upgrade == "websocket" && strings.HasPrefix(protocol, "rport-") {
		if protocol == chshare.ProtocolVersion {
			cl.handleWebsocket(w, r)
			return
		}
		//print into server logs and silently fall-through
		cl.Infof("ignored client connection using protocol '%s', expected '%s'",
			protocol, chshare.ProtocolVersion)
	}
	//proxy target was provided
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

// handleWebsocket is responsible for handling the websocket connection
func (cl *ClientListener) handleWebsocket(w http.ResponseWriter, req *http.Request) {
	clog := cl.Fork("client#%d", cl.nextClientIndex())
	wsConn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		clog.Debugf("Failed to upgrade (%s)", err)
		return
	}
	conn := chshare.NewWebSocketConn(wsConn)
	// perform SSH handshake on net.Conn
	clog.Debugf("Handshaking...")
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, cl.sshConfig)
	if err != nil {
		cl.Debugf("Failed to handshake (%s)", err)
		return
	}
	//verify configuration
	clog.Debugf("Verifying configuration")
	//wait for request, with timeout
	var r *ssh.Request
	select {
	case r = <-reqs:
	case <-time.After(10 * time.Second):
		_ = sshConn.Close()
		return
	}
	failed := func(err error) {
		clog.Debugf("Failed: %s", err)
		cl.replyConnectionError(r, err)
	}
	if r.Type != "new_connection" {
		failed(errors.New("expecting connection request"))
		return
	}
	if len(r.Payload) > int(cl.config.Server.MaxRequestBytesClient) {
		failed(fmt.Errorf("request data exceeds the limit of %d bytes, actual size: %d", cl.config.Server.MaxRequestBytesClient, len(r.Payload)))
		return
	}
	connRequest, err := chshare.DecodeConnectionRequest(r.Payload)
	if err != nil {
		failed(fmt.Errorf("invalid connection request: %s", err))
		return
	}

	checkVersions(clog, connRequest.Version)

	// get the current client auth id
	clientAuthID := sshConn.User()

	// client id
	cid, err := cl.getCID(connRequest.ID, cl.config, clientAuthID)
	if err != nil {
		failed(fmt.Errorf("could not get cid: %s", err))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := cl.clientService.StartClient(ctx, clientAuthID, cid, sshConn, cl.config.Server.AuthMultiuseCreds, connRequest, clog)
	if err != nil {
		failed(err)
		return
	}

	cl.replyConnectionSuccess(r, connRequest.Remotes)

	clientBanner := client.Banner()
	clog.Debugf("Open %s", clientBanner)
	go cl.handleSSHRequests(clog, cid, reqs)
	go cl.handleSSHChannels(clog, chans)
	_ = sshConn.Wait()
	clog.Debugf("Close %s", clientBanner)

	err = cl.clientService.Terminate(client)
	if err != nil {
		cl.Errorf("could not terminate client: %s", err)
	}
}

// checkVersions print if client and server versions dont match.
func checkVersions(log *chshare.Logger, clientVersion string) {
	if clientVersion == chshare.BuildVersion {
		return
	}

	v := clientVersion
	if v == "" {
		v = "<unknown>"
	}

	log.Infof("Client version (%s) differs from server version (%s)", v, chshare.BuildVersion)
}

func (cl *ClientListener) getCID(reqID string, config *Config, clientAuthID string) (string, error) {
	if reqID != "" {
		return reqID, nil
	}

	// use client auth id as client id if proper configs are set
	if !config.Server.AuthMultiuseCreds && config.Server.EquateClientauthidClientid {
		return clientAuthID, nil
	}

	return clients.NewClientID()
}

func getRemotes(tunnels []*clients.Tunnel) []*chshare.Remote {
	r := make([]*chshare.Remote, 0, len(tunnels))
	for _, t := range tunnels {
		r = append(r, &t.Remote)
	}
	return r
}

// GetTunnelsToReestablish returns old tunnels that should be re-establish taking into account new tunnels.
func GetTunnelsToReestablish(old, new []*chshare.Remote) []*chshare.Remote {
	if len(new) > len(old) {
		return nil
	}

	// check if old tunnels contain all new tunnels
	// NOTE: old tunnels contain random port if local was not specified
	oldMarked := make([]bool, len(old))

	// at first check new with local specified. It's done at first to cover a case when a new tunnel was specified
	// with a port that is among random ports in old tunnels.
loop1:
	for _, curNew := range new {
		if curNew.IsLocalSpecified() {
			for i, curOld := range old {
				if !oldMarked[i] && curNew.String() == curOld.String() {
					oldMarked[i] = true
					continue loop1
				}
			}
			return nil
		}
	}

	// then check without local
loop2:
	for _, curNew := range new {
		if !curNew.IsLocalSpecified() {
			for i, curOld := range old {
				if !oldMarked[i] && curOld.LocalPortRandom && curNew.Remote() == curOld.Remote() && curNew.EqualACL(curOld.ACL) {
					oldMarked[i] = true
					continue loop2
				}
			}
			return nil
		}
	}

	// add tunnels that left among old
	var res []*chshare.Remote
	for i, marked := range oldMarked {
		if !marked {
			r := *old[i]
			// if it was random then set up zero values
			if r.LocalPortRandom {
				r.LocalHost = ""
				r.LocalPort = ""
			}
			res = append(res, &r)
		}
	}

	return res
}

func (cl *ClientListener) replyConnectionSuccess(r *ssh.Request, remotes []*chshare.Remote) {
	replyPayload, err := json.Marshal(remotes)
	if err != nil {
		cl.Errorf("can't encode success reply payload")
		cl.replyConnectionError(r, err)
		return
	}

	_ = r.Reply(true, replyPayload)
}

func (cl *ClientListener) replyConnectionError(r *ssh.Request, err error) {
	_ = r.Reply(false, []byte(err.Error()))
}

func (cl *ClientListener) handleSSHRequests(clientLog *chshare.Logger, clientID string, reqs <-chan *ssh.Request) {
	for r := range reqs {
		if len(r.Payload) > int(cl.config.Server.MaxRequestBytesClient) {
			clientLog.Errorf("%s:request data exceeds the limit of %d bytes, actual size: %d", comm.RequestTypeSaveMeasurement, cl.config.Server.MaxRequestBytesClient, len(r.Payload))
			continue
		}
		switch r.Type {
		case comm.RequestTypePing:
			_ = r.Reply(true, nil)
		case comm.RequestTypeCmdResult:
			job, err := cl.saveCmdResult(r.Payload)
			if err != nil {
				clientLog.Errorf("Failed to save cmd result: %s", err)
				continue
			}
			clientLog.Debugf("%s, Command result saved successfully.", job.LogPrefix())

			if job.MultiJobID != nil {
				done := cl.jobsDoneChannel.Get(*job.MultiJobID)
				if done != nil {
					// to avoid blocking the exec - send job result in a new goroutine
					go func(done2 chan *models.Job, job2 *models.Job) {
						done2 <- job2
					}(done, job)
				}
			}
		case comm.RequestTypeUpdatesStatus:
			updatesStatus := &models.UpdatesStatus{}
			err := json.Unmarshal(r.Payload, updatesStatus)
			if err != nil {
				clientLog.Errorf("Failed to unmarshal updates status: %s", err)
				continue
			}
			err = cl.clientService.SetUpdatesStatus(clientID, updatesStatus)
			if err != nil {
				clientLog.Errorf("Failed to save updates status: %s", err)
				continue
			}
		case comm.RequestTypeSaveMeasurement:
			measurement := &models.Measurement{}
			err := json.Unmarshal(r.Payload, measurement)
			if err != nil {
				clientLog.Errorf("Failed to unmarshal save_measurement: %s", err)
				continue
			}
			measurement.ClientID = clientID
			err = cl.monitoringService.SaveMeasurement(context.Background(), measurement)
			if err != nil {
				clientLog.Errorf("Failed to save measurement for client %s: %s", clientID, err)
				continue
			}
		default:
			clientLog.Debugf("Unknown request: %s", r.Type)
		}
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
	ws := cl.Server.uiJobWebSockets.Get(wsJID)
	if ws != nil {
		err := ws.WriteMessage(websocket.TextMessage, respBytes)
		if err != nil {
			cl.Errorf("%s, failed to write message to UI Web Socket: %v", resp.LogPrefix(), err)
			// proceed further
		}
	} else {
		cl.Debugf("%s, WS conn not found", resp.LogPrefix())
	}

	err = cl.jobProvider.SaveJob(&resp)
	if err != nil {
		return nil, fmt.Errorf("failed to save job result: %s", err)
	}

	return &resp, nil
}

func (cl *ClientListener) handleSSHChannels(clientLog *chshare.Logger, chans <-chan ssh.NewChannel) {
	for ch := range chans {
		remote := string(ch.ExtraData())
		//accept rest
		stream, reqs, err := ch.Accept()
		if err != nil {
			clientLog.Debugf("Failed to accept stream: %s", err)
			continue
		}
		go ssh.DiscardRequests(reqs)
		//handle stream type
		connID := cl.connStats.New()
		go chshare.HandleTCPStream(clientLog.Fork("conn#%d", connID), &cl.connStats, stream, remote)
	}
}
