package chserver

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
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
	"github.com/cloudradar-monitoring/rport/server/sessions"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type ClientListener struct {
	*chshare.Logger

	sessionService *SessionService

	connStats         chshare.ConnStats
	httpServer        *chshare.HTTPServer
	reverseProxy      *httputil.ReverseProxy
	sshConfig         *ssh.ServerConfig
	sshSessionClients *clients.ClientCache // [ssh_session_id, Client] pairs, applicable only when auth is enabled
	allClients        *clients.ClientCache
	requestLogOptions *requestlog.Options
	maxRequestBytes   int64

	sessionIndexAutoIncrement int32
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func NewClientListener(config *Config, s *SessionService, allClients *clients.ClientCache, privateKey ssh.Signer) (*ClientListener, error) {
	cl := &ClientListener{
		sessionService:    s,
		httpServer:        chshare.NewHTTPServer(int(config.MaxRequestBytes)),
		allClients:        allClients,
		sshSessionClients: clients.NewEmptyClientCache(),
		Logger:            chshare.NewLogger("client-listener", config.LogOutput, config.LogLevel),
		requestLogOptions: config.InitRequestLogOptions(),
		maxRequestBytes:   config.MaxRequestBytes,
	}
	//create ssh config
	cl.sshConfig = &ssh.ServerConfig{
		ServerVersion: "SSH-" + chshare.ProtocolVersion + "-server",
	}
	if cl.allClients != nil {
		cl.sshConfig.PasswordCallback = cl.authUser
	}
	cl.sshConfig.AddHostKey(privateKey)
	//setup reverse proxy
	if config.Proxy != "" {
		u, err := url.Parse(config.Proxy)
		if err != nil {
			return nil, err
		}
		if u.Host == "" {
			return nil, cl.FormatError("Missing protocol (%s)", u)
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
	clientID := c.User()
	client := cl.allClients.Get(clientID)
	// constant time compare is used for security reasons
	if client == nil || subtle.ConstantTimeCompare([]byte(client.Password), password) != 1 {
		cl.Debugf("Login failed for client: %s", clientID)
		return nil, fmt.Errorf("invalid authentication for client: %s", clientID)
	}

	// add to session clients to retrieve him later by SSH session ID
	sshID := sessions.GetSessionID(c)
	cl.sshSessionClients.Set(sshID, client)

	return nil, nil
}

func (cl *ClientListener) Start(listenAddr string) error {
	if cl.reverseProxy != nil {
		cl.Infof("Reverse proxy enabled")
	}
	cl.Infof("Listening on %s...", listenAddr)

	h := http.Handler(middleware.MaxBytes(http.HandlerFunc(cl.handleClient), cl.maxRequestBytes))
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

func (cl *ClientListener) nextSessionIndex() int32 {
	return atomic.AddInt32(&cl.sessionIndexAutoIncrement, 1)
}

// handleWebsocket is responsible for handling the websocket connection
func (cl *ClientListener) handleWebsocket(w http.ResponseWriter, req *http.Request) {
	clog := cl.Fork("session#%d", cl.nextSessionIndex())
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
		failed(cl.FormatError("expecting connection request"))
		return
	}
	if len(r.Payload) > int(cl.maxRequestBytes) {
		failed(cl.FormatError("request data exceeds the limit of %d bytes, actual size: %d", cl.maxRequestBytes, len(r.Payload)))
		return
	}
	connRequest, err := chshare.DecodeConnectionRequest(r.Payload)
	if err != nil {
		failed(cl.FormatError("invalid connection request"))
		return
	}

	//print if client and server versions dont match
	if connRequest.Version != chshare.BuildVersion {
		v := connRequest.Version
		if v == "" {
			v = "<unknown>"
		}
		clog.Infof("Client version (%s) differs from server version (%s)",
			v, chshare.BuildVersion)
	}

	var sid string
	if connRequest.ID == "" {
		sid = sessions.GetSessionID(sshConn)
	} else {
		sid = connRequest.ID
	}

	// if session id is in use, deny connection
	session, err := cl.sessionService.GetByID(sid)
	if err != nil {
		failed(cl.FormatError("failed to get session by id `%s`", sid))
		return
	}
	if session != nil {
		if session.Disconnected == nil {
			failed(cl.FormatError("session id `%s` is already in use", sid))
			return
		}

		oldTunnels := GetTunnelsToReestablish(getRemotes(session.Tunnels), connRequest.Remotes)
		clog.Infof("Tunnels to create %d: %v", len(connRequest.Remotes), connRequest.Remotes)
		if len(oldTunnels) > 0 {
			clog.Infof("Old tunnels to re-establish %d: %v", len(oldTunnels), oldTunnels)
			connRequest.Remotes = append(connRequest.Remotes, oldTunnels...)
		}
	}

	// get the current client
	var client *clients.Client
	if cl.allClients != nil {
		sshID := sessions.GetSessionID(sshConn)
		client = cl.sshSessionClients.Get(sshID)
		// no need to keep it, remove
		cl.sshSessionClients.Delete(sshID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientSession, err := cl.sessionService.StartClientSession(ctx, sid, sshConn, connRequest, client, clog)
	if err != nil {
		failed(cl.FormatError("%s", err))
		return
	}

	cl.replyConnectionSuccess(r, connRequest.Remotes)

	sessionBanner := clientSession.Banner()
	clog.Debugf("Open %s", sessionBanner)
	go cl.handleSSHRequests(clog, reqs)
	go cl.handleSSHChannels(clog, chans)
	_ = sshConn.Wait()
	clog.Debugf("Close %s", sessionBanner)

	err = cl.sessionService.Terminate(clientSession)
	if err != nil {
		cl.Errorf("could not terminate client session: %s", err)
	}
}

func getRemotes(tunnels []*sessions.Tunnel) []*chshare.Remote {
	r := make([]*chshare.Remote, 0, len(tunnels))
	for _, t := range tunnels {
		r = append(r, &t.Remote)
	}
	return r
}

// GetTunnelsToReestablish returns old tunnels that should be re-establish taking into account new tunnels.
func GetTunnelsToReestablish(old, new []*chshare.Remote) []*chshare.Remote {
	if len(new) >= len(old) {
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
				if !oldMarked[i] && curOld.LocalPortRandom && curNew.Remote() == curOld.Remote() {
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

func (cl *ClientListener) handleSSHRequests(clientLog *chshare.Logger, reqs <-chan *ssh.Request) {
	for r := range reqs {
		switch r.Type {
		case "ping":
			_ = r.Reply(true, nil)
		default:
			clientLog.Debugf("Unknown request: %s", r.Type)
		}
	}
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
