package clients

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"

	"github.com/cloudradar-monitoring/rport/server/clients/tunnel/novnc"
	"github.com/cloudradar-monitoring/rport/share/logger"
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

	//router.NotFoundHandler = fs("noVNC-master", novnc.NoVNC)
	novncRoot := "/home/moo/devdata/rport/gek-server-1/novnc-root/"
	router.NotFoundHandler = fs("/", http.Dir(novncRoot))

	return router
}

func (tc *TunnelProxyConnectorVNC) DisableHTTP2() bool {
	return true
}

func (tc *TunnelProxyConnectorVNC) serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	novncParamsMap := map[string]string{
		"resize": "scale",
	}

	_ = novnc.IndexTMPL.Execute(w, map[string]interface{}{
		"arbitraryHosts":  false,
		"arbitraryPorts":  false,
		"host":            tc.tunnelProxy.Host,
		"port":            tc.tunnelProxy.Port,
		"addr":            tc.tunnelProxy.Addr(),
		"basicUI":         false,
		"noURLPassword":   true,
		"defaultViewOnly": false,
		"params":          novncParamsMap,
	})
}

func (tc *TunnelProxyConnectorVNC) serveVNC(w http.ResponseWriter, r *http.Request) {
	//w.Header().Set("X-Target-Addr", tc.tunnelProxy.TunnelAddr())

	tc.tunnelProxy.Logger.Infof("TunnelProxyConnectorVNC to tunnel: %s", tc.tunnelProxy.TunnelAddr())
	log.Println("websockify with ", r.RequestURI)
	tc.websockify(tc.tunnelProxy.TunnelAddr(), []byte("RFB"), tc.tunnelProxy.Config.CertFile, tc.tunnelProxy.Config.KeyFile).ServeHTTP(w, r)
}

//noCache middleware to disable caching
func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

// fs returns a http.Handler which serves a directory from a http.FileSystem.
func fs(dir string, fs http.FileSystem) http.Handler {
	return addPrefix("/"+strings.Trim(dir, "/"), http.FileServer(fs))
}

// addPrefix is similar to http.StripPrefix, except it adds a prefix.
func addPrefix(prefix string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		r2.URL.Path = prefix + r.URL.Path
		h.ServeHTTP(w, r2)
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
	rdr io.Reader
	exp []byte
	len int
	rem int
	act []byte
	fld bool
}

func newMagicCheck(r io.Reader, magic []byte) *magicCheck {
	return &magicCheck{r, magic, len(magic), len(magic), make([]byte, len(magic)), false}
}

// Failed returns true if the magic check has failed (note that it returns false
// if the source io.Reader reached io.EOF before the check was complete).
func (m *magicCheck) Failed() bool {
	return m.fld
}

// Magic returns the magic which was read so far.
func (m *magicCheck) Magic() []byte {
	return m.act
}

func (m *magicCheck) Read(buf []byte) (n int, err error) {
	if m.fld {
		return 0, io.EOF
	}
	n, err = m.rdr.Read(buf)
	if err == nil && n > 0 && m.rem > 0 {
		m.rem -= copy(m.act[m.len-m.rem:], buf[:n])
		for i := 0; i < m.len-m.rem; i++ {
			if m.act[i] != m.exp[i] {
				m.fld = true
				return 0, io.EOF
			}
		}
	}
	return n, err
}
