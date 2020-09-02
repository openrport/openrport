package chserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jpillora/requestlog"
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/sessions"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type ClientListener struct {
	*chshare.Logger

	sessionService *SessionService

	connStats          chshare.ConnStats
	httpServer         *chshare.HTTPServer
	reverseProxy       *httputil.ReverseProxy
	sshConfig          *ssh.ServerConfig
	authenticatedUsers *chshare.Users
	users              *chshare.UserIndex
	requestLogOptions  *requestlog.Options

	sessionIndexAutoIncrement int32
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func NewClientListener(config *Config, s *SessionService, privateKey ssh.Signer) (*ClientListener, error) {
	cl := &ClientListener{
		sessionService:     s,
		httpServer:         chshare.NewHTTPServer(),
		authenticatedUsers: chshare.NewUsers(),
		Logger:             chshare.NewLogger("client-listener", config.LogOutput, config.LogLevel),
		requestLogOptions:  config.InitRequestLogOptions(),
	}
	cl.users = chshare.NewUserIndex(cl.Logger)
	if config.AuthFile != "" {
		if err := cl.users.LoadUsers(config.AuthFile); err != nil {
			return nil, err
		}
	}
	if config.Auth != "" {
		u := &chshare.User{}
		u.Name, u.Pass = chshare.ParseAuth(config.Auth)
		if u.Name != "" {
			cl.users.AddUser(u)
		}
	}
	//create ssh config
	cl.sshConfig = &ssh.ServerConfig{
		ServerVersion:    "SSH-" + chshare.ProtocolVersion + "-server",
		PasswordCallback: cl.authUser,
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
	// check if user authentication is enable and it not allow all
	if cl.users.Len() == 0 {
		return nil, nil
	}
	// check the user exists and has matching password
	n := c.User()
	user, found := cl.users.Get(n)
	if !found || user.Pass != string(password) {
		cl.Debugf("Login failed for user: %s", n)
		return nil, errors.New("Invalid authentication for username: %s")
	}
	// insert the user session map
	// @note: this should probably have a lock on it given the map isn't thread-safe??
	cl.authenticatedUsers.Set(sessions.GetSessionID(c), user)
	return nil, nil
}

func (cl *ClientListener) Start(listenAddr string) error {
	if cl.users.Len() > 0 {
		cl.Infof("User authentication enabled")
	}
	if cl.reverseProxy != nil {
		cl.Infof("Reverse proxy enabled")
	}
	cl.Infof("Listening on %s...", listenAddr)

	h := http.Handler(http.HandlerFunc(cl.handleClient))
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
	session, err := cl.sessionService.GetActiveByID(sid)
	if err != nil {
		failed(cl.FormatError("failed to get session by id `%s`", sid))
		return
	}
	if session != nil {
		failed(cl.FormatError("session id `%s` is already in use", sid))
		return
	}

	// pull the users from the session map
	var user *chshare.User
	if cl.users.Len() > 0 {
		user, _ = cl.authenticatedUsers.Get(sid)
		cl.authenticatedUsers.Del(sid)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientSession, err := cl.sessionService.StartClientSession(ctx, sid, sshConn, connRequest, user, clog)
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
