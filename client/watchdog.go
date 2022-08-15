package chclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudradar-monitoring/rport/share/logger"
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
	enabled    bool
	stateFile  string
	logger     *logger.Logger
	socketAddr *net.UnixAddr
}

type watchdogState struct {
	LastUpdate   time.Time `json:"last_update"`
	LastUpdateTS int64     `json:"last_update_ts"` // Include a unix timestamp go easy processing with scripting languages
	LastState    string    `json:"last_state"`
	LastMessage  string    `json:"last_message"`
}

func NewWatchdog(enabled bool, dataDir string, logger *logger.Logger) (Watchdog, error) {
	if !enabled {
		logger.Debugf("Watchdog integration disabled")
		return Watchdog{
			enabled: enabled,
		}, nil
	}
	socketAddr := &net.UnixAddr{
		Name: os.Getenv("NOTIFY_SOCKET"),
		Net:  "unixgram",
	}
	w := Watchdog{
		enabled:    true,
		stateFile:  filepath.Join(dataDir, "state.json"),
		logger:     logger,
		socketAddr: socketAddr,
	}
	logger.Debugf("Created watchdog state file in %s", w.stateFile)
	if socketAddr.Name != "" {
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
	if !w.enabled {
		return
	}
	if err := w.update(state, msg); err != nil {
		w.logger.Errorf(err.Error())
	}
}

func (w *Watchdog) sdNotify() error {
	if w.socketAddr.Name == "" {
		return nil
	}

	conn, err := net.DialUnix(w.socketAddr.Net, nil, w.socketAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err = conn.Write([]byte(sdNotifyWatchdog)); err != nil {
		return err
	}
	return nil
}
