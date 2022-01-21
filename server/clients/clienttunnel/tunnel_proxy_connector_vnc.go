package clienttunnel

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"

	"github.com/cloudradar-monitoring/rport/server/api/middleware"
	"github.com/cloudradar-monitoring/rport/server/clients/clienttunnel/novnc"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

const rfbMagic = "RFB"

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
	tc.websockify(tc.tunnelProxy.TunnelAddr(), []byte(rfbMagic), tc.tunnelProxy.Config.CertFile, tc.tunnelProxy.Config.KeyFile).ServeHTTP(w, r)
}

//noCache middleware to disable caching
func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

// websockify returns a http.Handler which proxies websocket requests to a tcp
// address and checks magic bytes.
func (tc *TunnelProxyConnectorVNC) websockify(to string, magic []byte, certFile string, keyFile string) http.Handler {
	tlsConfig := new(tls.Config)
	tlsConfig.Certificates = make([]tls.Certificate, 1)
	tlsConfig.Certificates[0], _ = tls.LoadX509KeyPair(certFile, keyFile)

	return websocket.Server{
		Config: websocket.Config{
			TlsConfig: tlsConfig,
		},
		Handshake: wsProxyHandshake,
		Handler:   wsProxyHandler(to, magic, tc.tunnelProxy.Logger),
	}
}

// wsProxyHandshake is a handshake handler for a websocket.Server.
func wsProxyHandshake(config *websocket.Config, r *http.Request) error {
	if r.Header.Get("Sec-WebSocket-Protocol") != "" {
		config.Protocol = []string{"binary"}
	}
	r.Header.Set("Access-Control-Allow-Origin", "*")
	r.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
	return nil
}

// wsProxyHandler is a websocket.Handler which proxies to a tcp address with a
// magic byte check.
func wsProxyHandler(to string, magic []byte, logger *logger.Logger) websocket.Handler {
	return func(ws *websocket.Conn) {
		conn, err := net.Dial("tcp", to)
		if err != nil {
			ws.Close()
			return
		}

		ws.PayloadType = websocket.BinaryFrame

		m := newMagicCheck(conn, magic)

		done := make(chan error)
		go copyCh(conn, ws, done)
		go copyCh(ws, m, done)

		err = <-done
		if m.Failed() {
			logger.Infof("attempt to connect to non-VNC port (%s, %#v)", to, string(m.Magic()))
		} else if err != nil {
			logger.Infof("%v", err)
		}

		conn.Close()
		ws.Close()
		<-done
	}
}

// copyCh is like io.Copy, but it writes to a channel when finished.
func copyCh(dst io.Writer, src io.Reader, done chan error) {
	_, err := io.Copy(dst, src)
	done <- err
}

// magicCheck implements an efficient wrapper around an io.Reader which checks
// for magic bytes at the beginning, and will return a sticky io.EOF and stop
// reading from the original reader as soon as a mismatch starts.
type magicCheck struct {
	reader    io.Reader
	expected  []byte
	length    int
	remaining int
	actual    []byte
	failed    bool
}

func newMagicCheck(r io.Reader, magic []byte) *magicCheck {
	return &magicCheck{reader: r, expected: magic, length: len(magic), remaining: len(magic), actual: make([]byte, len(magic)), failed: false}
}

// Failed returns true if the magic check has failed (note that it returns false
// if the source io.Reader reached io.EOF before the check was complete).
func (m *magicCheck) Failed() bool {
	return m.failed
}

// Magic returns the magic which was read so far.
func (m *magicCheck) Magic() []byte {
	return m.actual
}

func (m *magicCheck) Read(buf []byte) (n int, err error) {
	if m.failed {
		return 0, io.EOF
	}
	n, err = m.reader.Read(buf)
	if err == nil && n > 0 && m.remaining > 0 {
		m.remaining -= copy(m.actual[m.length-m.remaining:], buf[:n])
		for i := 0; i < m.length-m.remaining; i++ {
			if m.actual[i] != m.expected[i] {
				m.failed = true
				return 0, io.EOF
			}
		}
	}
	return n, err
}
