package clients

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/db/migration/clients"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
)

var (
	hour    = time.Hour
	nowMock = nowMockF()
	// c2DisconnectedTime is "Random Rport Client 2" disconnected time
	c2DisconnectedTime, _ = time.Parse(time.RFC3339, "2020-08-19T13:04:23+03:00")
	c3DisconnectedTime    = c2DisconnectedTime.Add(-time.Hour)
	c4DisconnectedTime    = c2DisconnectedTime.Add(-2 * time.Hour)
	testLog               = logger.NewLogger("server", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
)

var c1 = &Client{
	ID:                     "aa1210c7-1899-491e-8e71-564cacaf1df8",
	Name:                   "Random Rport Client 1",
	OS:                     "Linux alpine-3-10-tk-01 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
	OSArch:                 "amd64",
	OSFamily:               "alpine",
	OSKernel:               "linux",
	OSFullName:             "Alpine",
	OSVersion:              "3.14.0",
	OSVirtualizationRole:   "guest",
	OSVirtualizationSystem: "KVM",
	CPUFamily:              "6",
	CPUModel:               "79",
	CPUModelName:           "Common KVM processor",
	CPUVendor:              "GenuineIntel",
	NumCPUs:                2,
	MemoryTotal:            1000000,
	Timezone:               "CEST (UTC+02:00)",
	Hostname:               "alpine-3-10-tk-01",
	IPv4:                   []string{"192.168.122.111"},
	IPv6:                   []string{"fe80::b84f:aff:fe59:a0b1"},
	Tags:                   []string{"Linux", "Datacenter 1"},
	Version:                "0.1.12",
	Address:                "88.198.189.161:50078",
	Tunnels: []*Tunnel{
		{
			ID: "1",
			Remote: models.Remote{
				LocalHost:  "0.0.0.0",
				LocalPort:  "2222",
				RemoteHost: "0.0.0.0",
				RemotePort: "22",
			},
		},
		{
			ID: "2",
			Remote: models.Remote{
				LocalHost:  "0.0.0.0",
				LocalPort:  "4000",
				RemoteHost: "0.0.0.0",
				RemotePort: "80",
			},
		},
	},
	ClientAuthID:   "client-1",
	DisconnectedAt: nil,
}

var c2 = &Client{
	ID:                     "2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
	Name:                   "Random Rport Client 2",
	OS:                     "Linux alpine-3-10-tk-02 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
	OSArch:                 "amd64",
	OSFamily:               "alpine",
	OSKernel:               "linux",
	OSFullName:             "Alpine Linux",
	OSVersion:              "2.0.0",
	OSVirtualizationRole:   "",
	OSVirtualizationSystem: "",
	CPUFamily:              "5",
	CPUModel:               "33",
	CPUModelName:           "Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz",
	CPUVendor:              "GenuineIntel",
	NumCPUs:                4,
	MemoryTotal:            1500000,
	Timezone:               "CEST (UTC+00:00)",
	Hostname:               "alpine-3-10-tk-02",
	IPv4:                   []string{"192.168.122.112"},
	IPv6:                   []string{"fe80::b84f:aff:fe59:a0b2"},
	Tags:                   []string{"Linux", "Datacenter 2"},
	Version:                "0.1.12",
	Address:                "88.198.189.162:50078",
	Tunnels: []*Tunnel{
		{
			ID: "1",
			Remote: models.Remote{
				LocalHost:  "0.0.0.0",
				LocalPort:  "2222",
				RemoteHost: "0.0.0.0",
				RemotePort: "22",
			},
		},
	},
	ClientAuthID:   "client-2",
	DisconnectedAt: &c2DisconnectedTime,
}

var c3 = &Client{
	ID:             "c1d3c6811e1282c675495c0b3149dfa3201883188c42727a318d4a0742564c96",
	Name:           "Random Rport Client 3",
	OS:             "Linux alpine-3-10-tk-03 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
	OSArch:         "amd64",
	OSFamily:       "alpine",
	OSKernel:       "linux",
	Hostname:       "alpine-3-10-tk-03",
	IPv4:           []string{"192.168.122.113"},
	IPv6:           []string{"fe80::b84f:aff:fe59:a0b3"},
	Tags:           []string{"Linux", "Datacenter 3"},
	Version:        "0.1.12",
	Address:        "88.198.189.163:50078",
	Tunnels:        make([]*Tunnel, 0),
	ClientAuthID:   "client-3",
	DisconnectedAt: &c3DisconnectedTime,
}

var c4 = &Client{
	ID:             "7d2e0e7b92115970d0aef41b8e23c080e3c41df10a042c5179c79973ae5bd235",
	Name:           "Random Rport Client 4",
	OS:             "Linux alpine-3-10-tk-04",
	OSArch:         "amd64",
	OSFamily:       "alpine",
	OSKernel:       "linux",
	Hostname:       "alpine-3-10-tk-04",
	IPv4:           []string{"192.168.122.114"},
	IPv6:           []string{"fe80::b84f:aff:fe59:a0b4"},
	Tags:           []string{"Linux", "Datacenter 4"},
	Version:        "0.1.12",
	Address:        "88.198.189.164:50078",
	Tunnels:        make([]*Tunnel, 0),
	ClientAuthID:   "client-4",
	DisconnectedAt: &c4DisconnectedTime,
}

var c5 = &Client{
	ID:                     "daflkdfjqlkerlkejrqlwedalfdfadfa",
	Name:                   "Windows Client",
	OS:                     "Windows",
	OSArch:                 "x86_64",
	OSFamily:               "Server",
	OSKernel:               "10.0.1 4393 Build 14393",
	OSFullName:             "Microsoft Windows Server 2016 Standard",
	OSVersion:              "10.0.14393 Build 14393",
	OSVirtualizationRole:   "",
	OSVirtualizationSystem: "",
	CPUFamily:              "1",
	CPUModel:               "4",
	CPUModelName:           "Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz",
	CPUVendor:              "GenuineIntel",
	NumCPUs:                2,
	MemoryTotal:            4294422528,
	Timezone:               "PDT (UTC-07:00)",
	Hostname:               "RPORT-WIN-SRV2016",
	IPv4:                   []string{"192.168.122.124"},
	IPv6:                   []string{"fe80::b84f:aff:fe56:a0b4"},
	Tags:                   []string{"Linux", "Datacenter 4"},
	Version:                "0.1.12",
	Address:                "88.198.189.124:50078",
	Tunnels:                make([]*Tunnel, 0),
	ClientAuthID:           "client-5",
	DisconnectedAt:         &c4DisconnectedTime,
}

// shallowCopy is used only in tests.
func shallowCopy(c *Client) *Client {
	if c == nil {
		return nil
	}

	return &Client{
		NumCPUs:                c.NumCPUs,
		MemoryTotal:            c.MemoryTotal,
		ID:                     c.ID,
		Name:                   c.Name,
		OS:                     c.OS,
		OSArch:                 c.OSArch,
		OSFamily:               c.OSFamily,
		OSKernel:               c.OSKernel,
		OSFullName:             c.OSFullName,
		OSVersion:              c.OSVersion,
		OSVirtualizationSystem: c.OSVirtualizationSystem,
		OSVirtualizationRole:   c.OSVirtualizationRole,
		CPUFamily:              c.CPUFamily,
		CPUModel:               c.CPUModel,
		CPUModelName:           c.CPUModelName,
		CPUVendor:              c.CPUVendor,
		Timezone:               c.Timezone,
		Hostname:               c.Hostname,
		IPv4:                   append([]string{}, c.IPv4...),
		IPv6:                   append([]string{}, c.IPv6...),
		Tags:                   append([]string{}, c.Tags...),
		Version:                c.Version,
		Address:                c.Address,
		Tunnels:                append([]*Tunnel{}, c.Tunnels...),
		DisconnectedAt:         c.DisconnectedAt,
		ClientAuthID:           c.ClientAuthID,
	}
}

func newFakeClientProvider(t *testing.T, exp time.Duration, cs ...*Client) *SqliteProvider {
	db, err := sqlite.New(":memory:", clients.AssetNames(), clients.Asset)
	require.NoError(t, err)
	p := newSqliteProvider(db, &exp)
	for _, cur := range cs {
		require.NoError(t, p.Save(context.Background(), cur))
	}
	return p
}
