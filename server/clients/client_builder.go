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

	"golang.org/x/crypto/ssh"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/random"
)

var clientsNow, _ = time.ParseInLocation(time.RFC3339, "2020-08-19T13:09:23+03:00", nil)

// nowMockF is used to override clients.now
var nowMockF = func() time.Time {
	return clientsNow
}

type ClientBuilder struct {
	t *testing.T

	id           string
	clientAuthID string
	disconnected *time.Time
	conn         ssh.Conn
}

// New returns a builder to generate a client that can be used in tests.
func New(t *testing.T) ClientBuilder {
	return ClientBuilder{
		t:            t,
		id:           generateRandomCID(),
		clientAuthID: generateRandomClientAuthID(),
	}
}

func (b ClientBuilder) ID(id string) ClientBuilder {
	b.id = id
	return b
}

func (b ClientBuilder) ClientAuthID(clientAuthID string) ClientBuilder {
	b.clientAuthID = clientAuthID
	return b
}

func (b ClientBuilder) DisconnectedDuration(disconnectedDuration time.Duration) ClientBuilder {
	// override client Now with static value
	now = nowMockF
	disconnected := now().Add(-disconnectedDuration)
	b.disconnected = &disconnected
	return b
}

func (b ClientBuilder) Connection(conn ssh.Conn) ClientBuilder {
	b.conn = conn
	return b
}

func (b ClientBuilder) Build() *Client {
	return &Client{
		ID:       b.id,
		Name:     "Random Rport Client",
		OS:       "Linux alpine-3-10-tk-01 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
		OSArch:   "amd64",
		OSFamily: "alpine",
		OSKernel: "linux",
		Hostname: "alpine-3-10-tk-01",
		IPv4:     []string{"192.168.122.111"},
		IPv6:     []string{"fe80::b84f:aff:fe59:a0b1"},
		Tags:     []string{"Linux", "Datacenter 1"},
		Version:  "0.1.12",
		Address:  "88.198.189.161:50078",
		Tunnels: []*Tunnel{
			{
				ID: "1",
				Remote: chshare.Remote{
					LocalHost:  "0.0.0.0",
					LocalPort:  "2222",
					RemoteHost: "0.0.0.0",
					RemotePort: "22",
				},
			},
			{
				ID: "2",
				Remote: chshare.Remote{
					LocalHost:  "0.0.0.0",
					LocalPort:  "4000",
					RemoteHost: "0.0.0.0",
					RemotePort: "80",
				},
			},
		},
		Disconnected: b.disconnected,
		ClientAuthID: b.clientAuthID,

		Connection: b.conn,
	}

}

func generateRandomCID() string {
	return "cid-" + random.AlphaNum(12)
}

func generateRandomClientAuthID() string {
	return "client-auth-" + random.Code(2)
}
