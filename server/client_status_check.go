package chserver

import (
	"context"
	"time"

	"github.com/realvnc-labs/rport/server/clients"
	"github.com/realvnc-labs/rport/share/comm"
	"github.com/realvnc-labs/rport/share/logger"
)

const DefaultMaxWorkers = 100

type ClientsStatusCheckTask struct {
	log         *logger.Logger
	clientsRepo *clients.ClientRepository
	threshold   time.Duration // Threshold after which a client to server ping is considered outdated.
	pingTimeout time.Duration // Don't wait longer than pingTimeout for a response
}

// NewClientsStatusCheckTask pings all active clients and marks them disconnected on ping failure
func NewClientsStatusCheckTask(log *logger.Logger, cr *clients.ClientRepository, th time.Duration, pingTimeout time.Duration) *ClientsStatusCheckTask {
	return &ClientsStatusCheckTask{
		log:         log.Fork("clients-status-check"),
		clientsRepo: cr,
		threshold:   th,
		pingTimeout: pingTimeout,
	}
}

func (t *ClientsStatusCheckTask) Run(ctx context.Context) error {
	t.log.Debugf("running")
	timerStart := time.Now()
	var confirmedClients = 0

	dueClients, totalClientsCount := t.getDueClients()
	if len(dueClients) == 0 {
		// Nothing to do
		t.log.Debugf("ended after %s, no clients to ping", time.Since(timerStart))
		return nil
	}

	// make sure no more workers than clients and limit to max workers
	maxWorkers := DefaultMaxWorkers
	if maxWorkers > len(dueClients) {
		maxWorkers = len(dueClients)
	}

	// make a channel that will receive all the clients to ping
	clientsToPing := make(chan *clients.Client, len(dueClients))
	// make another channel for ping results
	results := make(chan bool, len(dueClients))

	// create workers to ping clients
	for w := 1; w <= maxWorkers; w++ {
		go t.PingClients(ctx, w, clientsToPing, results)
	}

	// send the clients to ping to the workers
	for _, dueClient := range dueClients {
		clientsToPing <- dueClient
	}

	// we're done queuing clients for processing, so close the channel
	close(clientsToPing)

	// gather the results of pinged clients
	var dead = 0
	var alive = 0
	// TODO: (rs): note this is fragile. any mismatch between actual and expected results will cause
	// the task to block and essential hang. also there's no ctx checking.
	for a := 0; a < len(dueClients); a++ {
		if <-results {
			alive++
		} else {
			dead++
		}
	}

	t.log.Debugf("ended after %s, skipped: %d, pinged: %d, alive: %d, dead: %d, total: %d", time.Since(timerStart), confirmedClients, len(dueClients), alive, dead, totalClientsCount)
	return nil
}

func (t *ClientsStatusCheckTask) getDueClients() (dueClients []*clients.Client, totalCount int) {
	var confirmedClients = 0
	var now = time.Now()
	activeClients, _ := t.clientsRepo.GetAllActiveClients()
	for _, c := range activeClients {
		// Shorten the threshold aka make heartbeat older than it is because the ping response is stored after this check.
		// Clients would get checked only every second time otherwise.
		if c.HasLastHeartbeatAt() {
			lastHeartbeatAt := c.GetLastHeartbeatAtValue()
			if now.Sub(lastHeartbeatAt) < t.threshold-(10*time.Second) {
				// Skip all clients having sent a heartbeat from client to server recently
				// t.log.Debugf("skipping client: %s, %s, %s", c.GetID(), lastHeartbeatAt, now.Sub(lastHeartbeatAt) < t.threshold-(10*time.Second))
				confirmedClients++
				continue
			}
		}
		dueClients = append(dueClients, c)
	}
	return dueClients, len(activeClients)
}

func (t *ClientsStatusCheckTask) PingClients(ctx context.Context, workerNum int, clientsToPing <-chan *clients.Client, results chan<- bool) {
	// while there are clients to ping
	for cl := range clientsToPing {
		clientName := cl.GetName()
		clientID := cl.GetID()
		ok, response, rtt, err := comm.PingConnectionWithTimeout(ctx, cl.GetConnection(), t.pingTimeout, cl.Log())
		//t.log.Debugf("ok=%s, error=%s, response=%s", ok, err, response)

		// Old clients cannot respond properly to a ping request yet
		if !ok && err == nil && t.isLegacyClientResponse(response) {
			t.log.Debugf("ping to %s [%s] succeeded in %s. client < 0.8.2", clientName, clientID, rtt)
			cl.SetHeartbeatNow()
			results <- true
			continue
		}

		// client versions from 0.9.2 to 0.9.6 can return "null" as a ping response. this is due to a bug
		// in the client ping handling that cause 2 replies to be sent by the client. this breaks stuff.
		// for the server, assume the null reply is a successful ping. unfortunately, the extra reply
		// confuses the next send by the client which means it won't get a reply from the server and
		// will ultimately disconnect and reconnect. the work around is to make sure that the client
		// pings the server faster than the server pings the client. as the server has a recent heartbeat
		// already (from the client) it won't ping the client again, meaning that the client won't get a
		// chance to double reply to the server and cause ssh protocol confusion.
		if ok && err == nil && string(response) == "null" {
			t.log.Debugf("ping to %s [%s] succeeded in %s. client >= 0.8.2 *", clientName, clientID, rtt)
			cl.SetHeartbeatNow()
			results <- true
			continue
		}

		// Only an empty response confirms the ping
		if ok && err == nil && len(response) == 0 {
			t.log.Debugf("ping to %s [%s] succeeded in %s. client >= 0.8.2", clientName, clientID, rtt)
			cl.SetHeartbeatNow()
			results <- true
			continue
		}

		// None of the above. Ping must have failed or timed out.
		t.log.Infof("ping to %s [%s] failed: %s", clientName, clientID, err)

		cl.SetDisconnectedNow()

		cl.Close()
		results <- false
	}
}

func (t *ClientsStatusCheckTask) isLegacyClientResponse(response []byte) (isLegacy bool) {
	return string(response) == "unknown request"
}
