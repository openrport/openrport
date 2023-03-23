package ws

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/share/logger"
)

type Conn interface {
	NextReader() (messageType int, r io.Reader, err error)
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	WriteJSON(v interface{}) error
	Close() error
}

type ConcurrentWebSocket struct {
	conn              Conn
	mu                sync.Mutex
	log               *logger.Logger
	writesBeforeClose int
}

func NewConcurrentWebSocket(conn Conn, log *logger.Logger) *ConcurrentWebSocket {
	return &ConcurrentWebSocket{
		conn:              conn,
		log:               log,
		writesBeforeClose: 1,
	}
}

func (ws *ConcurrentWebSocket) ReadJSON(inboundMsg interface{}) error {
	_, r, err := ws.conn.NextReader()
	if err != nil {
		return err
	}
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec.Decode(inboundMsg)
}

func (ws *ConcurrentWebSocket) ReadMessage() (messageType int, p []byte, err error) {
	return ws.conn.ReadMessage()
}

func (ws *ConcurrentWebSocket) WriteError(title string, err error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	_ = ws.WriteJSON(api.NewErrAPIPayloadFromMessage("", title, errMsg))
}

// WriteJSON write json message to websocket, counting towards writes before close
func (ws *ConcurrentWebSocket) WriteJSON(jsonOutboundMsg interface{}) error {
	defer ws.dec()
	return ws.WriteNonFinalJSON(jsonOutboundMsg)
}

// WriteNonFinalJSON write json message to websocket, not counting towards writes before close
func (ws *ConcurrentWebSocket) WriteNonFinalJSON(jsonOutboundMsg interface{}) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	err := ws.conn.WriteJSON(jsonOutboundMsg)
	if err != nil {
		ws.log.Errorf("Error WS json write: %v", err)
	}
	return err
}

func (ws *ConcurrentWebSocket) WriteMessage(messageType int, data []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	defer ws.dec()
	return ws.conn.WriteMessage(messageType, data)
}

func (ws *ConcurrentWebSocket) SetWritesBeforeClose(n int) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.writesBeforeClose = n
}

func (ws *ConcurrentWebSocket) dec() {
	ws.writesBeforeClose--
	if ws.writesBeforeClose == 0 {
		err := ws.conn.Close()
		if err != nil {
			ws.log.Errorf("Close ws on dec(): %v", err)
		} else {
			ws.log.Debugf("Close ws on dec()")
		}
	}
}

func (ws *ConcurrentWebSocket) Close() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	err := ws.conn.Close()
	if err != nil {
		ws.log.Errorf("Error on Close ws: %v", err)
	} else {
		ws.log.Debugf("Close ws")
	}
	return err
}

func NewWebSocketCache() WebSocketCache {
	return WebSocketCache{
		m: map[string]*ConcurrentWebSocket{},
	}
}

type WebSocketCache struct {
	m  map[string]*ConcurrentWebSocket
	mu sync.RWMutex
}

func (c *WebSocketCache) Get(key string) *ConcurrentWebSocket {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.m[key]
}

func (c *WebSocketCache) Set(key string, ws *ConcurrentWebSocket) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = ws
}

func (c *WebSocketCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, key)
}

func (c *WebSocketCache) CloseConnections() error {
	for _, conn := range c.m {
		conn.Close()
	}
	return nil
}
