package chclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/openrport/openrport/share/logger"
)

const (
	WatchdogStateReconnecting = "reconnecting"
	WatchdogStateConnected    = "connected"
	WatchdogStateInit         = "initialized"

	// SdNotifyWatchdog tells the service manager to update the watchdog
	// timestamp for the service.
	sdNotifyWatchdog = "WATCHDOG=1"
)

type Watchdog struct {
	stateFile string
	logger    *logger.Logger
	socket    *net.UnixConn
}

type watchdogState struct {
	LastUpdate   time.Time `json:"last_update"`
	LastUpdateTS int64     `json:"last_update_ts"` // Include a unix timestamp go easy processing with scripting languages
	LastState    string    `json:"last_state"`
	LastMessage  string    `json:"last_message"`
}

func NewWatchdog(enabled bool, dataDir string, logger *logger.Logger) (*Watchdog, error) {
	if !enabled {
		logger.Debugf("Watchdog integration disabled")
		return nil, nil
	}
	socketAddr := &net.UnixAddr{
		Name: os.Getenv("NOTIFY_SOCKET"),
		Net:  "unixgram",
	}
	w := &Watchdog{
		stateFile: filepath.Join(dataDir, "state.json"),
		logger:    logger,
	}
	logger.Debugf("Will create a watchdog state file in %s", w.stateFile)
	if socketAddr.Name != "" {
		socket, err := net.DialUnix(socketAddr.Net, nil, socketAddr)
		if err != nil {
			return nil, err
		}
		w.socket = socket
		logger.Debugf("Using NOTIFY_SOCKET %s for systemd watchdog integration", socketAddr.Name)
	} else {
		logger.Debugf("Not using NOTIFY_SOCKET. Either not running in systemd context or systemd watchdog disabled.")
	}
	return w, w.update(WatchdogStateInit, "")
}

func (w *Watchdog) update(state string, msg string) error {
	if err := w.sdNotify(); err != nil {
		w.logger.Errorf("failed to send sd_notify to socket: %s", err)
	}
	s := watchdogState{
		LastUpdate:   time.Now(),
		LastUpdateTS: time.Now().Unix(),
		LastState:    state,
		LastMessage:  msg,
	}
	j, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal watchdog state: %s", err)
	}
	err = ioutil.WriteFile(w.stateFile, j, 0644) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to write watchdog state file %s: %s", w.stateFile, err)
	}
	return nil
}

func (w *Watchdog) Ping(state string, msg string) {
	if w == nil {
		return
	}
	if err := w.update(state, msg); err != nil {
		w.logger.Errorf(err.Error())
	}
}

func (w *Watchdog) sdNotify() error {
	if w.socket == nil {
		return nil
	}
	if _, err := w.socket.Write([]byte(sdNotifyWatchdog)); err != nil {
		return err
	}
	return nil
}

func (w *Watchdog) Close() error {
	if w == nil {
		return nil
	}
	if w.socket != nil {
		return w.socket.Close()
	}
	return nil
}
