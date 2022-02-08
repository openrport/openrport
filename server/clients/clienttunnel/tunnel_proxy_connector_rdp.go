package clienttunnel

import (
	_ "embed" //to embed novnc wrapper templates
	"net"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/wwt/guac"
)

const (
	queryParUsername = "username"
	queryParWidth    = "width"
	queryParHeight   = "height"
)

//go:embed guac/index.html
var guacIndexHTML string

//TunnelProxyConnectorRDP connects to a rdp tunnel via guacd (Guacamole server)
type TunnelProxyConnectorRDP struct {
	tunnelProxy         *TunnelProxy
	guacWebsocketServer *guac.WebsocketServer
}

func NewTunnelConnectorRDP(tp *TunnelProxy) *TunnelProxyConnectorRDP {
	tpc := &TunnelProxyConnectorRDP{tunnelProxy: tp}
	tpc.guacWebsocketServer = guac.NewWebsocketServer(tpc.connectToGuacamole)

	return tpc
}

//InitRouter called when tunnel proxy is started
func (tc *TunnelProxyConnectorRDP) InitRouter(router *mux.Router) *mux.Router {
	router.Use(tc.tunnelProxy.noCache)

	router.Handle("/websocket-tunnel", tc.guacWebsocketServer)

	router.HandleFunc("/", tc.serveIndex)

	return router
}

func (tc *TunnelProxyConnectorRDP) serveIndex(w http.ResponseWriter, r *http.Request) {
	if tc.tunnelProxy.Config.GuacdAddress == "" {
		tc.tunnelProxy.sendHTML(w, http.StatusBadRequest, "No guacd configured.")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	query := r.URL.Query()
	templateData := map[string]interface{}{
		queryParUsername: query.Get(queryParUsername),
		queryParWidth:    query.Get(queryParWidth),
		queryParHeight:   query.Get(queryParHeight),
	}

	tc.tunnelProxy.serveTemplate(w, r, guacIndexHTML, templateData)
}

// connectToGuacamole creates the tunnel to the remote machine (via guacd)
func (tc *TunnelProxyConnectorRDP) connectToGuacamole(request *http.Request) (guac.Tunnel, error) {
	tc.tunnelProxy.Logger.Infof("TunnelProxyConnectorRDP: connect to tunnel: %s", tc.tunnelProxy.TunnelAddr())
	config := guac.NewGuacamoleConfiguration()

	config.Protocol = "rdp"
	config.Parameters["hostname"] = tc.tunnelProxy.TunnelHost
	config.Parameters["port"] = tc.tunnelProxy.TunnelPort
	config.Parameters["ignore-cert"] = "true"

	var err error
	query := request.URL.Query()
	username := query.Get(queryParUsername)
	if username != "" {
		config.Parameters["username"] = username
	}

	width := query.Get(queryParWidth)
	if width != "" {
		config.OptimalScreenWidth, err = strconv.Atoi(width)
		if err != nil || config.OptimalScreenWidth == 0 {
			tc.tunnelProxy.Logger.Errorf("invalid screen width %s", width)
			config.OptimalScreenWidth = 1024
		}
	}

	height := query.Get(queryParHeight)
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
