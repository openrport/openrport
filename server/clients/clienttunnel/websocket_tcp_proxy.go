package clienttunnel

import (
	"net"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/cloudradar-monitoring/rport/share/logger"
)

const websocketBufferSize = 1024

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  websocketBufferSize,
	WriteBufferSize: websocketBufferSize,
	CheckOrigin:     func(r *http.Request) bool { return true },
	Subprotocols:    []string{"binary"},
}

// WebsocketTCPProxy holds state information about the connection being proxied.
type WebsocketTCPProxy struct {
	wsConn  *websocket.Conn
	tcpAddr *net.TCPAddr
	tcpConn *net.TCPConn
	logger  *logger.Logger
}

// Initialize WebsocketTCPProxy
func (p *WebsocketTCPProxy) Initialize(wsConn *websocket.Conn, tcpAddr *net.TCPAddr, logger *logger.Logger) *WebsocketTCPProxy {
	p.wsConn = wsConn
	p.tcpAddr = tcpAddr
	p.logger = logger

	return p
}

// Start the bidirectional communication channel between the WebSocket and the TCP connection.
func (p *WebsocketTCPProxy) Start() {
	go p.ReadWebSocket()
	go p.ReadTCP()
}

func (p *WebsocketTCPProxy) Dial() error {
	tcpConn, err := net.DialTCP(p.tcpAddr.Network(), nil, p.tcpAddr)

	if err != nil {
		message := "tcp dialing failed: " + err.Error()
		_ = p.wsConn.WriteMessage(websocket.TextMessage, []byte(message))
		return err
	}

	p.tcpConn = tcpConn

	p.logger.Infof("WebSocket %s connected to TCP %+v:%d", p.wsConn.RemoteAddr(), p.tcpAddr.IP, p.tcpAddr.Port)

	return nil
}

// ReadWebSocket reads from the WebSocket and writes to the TCP connection.
func (p *WebsocketTCPProxy) ReadWebSocket() {
	for {
		_, data, err := p.wsConn.ReadMessage()
		if err != nil {
			p.Teardown()
			break
		}

		_, err = p.tcpConn.Write(data)
		if err != nil {
			p.logger.Errorf(" error writing websocket buffer to tcp connection: %v", err)
			break
		}
	}
}

// ReadTCP reads from the backend TCP connection and writes to the WebSocket.
func (p *WebsocketTCPProxy) ReadTCP() {
	buffer := make([]byte, websocketBufferSize)

	for {
		bytesRead, err := p.tcpConn.Read(buffer)

		if err != nil {
			p.Teardown()
			break
		}

		if err := p.wsConn.WriteMessage(websocket.BinaryMessage, buffer[:bytesRead]); err != nil {
			p.logger.Errorf(" error writing tcp buffer to websocket: %v", err)
			break
		}
	}
}

// Teardown the WebSocket and TCP connection.
func (p *WebsocketTCPProxy) Teardown() {
	p.tcpConn.Close()
	p.wsConn.Close()
}
