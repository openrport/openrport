package clienttunnel

import (
	_ "embed" //to embed html templates
	"net"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/wwt/guac"
)

const (
	queryParToken     = "token"
	queryParSecurity  = "security"
	queryParUsername  = "username"
	queryParPassword  = "password"
	queryParDomain    = "domain"
	queryParWidth     = "width"
	queryParHeight    = "height"
	queryParKeyboard  = "keyboard"
	queryParGuacError = "guac-error"
)

var keysSecurity = []string{"", "any", "nla", "nla-ext", "tls", "vmconnect", "rdp"}
var keysKeyboard = []string{"", "pt-br-qwerty", "en-gb-qwerty", "en-us-qwerty", "fr-fr-azerty", "fr-be-azerty", "fr-ch-qwertz", "de-de-qwertz", "de-ch-qwertz", "hu-hu-qwertz",
	"it-it-qwerty", "ja-jp-qwerty", "no-no-qwerty", "es-es-qwerty", "es-latam-qwerty", "sv-se-qwerty", "tr-tr-qwerty"}
var valuesKeyboard = []string{"", "Brazilian (Portuguese)", "English (UK)", "English (US)", "French", "French (Belgian)", "French (Swiss)", "German", "German (Swiss)", "Hungarian",
	"Italian", "Japanese", "Norwegian", "Spanish", "Spanish (Latin American)", "Swedish", "Turkish-Q"}

//go:embed guac/index.html
var guacIndexHTML string

//go:embed guac/start-tunnel.html
var guacStartTunnelHTML string

//TunnelProxyConnectorRDP connects to a rdp tunnel via guacd (Guacamole server)
type TunnelProxyConnectorRDP struct {
	tunnelProxy         *TunnelProxy
	guacWebsocketServer *guac.WebsocketServer
	guacTokenStore      *GuacTokenStore
}

func NewTunnelConnectorRDP(tp *TunnelProxy) *TunnelProxyConnectorRDP {
	tpc := &TunnelProxyConnectorRDP{tunnelProxy: tp}
	tpc.guacWebsocketServer = guac.NewWebsocketServer(tpc.connectToGuacamole)
	tpc.guacTokenStore = NewGuacTokenStore()

	return tpc
}

//InitRouter called when tunnel proxy is started
func (tc *TunnelProxyConnectorRDP) InitRouter(router *mux.Router) *mux.Router {
	router.Use(tc.tunnelProxy.noCache)
	router.Use(tc.handleFormValues)

	router.Handle("/websocket-tunnel", tc.guacWebsocketServer)

	router.HandleFunc("/", tc.serveIndex)
	router.HandleFunc("/createToken", tc.serveTunnelStarter)

	return router
}

func (tc *TunnelProxyConnectorRDP) serveIndex(w http.ResponseWriter, r *http.Request) {
	if tc.tunnelProxy.Config.GuacdAddress == "" {
		tc.tunnelProxy.sendHTML(w, http.StatusBadRequest, "No guacd configured.")
		return
	}

	selSecurity := r.Form.Get(queryParSecurity)
	selKeyboard := r.Form.Get(queryParKeyboard)
	guacError := r.Form.Get(queryParGuacError)
	templateData := map[string]interface{}{
		queryParUsername:  r.Form.Get(queryParUsername),
		queryParDomain:    r.Form.Get(queryParDomain),
		queryParSecurity:  r.Form.Get(queryParSecurity),
		queryParKeyboard:  r.Form.Get(queryParKeyboard),
		queryParWidth:     r.Form.Get(queryParWidth),
		queryParHeight:    r.Form.Get(queryParHeight),
		"errorMessage":    guacError,
		"isError":         guacError != "",
		"securityOptions": CreateOptions(keysSecurity, keysSecurity, selSecurity),
		"keyboardOptions": CreateOptions(keysKeyboard, valuesKeyboard, selKeyboard),
	}

	tc.tunnelProxy.serveTemplate(w, r, guacIndexHTML, templateData)
}

func (tc *TunnelProxyConnectorRDP) serveTunnelStarter(w http.ResponseWriter, r *http.Request) {
	guacToken := parseGuacToken(r)
	token := uuid.New().String()
	tc.guacTokenStore.Add(token, guacToken)

	templateData := map[string]interface{}{
		"token":          token,
		queryParUsername: guacToken.username,
		queryParDomain:   guacToken.domain,
		queryParSecurity: guacToken.security,
		queryParKeyboard: guacToken.keyboard,
	}

	tc.tunnelProxy.serveTemplate(w, r, guacStartTunnelHTML, templateData)
}

func parseGuacToken(r *http.Request) *GuacToken {
	token := &GuacToken{}
	token.security = r.Form.Get(queryParSecurity)
	token.username = r.Form.Get(queryParUsername)
	token.password = r.Form.Get(queryParPassword)
	token.domain = r.Form.Get(queryParDomain)
	token.width = r.Form.Get(queryParWidth)
	token.height = r.Form.Get(queryParHeight)
	token.keyboard = r.Form.Get(queryParKeyboard)

	return token
}

// connectToGuacamole creates the tunnel to the remote machine (via guacd)
func (tc *TunnelProxyConnectorRDP) connectToGuacamole(r *http.Request) (guac.Tunnel, error) {
	tc.tunnelProxy.Logger.Infof("TunnelProxyConnectorRDP: connect to tunnel: %s", tc.tunnelProxy.TunnelAddr())
	config := guac.NewGuacamoleConfiguration()

	config.Protocol = "rdp"
	config.Parameters["hostname"] = tc.tunnelProxy.TunnelHost
	config.Parameters["port"] = tc.tunnelProxy.TunnelPort
	config.Parameters["ignore-cert"] = "true"

	var err error

	token := r.Form.Get(queryParToken)
	guacToken := tc.guacTokenStore.Get(token)
	if guacToken == nil {
		tc.tunnelProxy.Logger.Errorf("Cannot find guac token %s", token)
		return nil, err
	}
	tc.guacTokenStore.Delete(token)

	config.Parameters[queryParSecurity] = guacToken.security
	config.Parameters[queryParUsername] = guacToken.username
	config.Parameters[queryParPassword] = guacToken.password
	config.Parameters[queryParDomain] = guacToken.domain
	config.Parameters["server-layout"] = guacToken.keyboard

	width := guacToken.width
	if width != "" {
		config.OptimalScreenWidth, err = strconv.Atoi(width)
		if err != nil || config.OptimalScreenWidth == 0 {
			tc.tunnelProxy.Logger.Errorf("invalid screen width %s", width)
			config.OptimalScreenWidth = 1024
		}
	}

	height := guacToken.height
	if height != "" {
		config.OptimalScreenHeight, err = strconv.Atoi(height)
		if err != nil || config.OptimalScreenHeight == 0 {
			tc.tunnelProxy.Logger.Errorf("invalid screen height %s", height)
			config.OptimalScreenHeight = 768
		}
	}

	config.AudioMimetypes = []string{"audio/L16", "rate=44100", "channels=2"}

	tc.tunnelProxy.Logger.Debugf("Connecting to guacd")
	addr, err := net.ResolveTCPAddr("tcp", tc.tunnelProxy.Config.GuacdAddress)
	if err != nil {
		tc.tunnelProxy.Logger.Errorf("error while resolving guacd address:%v", err)
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		tc.tunnelProxy.Logger.Errorf("error while connecting to guacd:%v", err)
		return nil, err
	}

	stream := guac.NewStream(conn, guac.SocketTimeout)

	tc.tunnelProxy.Logger.Debugf("Connected to guacd")
	tc.tunnelProxy.Logger.Debugf("Starting handshake with %#v", config)
	err = stream.Handshake(config)
	if err != nil {
		tc.tunnelProxy.Logger.Errorf("Handshaking with guacd failed with %#v", err)
		return nil, err
	}
	tc.tunnelProxy.Logger.Debugf("Socket configured")
	return guac.NewSimpleTunnel(stream), nil
}

//handleFormValues middleware to handle parsing form values
func (tc *TunnelProxyConnectorRDP) handleFormValues(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			tc.tunnelProxy.Logger.Errorf("Error parsing form values: %v", err)
			tc.tunnelProxy.sendHTML(w, http.StatusInternalServerError, "Cannot parse form values")
			return
		}
		next.ServeHTTP(w, r)
	})
}
