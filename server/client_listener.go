package chserver

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jpillora/requestlog"
	"golang.org/x/crypto/ssh"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type ClientListener struct {
	*chshare.Logger

	sessionRepo *SessionRepository

	connStats          chshare.ConnStats
	fingerprint        string
	httpServer         *chshare.HTTPServer
	reverseProxy       *httputil.ReverseProxy
	sshConfig          *ssh.ServerConfig
	authenticatedUsers *chshare.Users
	users              *chshare.UserIndex

	sessionIndexAutoIncrement int32
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func NewClientListener(config *Config, s *SessionRepository) (*ClientListener, error) {
	cl := &ClientListener{
		sessionRepo:        s,
		httpServer:         chshare.NewHTTPServer(),
		authenticatedUsers: chshare.NewUsers(),
		Logger:             chshare.NewLogger("client-listener"),
	}
	cl.Info = true
	cl.Debug = config.Verbose
	cl.users = chshare.NewUserIndex(cl.Logger)
	if config.AuthFile != "" {
		if err := cl.users.LoadUsers(config.AuthFile); err != nil {
			return nil, err
		}
	}
	if config.Auth != "" {
		u := &chshare.User{Addrs: []*regexp.Regexp{chshare.UserAllowAll}}
		u.Name, u.Pass = chshare.ParseAuth(config.Auth)
		if u.Name != "" {
			cl.users.AddUser(u)
		}
	}
	//generate private key (optionally using seed)
	key, _ := chshare.GenerateKey(config.KeySeed)
	//convert into ssh.PrivateKey
	private, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatal("Failed to parse key")
	}
	//fingerprint this key
	cl.fingerprint = chshare.FingerprintKey(private.PublicKey())
	//create ssh config
	cl.sshConfig = &ssh.ServerConfig{
		ServerVersion:    "SSH-" + chshare.ProtocolVersion + "-server",
		PasswordCallback: cl.authUser,
	}
	cl.sshConfig.AddHostKey(private)
	//setup reverse proxy
	if config.Proxy != "" {
		var u *url.URL
		u, err = url.Parse(config.Proxy)
		if err != nil {
			return nil, err
		}
		if u.Host == "" {
			return nil, cl.Errorf("Missing protocol (%s)", u)
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
	// check if user authenication is enable and it not allow all
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
	cl.authenticatedUsers.Set(GetSessionID(c), user)
	return nil, nil
}

func (cl *ClientListener) Start(listenAddr string) error {
	cl.Infof("Fingerprint %s", cl.fingerprint)
	if cl.users.Len() > 0 {
		cl.Infof("User authentication enabled")
	}
	if cl.reverseProxy != nil {
		cl.Infof("Reverse proxy enabled")
	}
	cl.Infof("Listening on %s...", listenAddr)

	h := http.Handler(http.HandlerFunc(cl.handleClient))
	if cl.Debug {
		o := requestlog.DefaultOptions
		o.TrustProxy = true
		h = requestlog.WrapWith(h, o)
	}
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
		_ = r.Reply(false, []byte(err.Error()))
	}
	if r.Type != "config" {
		failed(cl.Errorf("expecting config request"))
		return
	}
	c, err := chshare.DecodeConfig(r.Payload)

	if err != nil {
		failed(cl.Errorf("invalid config"))
		return
	}

	//print if client and server versions dont match
	if c.Version != chshare.BuildVersion {
		v := c.Version
		if v == "" {
			v = "<unknown>"
		}
		clog.Infof("Client version (%s) differs from server version (%s)",
			v, chshare.BuildVersion)
	}

	// pull the users from the session map
	var user *chshare.User
	var sid string
	if c.ID == "" {
		sid = GetSessionID(sshConn)
	} else {
		sid = c.ID
	}

	// if session id is in use, deny connection
	session, _ := cl.sessionRepo.FindOne(sid)
	if session != nil {
		failed(cl.Errorf("session id `%s` is already in use", sid))
		return
	}

	if cl.users.Len() > 0 {
		user, _ = cl.authenticatedUsers.Get(sid)
		cl.authenticatedUsers.Del(sid)
	}

	//if user is provided, ensure they have
	//access to the desired remotes
	if user != nil {
		for _, remote := range c.Remotes {
			if !user.HasAccess(remote) {
				failed(cl.Errorf("access to requested address denied (%s:%s)", remote.LocalHost, remote.LocalPort))
				return
			}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sessionInfo := &ClientSession{
		ID:         sid,
		Name:       c.Name,
		Tags:       c.Tags,
		OS:         c.OS,
		Hostname:   c.Hostname,
		Version:    c.Version,
		IPv4:       c.IPv4,
		IPv6:       c.IPv6,
		Address:    sshConn.RemoteAddr().String(),
		Tunnels:    make([]*Tunnel, 0),
		Connection: sshConn,
		Context:    ctx,
		User:       user,
		Logger:     clog,
	}

	//set up reverse port forwarding
	for _, r := range c.Remotes {
		_, err = sessionInfo.StartRemoteTunnel(r)
		if err != nil {
			failed(cl.Errorf("%s", err))
			return
		}
	}

	err = cl.sessionRepo.Save(sessionInfo)
	if err != nil {
		failed(cl.Errorf("%s", err))
		return
	}

	//success!
	_ = r.Reply(true, nil)

	sessionBanner := sessionInfo.banner()
	clog.Debugf("Open %s", sessionBanner)
	go cl.handleSSHRequests(clog, reqs)
	go cl.handleSSHChannels(clog, chans)
	_ = sshConn.Wait()
	clog.Debugf("Close %s", sessionBanner)

	err = cl.sessionRepo.Delete(sessionInfo)
	if err != nil {
		cl.Debugf("could not delete session from repo: %s", err)
	}
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
