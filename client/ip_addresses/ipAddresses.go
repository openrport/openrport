package ipAddresses

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/openrport/openrport/share/comm"
	"github.com/openrport/openrport/share/logger"
	"github.com/openrport/openrport/share/models"
	"github.com/openrport/openrport/share/myip"
)

type Fetcher struct {
	// mtx protects conn
	mtx         sync.RWMutex
	conn        ssh.Conn
	logger      *logger.Logger
	loopWait    time.Duration
	IPAPIURL    string
	refreshChan chan struct{}
}

var current = &models.IPAddresses{}

func NewFetcher(logger *logger.Logger, IPAPIURL string, loopWait time.Duration) *Fetcher {
	return &Fetcher{
		logger:   logger,
		IPAPIURL: IPAPIURL,
		loopWait: loopWait,
	}
}

func (i *Fetcher) sendIPAddresses(ctx context.Context) {
	if i.conn == nil {
		return
	}

	ips, err := myip.GetMyIPs(ctx, i.IPAPIURL)
	if err != nil {
		i.logger.Errorf("Failed to determine IP addresses: %s", err)
		return
	}
	if ips.IPv6 == current.IPv6 && ips.IPv4 == current.IPv4 {
		i.logger.Debugf("Clients external IP addresses did not change.")
		return
	}
	i.logger.Debugf("Client external IP addresses changed: '%s','%s'.", ips.IPv4, ips.IPv6)

	data, err := json.Marshal(ips)
	if err != nil {
		i.logger.Errorf("Failed json marshaling external IP address update: %s", err)
		return
	}

	i.logger.Debugf("Sending external IP addresses update.")
	i.mtx.RLock()
	defer i.mtx.RUnlock()
	_, _, err = i.conn.SendRequest(comm.RequestTypeIPAddresses, false, data)
	if err != nil {
		i.logger.Errorf("failed updating IP addresses: %s", err)
		return
	}
	current = ips
}

func (i *Fetcher) refreshLoop(ctx context.Context) {
	for {
		i.sendIPAddresses(ctx)

		select {
		case <-ctx.Done():
			i.logger.Debugf("ip addresses refreshLoop finished")
			return
		case <-time.After(i.loopWait * time.Minute):
		case <-i.refreshChan:
		}
	}
}

func (i *Fetcher) SetConn(c ssh.Conn) {
	if i.IPAPIURL == "" {
		i.logger.Infof("Fetching external IP addresses disabled.")
		return
	}
	i.logger.Infof("Will fetch external IP addresses every %d min from '%s'", i.loopWait, i.IPAPIURL)
	i.mtx.Lock()
	defer i.mtx.Unlock()

	i.conn = c

}

func (i *Fetcher) Start(ctx context.Context) {
	if i.loopWait <= 0 {
		return
	}

	go i.refreshLoop(ctx)
}

func (i *Fetcher) Stop() {
	i.conn = nil
}
