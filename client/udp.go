package chclient

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/realvnc-labs/rport/share/comm"
	"github.com/realvnc-labs/rport/share/logger"
)

const activeConnectionTimeout = 15 * time.Second

type udpHandler struct {
	*logger.Logger
	addr    string
	channel *comm.UDPChannel

	mtx        sync.Mutex
	conns      map[string]net.Conn
	lastActive map[string]time.Time
}

func newUDPHandler(logger *logger.Logger, addr string) *udpHandler {
	return &udpHandler{
		Logger:     logger,
		addr:       addr,
		conns:      make(map[string]net.Conn),
		lastActive: make(map[string]time.Time),
	}
}

func (h *udpHandler) Handle(stream io.ReadWriteCloser) error {
	defer stream.Close()

	h.channel = comm.NewUDPChannel(stream)
	for {
		id, data, err := h.channel.Decode()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		conn, err := h.getConn(id)
		if err != nil {
			return err
		}

		h.setActive(id)
		_, err = conn.Write(data)
		if err != nil {
			return err
		}
	}
}

func (h *udpHandler) getConn(id *net.UDPAddr) (net.Conn, error) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	idStr := id.String()
	conn, ok := h.conns[idStr]
	if ok {
		return conn, nil
	}

	conn, err := net.Dial("udp", h.addr)
	if err != nil {
		return nil, err
	}
	go func() {
		err := h.receive(id, conn)
		if err != nil {
			h.Errorf("Error in receive: %v", err)
		}
	}()

	h.conns[idStr] = conn

	return conn, nil
}

func (h *udpHandler) close(id string) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	conn, ok := h.conns[id]
	if !ok {
		return
	}

	h.Debugf("Closing connection for client: %v", id)
	conn.Close()
	delete(h.conns, id)
}

func (h *udpHandler) receive(id *net.UDPAddr, conn net.Conn) error {
	defer h.close(id.String())

	const maxMTU = 9012
	buff := make([]byte, maxMTU)
	for h.isActive(id) {
		err := conn.SetReadDeadline(time.Now().Add(time.Second))
		if err != nil {
			return err
		}

		n, err := conn.Read(buff)
		if e, ok := err.(net.Error); ok && (e.Timeout() || e.Temporary()) {
			continue
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		h.setActive(id)
		err = h.channel.Encode(id, buff[:n])
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *udpHandler) isActive(id *net.UDPAddr) bool {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	la, ok := h.lastActive[id.String()]
	if !ok {
		return true
	}
	return time.Since(la) < activeConnectionTimeout
}

func (h *udpHandler) setActive(id *net.UDPAddr) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	h.lastActive[id.String()] = time.Now()
}
