//go:build !windows
// +build !windows

package users

import (
	"os"
	"os/signal"
	"syscall"
)

// ReloadAPIUsers reloads API users from file when SIGUSR1 is received.
func (fa *FileAdapter) reload() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1)
	for range signals {
		fa.Infof("Signal SIGUSR1 received. Start to reload API users from file.")

		err := fa.load()
		if err != nil {
			fa.Errorf("Failed to reload API users: %v", err)
			continue
		}

		fa.Infof("Finished to reload API users from file.")
	}
}
