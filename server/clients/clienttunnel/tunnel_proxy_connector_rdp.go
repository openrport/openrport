package clienttunnel

import (
	_ "embed" //to embed novnc wrapper templates
	"html/template"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/wwt/guac"
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
	router.Use(noCache)

	router.Handle("/websocket-tunnel", tc.guacWebsocketServer)

	router.HandleFunc("/", tc.serveIndex)

	return router
}

func (tc *TunnelProxyConnectorRDP) serveIndex(w http.ResponseWriter, r *http.Request) {
	tc.serveTemplate(w, r, guacIndexHTML, map[string]interface{}{})
}

func (tc *TunnelProxyConnectorRDP) serveTemplate(w http.ResponseWriter, r *http.Request, templateContent string, templateData map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	tmpl, err := template.New("").Parse(templateContent)
	if err == nil {
		err = tmpl.Execute(w, templateData)
	}
	if err != nil {
		tc.tunnelProxy.Logger.Errorf("Error while serving template for request %s: %v", r.RequestURI, err)
	}
}

// connectToGuacamole creates the tunnel to the remote machine (via guacd)
func (tc *TunnelProxyConnectorRDP) connectToGuacamole(request *http.Request) (guac.Tunnel, error) {
	tc.tunnelProxy.Logger.Infof("TunnelProxyConnectorRDP: connect to tunnel: %s", tc.tunnelProxy.TunnelAddr())
	config := guac.NewGuacamoleConfiguration()

	config.Protocol = "rdp"
	config.Parameters["hostname"] = "127.0.0.1"
	config.Parameters["port"] = tc.tunnelProxy.TunnelPort
	config.Parameters["ignore-cert"] = "true"

	config.AudioMimetypes = []string{"audio/L16", "rate=44100", "channels=2"}

	tc.tunnelProxy.Logger.Debugf("Connecting to guacd")
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:4822")
	if err != nil {
		tc.tunnelProxy.Logger.Errorf("error while resolving guacd address:%v", err)
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		tc.tunnelProxy.Logger.Errorf("error while connecting to guacd:%v", err)
		return nil, err
	}

	stream := guac.NewStream(conn, guac.SocketTimeout)

	tc.tunnelProxy.Logger.Debugf("Connected to guacd")
	if request.URL.Query().Get("uuid") != "" {
		config.ConnectionID = request.URL.Query().Get("uuid")
	}
	tc.tunnelProxy.Logger.Debugf("Starting handshake with %#v", config)
	err = stream.Handshake(config)
	if err != nil {
		tc.tunnelProxy.Logger.Errorf("Handshaking with guacd failed with %#v", err)
		return nil, err
	}
	tc.tunnelProxy.Logger.Debugf("Socket configured")
	return guac.NewSimpleTunnel(stream), nil
}
