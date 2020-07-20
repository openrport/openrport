package chserver

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/ssh"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

// handleClientHandler is the main http websocket handler for the rport server
func (s *Server) handleClientHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			s.Infof("panic while handling client request: %s", err)
		}
	}()

	//websockets upgrade AND has rport prefix
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))
	protocol := r.Header.Get("Sec-WebSocket-Protocol")
	if upgrade == "websocket" && strings.HasPrefix(protocol, "rport-") {
		if protocol == chshare.ProtocolVersion {
			s.handleWebsocket(w, r)
			return
		}
		//print into server logs and silently fall-through
		s.Infof("ignored client connection using protocol '%s', expected '%s'",
			protocol, chshare.ProtocolVersion)
	}
	//proxy target was provided
	if s.reverseProxy != nil {
		s.reverseProxy.ServeHTTP(w, r)
		return
	}

	//no proxy defined, provide access to REST API
	s.handleAPIRequest(w, r)
}

func (s *Server) handleAPIRequest(w http.ResponseWriter, req *http.Request) {
	var matchedRoute mux.RouteMatch
	routeExists := s.apiRouter.Match(req, &matchedRoute)
	if routeExists {
		req = mux.SetURLVars(req, matchedRoute.Vars) // allows retrieving Vars later from request object
		matchedRoute.Handler.ServeHTTP(w, req)
		return
	}
	w.WriteHeader(404)
	_, _ = w.Write([]byte{})
}

// handleWebsocket is responsible for handling the websocket connection
func (s *Server) handleWebsocket(w http.ResponseWriter, req *http.Request) {
	id := atomic.AddInt32(&s.sessionIDAutoIncrement, 1)
	clog := s.Fork("session#%d", id)
	wsConn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		clog.Debugf("Failed to upgrade (%s)", err)
		return
	}
	conn := chshare.NewWebSocketConn(wsConn)
	// perform SSH handshake on net.Conn
	clog.Debugf("Handshaking...")
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.sshConfig)
	if err != nil {
		s.Debugf("Failed to handshake (%s)", err)
		return
	}
	// pull the users from the session map
	var user *chshare.User
	sid := GetSessionID(sshConn)
	if s.users.Len() > 0 {
		user, _ = s.authenticatedUsers.Get(sid)
		s.authenticatedUsers.Del(sid)
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
		failed(s.Errorf("expecting config request"))
		return
	}
	c, err := chshare.DecodeConfig(r.Payload)
	if err != nil {
		failed(s.Errorf("invalid config"))
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

	//if user is provided, ensure they have
	//access to the desired remotes
	if user != nil {
		for _, remote := range c.Remotes {
			if !user.HasAccess(remote) {
				failed(s.Errorf("access to requested address denied (%s:%s)", remote.LocalHost, remote.LocalPort))
				return
			}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sessionInfo := &ClientSession{
		ID:         sid,
		Version:    c.Version,
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
			failed(s.Errorf("%s", err))
			return
		}
	}

	//success!
	_ = r.Reply(true, nil)

	clog.Debugf("Open %s", sessionInfo.ID)
	s.sessions[sessionInfo.ID] = sessionInfo
	go s.handleSSHRequests(clog, reqs)
	go s.handleSSHChannels(clog, chans)
	_ = sshConn.Wait()
	delete(s.sessions, sessionInfo.ID)
	clog.Debugf("Close %s", sessionInfo.ID)
}

func (s *Server) handleSSHRequests(clientLog *chshare.Logger, reqs <-chan *ssh.Request) {
	for r := range reqs {
		switch r.Type {
		case "ping":
			_ = r.Reply(true, nil)
		default:
			clientLog.Debugf("Unknown request: %s", r.Type)
		}
	}
}

func (s *Server) handleSSHChannels(clientLog *chshare.Logger, chans <-chan ssh.NewChannel) {
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
		connID := s.connStats.New()
		go chshare.HandleTCPStream(clientLog.Fork("conn#%d", connID), &s.connStats, stream, remote)
	}
}
