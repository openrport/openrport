package clients

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/server/cgroups"
)

func TestClientBelongsToGroup(t *testing.T) {
	c1 := &Client{
		ID:           "test-client-id-1",
		Name:         "Random Rport Client 1",
		OS:           "Linux alpine-3-10-tk-03 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
		OSArch:       "amd64",
		OSFamily:     "alpine",
		OSKernel:     "linux",
		Hostname:     "alpine-1-10-tk-01",
		IPv4:         []string{"192.168.122.113", "192.168.122.114"},
		IPv6:         []string{"fe80::b84f:aff:fe59:a0b3"},
		Tags:         []string{"Linux", "Datacenter 3", "Tag1", "Tag2", "Tag3"},
		Version:      "0.1.12",
		Address:      "88.198.189.163:50078",
		ClientAuthID: "client-auth-1",
	}
	c2 := &Client{
		ID:       "test-client-id-1",
		OS:       "Linux alpine-3-10-tk-03 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
		OSArch:   "amd64",
		OSFamily: "alpine",
		OSKernel: "linux",
		IPv4:     []string{"192.168.122.113", "192.168.122.114"},
		Tags:     []string{"Linux", "Datacenter 3", "Tag1", "Tag2", "Tag3"},
		Version:  "0.1.12",
	}
	g1 := &cgroups.ClientGroup{
		ID: "group-1",
		Params: &cgroups.ClientParams{
			ClientID:     cgroups.ParamValues{"test-client-id-1", "test-client-id-2"},
			Name:         cgroups.ParamValues{"Random Rport Client*", "My Client*"},
			OS:           cgroups.ParamValues{"Linux*"},
			OSArch:       cgroups.ParamValues{"amd64", "darwin", "windows"},
			OSFamily:     cgroups.ParamValues{"alpine", "win*"},
			OSKernel:     cgroups.ParamValues{"linux", "solaris"},
			Hostname:     cgroups.ParamValues{"a*", "l*", "w*"},
			IPv4:         cgroups.ParamValues{"192.168.122.121", "192.168.122.11*"},
			IPv6:         cgroups.ParamValues{"fe80::b84f:aff:fe59:a0b3"},
			Tag:          cgroups.ParamValues{"Linux", "Tag1", "Data*", "Some Tag", "AB*"},
			Version:      cgroups.ParamValues{"0.1.1*"},
			Address:      cgroups.ParamValues{"88.198.189.163*"},
			ClientAuthID: cgroups.ParamValues{"client-auth-1", "client-auth-2", "client-auth-3*"},
		},
	}
	g2 := &cgroups.ClientGroup{
		ID: "group-1",
		Params: &cgroups.ClientParams{
			ClientID:     cgroups.ParamValues{"test-client-id-1", "test-client-id-2"},
			OS:           cgroups.ParamValues{"Linux*"},
			IPv4:         cgroups.ParamValues{"192.168.122.121", "192.168.122.11*"},
			Version:      cgroups.ParamValues{"0.1.1*"},
			ClientAuthID: cgroups.ParamValues{"client-auth-1", "client-auth-2", "client-auth-3*"},
		},
	}
	testCases := []struct {
		name string

		client *Client
		group  *cgroups.ClientGroup

		wantRes bool
	}{
		{
			name: "all group params, all client params",

			client: c1,
			group:  g1,

			wantRes: true,
		},
		{
			name: "all group params, not all client params",

			client: c2,
			group:  g1,

			wantRes: true,
		},
		{
			name: "not all group params, not all client params",

			client: c2,
			group:  g2,

			wantRes: true,
		},
		{
			name: "not all group params, all client params",

			client: c1,
			group:  g2,

			wantRes: true,
		},
		{
			name: "one param does not match",

			client: c1,
			group: &cgroups.ClientGroup{
				ID: "group-1",
				Params: &cgroups.ClientParams{
					ClientID: cgroups.ParamValues{"test-client-id-1", "test-client-id-2"},
					Name:     cgroups.ParamValues{"Random Rport Client*", "My Client*"},
					OS:       cgroups.ParamValues{"Linux*"},
					Tag:      cgroups.ParamValues{"Some Tag", "AB*"},
				},
			},

			wantRes: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			gotRes := tc.client.belongsTo(tc.group)

			// then
			assert.Equal(t, tc.wantRes, gotRes)
		})
	}
}
