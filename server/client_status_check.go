package chserver

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

type ClientsStatusCheckTask struct {
	log         *logger.Logger
	cr          *clients.ClientRepository
	th          time.Duration // Threshold after which a client to server ping is considered outdated.
	pingTimeout time.Duration // Don't wait longer than pingTimeout for a response
}

// NewClientsStatusCheckTask pings all active clients and marks them disconnected on ping failure
func NewClientsStatusCheckTask(log *logger.Logger, cr *clients.ClientRepository, th time.Duration, pingTimeout time.Duration) *ClientsStatusCheckTask {
	return &ClientsStatusCheckTask{
		log:         log.Fork("ClientsStatusCheck"),
		cr:          cr,
		th:          th,
		pingTimeout: pingTimeout,
	}
}

func (t *ClientsStatusCheckTask) Run(ctx context.Context) error {
	timerStart := time.Now()
	var dueClients []*clients.Client
	var confirmedClients = 0
	var now = time.Now()
	// Shorten the threshold aka make heartbeat older than it is because the ping response is stored after this check.
	// Clients would get checked only every second time otherwise.
	t.th = t.th - 10*time.Second
	for _, c := range t.cr.GetAllActive() {
		if c.LastHeartbeatAt != nil && now.Sub(*c.LastHeartbeatAt) < t.th {
			// Skip all clients having sent a heartbeat from client to server recently
			confirmedClients++
			continue
		}
		dueClients = append(dueClients, c)
	}
	if len(dueClients) == 0 {
		// Nothing to do
		t.log.Debugf("ended after %s, no clients to ping", time.Since(timerStart))
		return nil
	}
	maxWorkers := 100
	if maxWorkers > len(dueClients) {
		maxWorkers = len(dueClients)
	}
	jobs := make(chan *clients.Client, len(dueClients))
	results := make(chan bool, len(dueClients))
	for w := 1; w <= maxWorkers; w++ {
		go t.PingClients(jobs, results)
	}
	for _, dueClient := range dueClients {
		jobs <- dueClient
	}
	var dead = 0
	var alive = 0
	for a := 1; a <= len(dueClients); a++ {
		if <-results {
			alive++
		} else {
			dead++
		}
	}
	t.log.Debugf("ended after %s, skipped: %d, pinged: %d, alive: %d, dead: %d", time.Since(timerStart), confirmedClients, len(dueClients), alive, dead)
	return nil
}

func (t *ClientsStatusCheckTask) PingClients(jobs <-chan *clients.Client, results chan<- bool) {
	for j := range jobs {
		ok, response, rtt, err := t.PingClientWithTimeout(j)
		//t.log.Debugf("ok=%s, error=%s, response=%s", ok, err, response)
		var now = time.Now()
		//Old clients cannot respond properly to a ping request yet
		if !ok && err == nil && string(response) == "unknown request" {
			t.log.Debugf("ping to %s [%s] succeeded in %s. client < 0.8.2", j.Name, j.ID, rtt)
			j.LastHeartbeatAt = &now
			results <- true
			continue
		}
		// Only an empty response confirms the ping
		if ok && err == nil && len(response) == 0 {
			t.log.Debugf("ping to %s [%s] succeeded in %s. client >= 0.8.2", j.Name, j.ID, rtt)
			j.LastHeartbeatAt = &now
			results <- true
			continue
		}
		// None of the above. Ping must have failed or timed out.
		t.log.Infof("ping to %s [%s] failed: %s", j.Name, j.ID, err)
		j.DisconnectedAt = &now
		j.Close()
		results <- false
	}
}

func (t *ClientsStatusCheckTask) PingClientWithTimeout(client *clients.Client) (bool, []byte, time.Duration, error) {
	var (
		ok         bool
		response   []byte
		err        error
		timerStart = time.Now()
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan bool, 1)
	go func() {
		ok, response, err = client.Connection.SendRequest(comm.RequestTypePing, true, nil)
		select {
		default:
			ch <- true
		case <-ctx.Done():
			return
		}
	}()
	select {
	case <-ch:
		return ok, response, time.Since(timerStart), err
	case <-time.After(t.pingTimeout):
		return false, nil, 0, fmt.Errorf("timeout %s exceeded", t.pingTimeout)
	}
}
