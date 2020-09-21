//+build !windows

package chserver

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudradar-monitoring/rport/server/api/users"
)

// ReloadAPIUsers reloads API users from file when SIGUSR1 is received.
func (al *APIListener) ReloadAPIUsers() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1)
	for range signals {
		al.Infof("Signal SIGUSR1 received. Start to reload API users from file.")
		newUsers, err := users.GetUsersFromFile(al.authFile)
		if err != nil {
			al.Errorf("Failed to reload API users from the file: %v", err)
			continue
		}

		al.userSrv = users.NewUserRepository(newUsers)
		al.Infof("Finished to reload API users from file. Loaded %d users.", len(newUsers))
	}
}
