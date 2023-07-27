package ipAddresses

import (
	"encoding/json"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/realvnc-labs/rport/share/comm"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/models"
	"github.com/realvnc-labs/rport/share/myip"
)

type IPAddresses struct {
	// mtx protects both conn and status
	mtx      sync.RWMutex
	conn     ssh.Conn
	logger   *logger.Logger
	loopWait time.Duration
	IPAPIURL string
}

var current = &models.IPAddresses{}

func New(logger *logger.Logger, IPAPIURL string, loopWait time.Duration) *IPAddresses {
	return &IPAddresses{
		logger:   logger,
		IPAPIURL: IPAPIURL,
		loopWait: loopWait,
	}
}

func (i *IPAddresses) sendIPAddresses() {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	if i.conn != nil {
		ips, err := myip.GetMyIPs(i.IPAPIURL)
		if err != nil {
			i.logger.Errorf("Failed to determine IP addresses: %s", err)
		}
		if ips.IPv6 == current.IPv6 && ips.IPv4 == current.IPv4 {
			i.logger.Debugf("Clients external IP addresses did not change.")
			return
		}
		i.logger.Debugf("Client external IP addresses changed: '%s','%s'.", ips.IPv4, ips.IPv6)

		data, err := json.Marshal(ips)
		if err != nil {
			i.logger.Errorf("Failed json marshaling external IP address update: %s", err)
		}
		i.logger.Debugf("Sending external IP addresses update.")
		_, _, err = i.conn.SendRequest(comm.RequestTypeIPAddresses, false, data)
		if err != nil {
			i.logger.Errorf("failed updating IP addresses: %s", err)
			return
		}
		current = ips
	}
}

func (i *IPAddresses) refreshLoop() {
	for {
		i.sendIPAddresses()
		time.Sleep(i.loopWait * time.Minute)
	}
}

func (i *IPAddresses) SetConn(c ssh.Conn) {
	if i.IPAPIURL == "" {
		i.logger.Infof("Fetching external IP addresses disabled.")
		return
	}
	i.logger.Infof("Will fetch external IP addresses every %d min from '%s'", i.loopWait, i.IPAPIURL)
	i.mtx.Lock()
	defer i.mtx.Unlock()

	i.conn = c
	go i.refreshLoop()
}
