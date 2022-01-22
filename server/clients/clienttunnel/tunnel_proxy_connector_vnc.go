package clienttunnel

import (
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/cloudradar-monitoring/rport/server/api/middleware"
	"github.com/cloudradar-monitoring/rport/server/clients/clienttunnel/novnc"
)

//TunnelProxyConnectorVNC is a kind of 'websockify' vnc to tcp proxy to be used by a novnc instance to connect to a vnc tunnel
type TunnelProxyConnectorVNC struct {
	tunnelProxy *TunnelProxy
}

func NewTunnelConnectorVNC(tp *TunnelProxy) *TunnelProxyConnectorVNC {
	return &TunnelProxyConnectorVNC{tunnelProxy: tp}
}

//InitRouter called when tunnel proxy is started
func (tc *TunnelProxyConnectorVNC) InitRouter(router *mux.Router) *mux.Router {
	router.Use(noCache)

	router.HandleFunc("/vnc", tc.serveVNC)

	router.HandleFunc("/", tc.serveIndex)
	router.HandleFunc("/error404", tc.serveError404)

	//handle novnc javascript app from local filesystem
	tc.tunnelProxy.Logger.Infof("serving novnc javascript app from: %s", tc.tunnelProxy.Config.NovncRoot)
	router.PathPrefix("/").Handler(middleware.Handle404(http.FileServer(http.Dir(tc.tunnelProxy.Config.NovncRoot)), http.HandlerFunc(tc.serveError404)))

	return router
}

func (tc *TunnelProxyConnectorVNC) serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	novncParamsMap := map[string]string{
		"resize": "scale",
	}

	err := novnc.IndexTMPL.Execute(w, map[string]interface{}{
		"host":            tc.tunnelProxy.Host,
		"port":            tc.tunnelProxy.Port,
		"addr":            tc.tunnelProxy.Addr(),
		"basicUI":         false,
		"noURLPassword":   true,
		"defaultViewOnly": false,
		"params":          novncParamsMap,
	})
	if err != nil {
		tc.tunnelProxy.Logger.Errorf("Error while executing novnc index template: %v", err)
	}
}

func (tc *TunnelProxyConnectorVNC) serveError404(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	err := novnc.Error404TMPL.Execute(w, map[string]interface{}{})
	if err != nil {
		tc.tunnelProxy.Logger.Errorf("Error while executing novnc error404 template: %v", err)
	}
}

func (tc *TunnelProxyConnectorVNC) serveVNC(w http.ResponseWriter, r *http.Request) {
	tc.tunnelProxy.Logger.Infof("TunnelProxyConnectorVNC to tunnel: %s", tc.tunnelProxy.TunnelAddr())

	wsConn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		tc.tunnelProxy.Logger.Errorf("failed to upgrade websocket request: %v", err)
	}

	if err == nil {
		tcpAddr, err := net.ResolveTCPAddr("tcp", tc.tunnelProxy.TunnelAddr())
		if err != nil {
			tc.tunnelProxy.Logger.Errorf("failed to resolve tcp destination: %v", err)
		}

		if err == nil {
			p := new(WebsocketTCPProxy)
			p.Initialize(wsConn, tcpAddr, tc.tunnelProxy.Logger)

			if err = p.Dial(); err != nil {
				tc.tunnelProxy.Logger.Errorf("failed to dial tcp addr: %v", err)
			}

			if err == nil {
				go p.Start()
				return
			}
		}
	}
	_ = wsConn.WriteMessage(websocket.CloseMessage, []byte("could not start websocket tcp proxy"))
}

//noCache middleware to disable caching
func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}
