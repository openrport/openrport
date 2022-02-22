package clienttunnel

import (
	"sync"
)

// GuacToken ... used to transport guacd config parameters from request to request
/*
RPort RDP proxy uses Apache Guacamole to connect to remote RDP server.
The RDP connection process is started in browser by showing a html form, where parameters for controlling guacd are requested.
These parameters are sent to the RPort proxy via a POST-request and stored there in GuacTokenStore for further handling during the guacd handshaking.
This "extra" POST-Request is necessary because the javascript-library "guacamole-common-js", which initiates the websocket-connection to guacd, is sending a GET-Request to connect.
Sending the connection parameters with this GET-Request(which would be possible) would show sensitive data like password as part of the querystring.
*/
type GuacToken struct {
	username string
	password string
	domain   string
	security string
	width    string
	height   string
	keyboard string
}

type GuacTokenStore struct {
	sync.RWMutex
	GuacTokens map[string]*GuacToken
}

func NewGuacTokenStore() *GuacTokenStore {
	return &GuacTokenStore{
		GuacTokens: map[string]*GuacToken{},
	}
}

func (s *GuacTokenStore) Add(uuid string, token *GuacToken) {
	s.Lock()
	defer s.Unlock()
	s.GuacTokens[uuid] = token
}

func (s *GuacTokenStore) Get(uuid string) *GuacToken {
	s.RLock()
	defer s.RUnlock()
	return s.GuacTokens[uuid]
}

func (s *GuacTokenStore) Delete(uuid string) {
	s.Lock()
	defer s.Unlock()
	delete(s.GuacTokens, uuid)
}
