// Generating data for tests is always cumbersome.
// To make it easier this package should be a single source of truth for generating Clients data.
//
// This package provides a builder that can generate Clients with:
// - preset fields,
// - randomly generated fields,
// - fields set on demand.
//
// It can be extended by needs.
package clients

import (
	"testing"
	"time"

	"github.com/openrport/openrport/share/clientconfig"
	"github.com/openrport/openrport/share/logger"

	"golang.org/x/crypto/ssh"

	"github.com/stretchr/testify/require"

	"github.com/openrport/openrport/server/clients/clientdata"
	"github.com/openrport/openrport/server/clients/clienttunnel"
	chshare "github.com/openrport/openrport/share/models"
	"github.com/openrport/openrport/share/random"
)

var clientsNow, _ = time.ParseInLocation(time.RFC3339, "2020-08-19T13:09:23+03:00", nil)

// nowMockF is used to override clients.Now
var nowMockF = func() time.Time {
	return clientsNow
}

type ClientBuilder struct {
	t *testing.T

	id                string
	clientAuthID      string
	disconnectedAt    *time.Time
	allowedUserGroups []string
	conn              ssh.Conn
	logger            *logger.Logger
	cfg               *clientconfig.Config
}

// New returns a builder to generate a client that can be used in tests.
func New(t *testing.T) ClientBuilder {
	id, err := clientdata.NewClientID()
	require.NoError(t, err)
	return ClientBuilder{
		t:            t,
		id:           id,
		clientAuthID: generateRandomClientAuthID(),
	}
}

func (b ClientBuilder) ID(id string) ClientBuilder {
	b.id = id
	return b
}

func (b ClientBuilder) Logger(l *logger.Logger) ClientBuilder {
	b.logger = l
	return b
}

func (b ClientBuilder) ClientAuthID(clientAuthID string) ClientBuilder {
	b.clientAuthID = clientAuthID
	return b
}

func (b ClientBuilder) DisconnectedDuration(disconnectedDuration time.Duration) ClientBuilder {
	// override client Now with static value
	clientdata.Now = nowMockF
	disconnectedAt := clientdata.Now().Add(-disconnectedDuration)
	b.disconnectedAt = &disconnectedAt
	return b
}

func (b ClientBuilder) AllowedUserGroups(allowedUserGroups []string) ClientBuilder {
	b.allowedUserGroups = allowedUserGroups
	return b
}

func (b ClientBuilder) Connection(conn ssh.Conn) ClientBuilder {
	b.conn = conn
	return b
}

func (b ClientBuilder) Config(cfg *clientconfig.Config) ClientBuilder {
	b.cfg = cfg
	return b
}

func (b ClientBuilder) Build() *clientdata.Client {
	cl := &clientdata.Client{
		NumCPUs:                2,
		MemoryTotal:            100000,
		ID:                     b.id,
		Name:                   "Random Rport Client",
		OS:                     "Linux alpine-3-10-tk-01 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
		OSArch:                 "amd64",
		OSFamily:               "alpine",
		OSKernel:               "linux",
		OSFullName:             "Debian 18.0",
		OSVersion:              "18.0",
		OSVirtualizationSystem: "LVM",
		OSVirtualizationRole:   "guest",
		CPUFamily:              "Virtual CPU",
		CPUModel:               "Virtual CPU",
		CPUModelName:           "",
		CPUVendor:              "GenuineIntel",
		Timezone:               "UTC-0",
		Hostname:               "alpine-3-10-tk-01",
		IPv4:                   []string{"192.168.122.111"},
		IPv6:                   []string{"fe80::b84f:aff:fe59:a0b1"},
		Tags:                   []string{"Linux", "Datacenter 1"},
		Labels:                 map[string]string{"country": "Germany", "city": "Cologne", "datacenter": "NetCologne GmbH"},
		Version:                "0.1.12",
		Address:                "88.198.189.161:50078",
		Tunnels: []*clienttunnel.Tunnel{
			{
				ID: "1",
				Remote: chshare.Remote{
					Protocol:   chshare.ProtocolTCP,
					LocalHost:  "0.0.0.0",
					LocalPort:  "2222",
					RemoteHost: "0.0.0.0",
					RemotePort: "22",
				},
			},
			{
				ID: "2",
				Remote: chshare.Remote{
					Protocol:   chshare.ProtocolTCP,
					LocalHost:  "0.0.0.0",
					LocalPort:  "4000",
					RemoteHost: "0.0.0.0",
					RemotePort: "80",
				},
			},
		},
		DisconnectedAt:    b.disconnectedAt,
		ClientAuthID:      b.clientAuthID,
		AllowedUserGroups: b.allowedUserGroups,

		Connection:          b.conn,
		ClientConfiguration: b.cfg,
		Logger:              b.logger,
	}

	return cl
}

func generateRandomClientAuthID() string {
	return "client-auth-" + random.Code(2)
}
