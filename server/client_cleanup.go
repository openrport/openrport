package chserver

import (
	"context"
	"runtime"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

type DisconnectedHostsCleanupTask struct {
	log *logger.Logger
	cs  *ClientService
}

// NewDisconnectedHostsCleanupTask pings all active clients and marks them disconnected on ping failure
func NewDisconnectedHostsCleanupTask(log *logger.Logger, cs *ClientService) *DisconnectedHostsCleanupTask {
	return &DisconnectedHostsCleanupTask{
		log: log,
		cs:  cs,
	}
}

func (t *DisconnectedHostsCleanupTask) Run(ctx context.Context) error {
	clients, err := t.cs.GetAll()
	if err != nil {
		t.log.Errorf("Failed to get list of clients.")
		return err
	}
	t.log.Debugf("Starting NewDisconnectedHostsCleanupTask task for %d clients now.", len(clients))
	//jclients, _ := json.Marshal(clients)
	//t.log.Debugf("Client List: %s", jclients)
	var wg sync.WaitGroup
	for _, client := range clients {
		t.log.Debugf("Running ping task for client %s %s", client.ID, client.Name)
		activeClient, err := t.cs.GetActiveByID(client.ID)
		if activeClient == nil {
			t.log.Debugf("Client '%s' [%s] is not active. Skipping.", client.Name, client.ID)
			continue
		}
		if err != nil {
			t.log.Errorf("Failed to get client: %s", err)
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := t.PingClient(activeClient.Connection, activeClient); err != nil {
				t.log.Errorf("Error dispatching ping go routine: %s", err)
			}
		}()
		t.log.Debugf("number of goroutines: %d", runtime.NumGoroutine())
	}
	wg.Wait()
	return nil
}

func (t *DisconnectedHostsCleanupTask) PingClient(conn ssh.Conn, client *clients.Client) error {
	t.log.Debugf("Sending ping to %s", client.ID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer func() {
		cancel()
	}()
	done := make(chan error, 1)
	go func() {
		done <- func() error {
			// Send a ping request.
			ok, response, err := conn.SendRequest(comm.RequestTypePing, true, nil)
			//t.log.Debugf("ok=%s, error=%s, response=%s", ok, err, response)

			if !ok && err == nil && string(response) == "unknown request" {
				//Old clients cannot respond properly to a ping request
				t.log.Debugf("Ping to %s succeeded. client < 0.8.1", client.ID)
				return nil
			}
			if ok && err == nil {
				t.log.Debugf("Ping to %s succeeded. client >= 0.8.1", client.ID)
				return nil
			}
			if err != nil {
				t.log.Errorf("Ping to %s failed: %s", client.ID, err)
				now := time.Now()
				client.DisconnectedAt = &now
				client.Close()
				return nil
			}
			return nil
		}()
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
