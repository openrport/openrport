package clients

import (
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/server/cgroups"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/share/query"
)

type UserMock struct {
	ReturnIsAdmin bool
	ReturnGroups  []string
}

func (u UserMock) IsAdmin() bool {
	return u.ReturnIsAdmin
}

func (u UserMock) GetGroups() []string {
	return u.ReturnGroups
}

var admin = UserMock{
	ReturnIsAdmin: true,
}

func TestCRWithExpiration(t *testing.T) {
	now = nowMockF

	exp := 2 * time.Hour
	repo := NewClientRepository([]*Client{c1, c2}, &exp, testLog)

	assert := assert.New(t)
	assert.NoError(repo.Save(c3))
	assert.NoError(repo.Save(c4))

	gotCount := repo.Count()
	assert.Equal(3, gotCount)

	gotCountActive := repo.CountActive()
	assert.Equal(1, gotCountActive)

	gotCountDisconnected, err := repo.CountDisconnected()
	assert.NoError(err)
	assert.Equal(2, gotCountDisconnected)

	gotClients := repo.GetAllClients()
	assert.ElementsMatch([]*Client{c1, c2, c3}, gotClients)

	// active
	gotClient, err := repo.GetActiveByID(c1.GetID())
	assert.NoError(err)
	assert.Equal(c1, gotClient)

	// disconnected
	gotClient, err = repo.GetActiveByID(c2.GetID())
	assert.NoError(err)
	assert.Nil(gotClient)

	deleted, err := repo.DeleteObsolete()
	assert.NoError(err)
	require.Len(t, deleted, 1)
	assert.Equal(c4, deleted[0])
	gotClients = repo.GetAllClients()
	assert.ElementsMatch([]*Client{c1, c2, c3}, gotClients)

	assert.NoError(repo.Delete(c3))
	gotClients = repo.GetAllClients()
	assert.NoError(err)
	assert.ElementsMatch([]*Client{c1, c2}, gotClients)
}

func TestCRWithNoExpiration(t *testing.T) {
	now = nowMockF

	repo := NewClientRepository([]*Client{c1, c2, c3}, nil, testLog)
	c4Active := shallowCopy(c4)
	c4Active.DisconnectedAt = nil

	assert := assert.New(t)
	assert.NoError(repo.Save(c4Active))

	gotCount := repo.Count()
	assert.Equal(4, gotCount)

	gotCountActive := repo.CountActive()
	assert.Equal(2, gotCountActive)

	gotCountDisconnected, err := repo.CountDisconnected()
	assert.NoError(err)
	assert.Equal(2, gotCountDisconnected)

	gotClients := repo.GetAllClients()
	assert.ElementsMatch([]*Client{c1, c2, c3, c4Active}, gotClients)

	// active
	gotClient, err := repo.GetActiveByID(c1.GetID())
	assert.NoError(err)
	assert.Equal(c1, gotClient)

	// disconnected
	gotClient, err = repo.GetActiveByID(c2.GetID())
	assert.NoError(err)
	assert.Nil(gotClient)

	deleted, err := repo.DeleteObsolete()
	assert.NoError(err)
	assert.Len(deleted, 0)

	assert.NoError(repo.Delete(c4Active))
	gotClients = repo.GetAllClients()
	assert.ElementsMatch([]*Client{c1, c2, c3}, gotClients)
}

func TestCRWithFilter(t *testing.T) {
	testCases := []struct {
		name              string
		filters           []query.FilterOption
		expectedClientIDs []string
	}{
		{
			name: "single value",
			filters: []query.FilterOption{
				{
					Column: []string{"os_full_name"},
					Values: []string{
						"Alpine Linux",
					},
				},
			},
			expectedClientIDs: []string{
				"2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
			},
		},
		{
			name: "case insensitive",
			filters: []query.FilterOption{
				{
					Column: []string{"os_full_name"},
					Values: []string{
						"aLpInE lInUx",
					},
				},
			},
			expectedClientIDs: []string{
				"2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
			},
		},
		{
			name: "wildcard",
			filters: []query.FilterOption{
				{
					Column: []string{"os_full_name"},
					Values: []string{
						"Alpine*",
					},
				},
			},
			expectedClientIDs: []string{
				"aa1210c7-1899-491e-8e71-564cacaf1df8",
				"2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
			},
		},
		{
			name: "wildcard case insensitive",
			filters: []query.FilterOption{
				{
					Column: []string{"os_full_name"},
					Values: []string{
						"aLpInE*",
					},
				},
			},
			expectedClientIDs: []string{
				"aa1210c7-1899-491e-8e71-564cacaf1df8",
				"2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
			},
		},
		{
			name: "array value",
			filters: []query.FilterOption{
				{
					Column: []string{"ipv4"},
					Values: []string{
						"192.168.122.111",
					},
				},
			},
			expectedClientIDs: []string{
				"aa1210c7-1899-491e-8e71-564cacaf1df8",
			},
		},
		{
			name: "multiple values",
			filters: []query.FilterOption{
				{
					Column: []string{"os_full_name"},
					Values: []string{
						"Alpine*",
						"Microsoft Windows Server 2016 Standard",
					},
				},
			},
			expectedClientIDs: []string{
				"2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
				"aa1210c7-1899-491e-8e71-564cacaf1df8",
				"daflkdfjqlkerlkejrqlwedalfdfadfa",
			},
		},
		{
			name: "multiple filters",
			filters: []query.FilterOption{
				{
					Column: []string{"os_virtualization_system"},
					Values: []string{
						"KVM",
						"Microsoft Windows Server 2016 Standard",
					},
				},
				{
					Column: []string{"os_virtualization_role"},
					Values: []string{
						"guest",
					},
				},
			},
			expectedClientIDs: []string{
				"aa1210c7-1899-491e-8e71-564cacaf1df8",
			},
		},
		{
			name: "or columns",
			filters: []query.FilterOption{
				{
					Column: []string{"os_full_name", "ipv4"},
					Values: []string{
						"Microsoft Windows Server 2016 Standard",
						"192.168.*.111",
					},
				},
			},
			expectedClientIDs: []string{
				"daflkdfjqlkerlkejrqlwedalfdfadfa",
				"aa1210c7-1899-491e-8e71-564cacaf1df8",
			},
		},
		{
			name: "no results",
			filters: []query.FilterOption{
				{
					Column: []string{"os_full_name"},
					Values: []string{
						"Oracle",
					},
				},
			},
			expectedClientIDs: []string{},
		},
		{
			name: "or inside tags, old notation",
			filters: []query.FilterOption{
				{
					Column: []string{"tags"},
					Values: []string{
						"Datacenter 4",
						"Linux",
					},
				},
			},
			expectedClientIDs: []string{
				"daflkdfjqlkerlkejrqlwedalfdfadfa",
				"aa1210c7-1899-491e-8e71-564cacaf1df8",
				"2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
			},
		},
		{
			name: "or inside tags, new notation",
			filters: []query.FilterOption{
				{
					Column:                []string{"tags"},
					ValuesLogicalOperator: query.FilterLogicalOperatorTypeOR,
					Values: []string{
						"Datacenter 4",
						"Linux",
					},
				},
			},
			expectedClientIDs: []string{
				"daflkdfjqlkerlkejrqlwedalfdfadfa",
				"aa1210c7-1899-491e-8e71-564cacaf1df8",
				"2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
			},
		},
		{
			name: "and inside tags, one result",
			filters: []query.FilterOption{
				{
					Column:                []string{"tags"},
					ValuesLogicalOperator: query.FilterLogicalOperatorTypeAND,
					Values: []string{
						"Datacenter 4",
						"Linux",
					},
				},
			},
			expectedClientIDs: []string{
				"daflkdfjqlkerlkejrqlwedalfdfadfa",
			},
		},
		{
			name: "and inside tags, three operands no result",
			filters: []query.FilterOption{
				{
					Column:                []string{"tags"},
					ValuesLogicalOperator: query.FilterLogicalOperatorTypeAND,
					Values: []string{
						"Datacenter 4",
						"Datacenter 2",
						"Linux",
					},
				},
			},
			expectedClientIDs: []string{},
		},
		{
			name: "and inside tags, three operands, wide wildcards, one result",
			filters: []query.FilterOption{
				{
					Column:                []string{"tags"},
					ValuesLogicalOperator: query.FilterLogicalOperatorTypeAND,
					Values: []string{
						"*n*",
						"*a*",
						"Datacenter 2",
					},
				},
			},
			expectedClientIDs: []string{
				"2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
			},
		},

		{
			name: "all filters",
			filters: []query.FilterOption{
				{
					Column: []string{"id"},
					Values: []string{
						"a*",
					},
				},
				{
					Column: []string{"name"},
					Values: []string{
						"*Client*",
					},
				},
				{
					Column: []string{"os"},
					Values: []string{
						"Linux*",
					},
				},
				{
					Column: []string{"os_arch"},
					Values: []string{
						"amd64",
					},
				},
				{
					Column: []string{"os_family"},
					Values: []string{
						"alpine",
					},
				},
				{
					Column: []string{"os_kernel"},
					Values: []string{
						"linux",
					},
				},
				{
					Column: []string{"os_full_name"},
					Values: []string{
						"Alpine",
					},
				},
				{
					Column: []string{"os_version"},
					Values: []string{
						"3.14.*",
					},
				},
				{
					Column: []string{"os_virtualization_role"},
					Values: []string{
						"guest",
					},
				},
				{
					Column: []string{"os_virtualization_system"},
					Values: []string{
						"KVM",
					},
				},
				{
					Column: []string{"cpu_family"},
					Values: []string{
						"6",
					},
				},
				{
					Column: []string{"cpu_model"},
					Values: []string{
						"79",
					},
				},
				{
					Column: []string{"cpu_model_name"},
					Values: []string{
						"*KVM*",
					},
				},
				{
					Column: []string{"cpu_vendor"},
					Values: []string{
						"GenuineIntel",
					},
				},
				{
					Column: []string{"num_cpus"},
					Values: []string{
						"2",
					},
				},
				{
					Column: []string{"timezone"},
					Values: []string{
						"CEST*",
					},
				},
				{
					Column: []string{"hostname"},
					Values: []string{
						"alpine-*",
					},
				},
				{
					Column: []string{"ipv4"},
					Values: []string{
						"192.168.*.*",
					},
				},
				{
					Column: []string{"ipv6"},
					Values: []string{
						"fe80::b84f:aff:fe59:a0b1",
					},
				},
				{
					Column: []string{"tags"},
					Values: []string{
						"Datacenter 1",
					},
				},
				{
					Column: []string{"labels"},
					Values: []string{"country: Germany", "city: Cologne", "datacenter: NetCologne GmbH"},
				},
				{
					Column: []string{"version"},
					Values: []string{
						"0.1.12",
					},
				},
				{
					Column: []string{"address"},
					Values: []string{
						"88.198.189.161:50078",
					},
				},
				{
					Column: []string{"client_auth_id"},
					Values: []string{
						"client-1",
					},
				},
				{
					Column: []string{"allowed_user_groups"},
					Values: []string{
						"Administrators",
					},
				},
			},
			expectedClientIDs: []string{
				"aa1210c7-1899-491e-8e71-564cacaf1df8",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := NewClientRepository([]*Client{c1, c2, c5}, nil, testLog)

			actualClients, err := repo.GetFilteredUserClients(admin, tc.filters, nil)
			require.NoError(t, err)

			actualClientIDs := make([]string, 0, len(actualClients))

			for _, actualClient := range actualClients {
				actualClientIDs = append(actualClientIDs, actualClient.GetID())
			}

			assert.ElementsMatch(t, tc.expectedClientIDs, actualClientIDs)
		})
	}
}

func TestCRWithUnsupportedFilter(t *testing.T) {
	repo := NewClientRepository([]*Client{c1}, nil, testLog)
	_, err := repo.GetFilteredUserClients(admin, []query.FilterOption{
		{
			Column: []string{"unknown_field"},
			Values: []string{
				"1",
			},
		},
	}, nil)
	require.EqualError(t, err, "unsupported filter column: unknown_field")
}

func TestGetUserClients(t *testing.T) {
	c1 := New(t).Logger(testLog).Build()                                                             // no groups
	c2 := New(t).AllowedUserGroups([]string{users.Administrators}).Logger(testLog).Build()           // admin
	c3 := New(t).AllowedUserGroups([]string{users.Administrators, "group1"}).Logger(testLog).Build() // admin + group1
	c4 := New(t).AllowedUserGroups([]string{"group1"}).Logger(testLog).Build()                       // group1
	c5 := New(t).AllowedUserGroups([]string{"group1", "group2"}).Logger(testLog).Build()             // group1 + group2
	c6 := New(t).AllowedUserGroups([]string{"group2"}).Logger(testLog).Build()                       // group2
	c7 := New(t).AllowedUserGroups([]string{"group3"}).Logger(testLog).Build()                       // group3
	c8 := New(t).AllowedUserGroups([]string{"group2", "group3"}).Logger(testLog).Build()             // group2 + group3
	c9 := New(t).Logger(testLog).Build()
	allClients := []*Client{c1, c2, c3, c4, c5, c6, c7, c8, c9}

	clientGroups := []*cgroups.ClientGroup{
		{
			ID:                "1",
			AllowedUserGroups: []string{"group6"},
			Params: &cgroups.ClientParams{
				ClientID: &cgroups.ParamValues{cgroups.Param(c9.GetID())},
			},
		},
	}

	repo := NewClientRepository(allClients, nil, testLog)
	testCases := []struct {
		name          string
		user          User
		wantClientIDs []*Client
	}{
		{
			name:          "admin user",
			user:          admin,
			wantClientIDs: allClients,
		},
		{
			name:          "user with no groups has no access",
			user:          &UserMock{ReturnGroups: nil},
			wantClientIDs: []*Client{},
		},
		{
			name:          "user with unknown group",
			user:          &UserMock{ReturnGroups: []string{"unknown"}},
			wantClientIDs: []*Client{},
		},
		{
			name:          "non-admin user with access to few clients",
			user:          &users.User{Groups: []string{"group1", "group2"}},
			wantClientIDs: []*Client{c3, c4, c5, c6, c8},
		},
		{
			name:          "non-admin user with access via client groups",
			user:          &users.User{Groups: []string{"group6"}},
			wantClientIDs: []*Client{c9},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			gotClients := repo.GetUserClients(tc.user, clientGroups)
			t.Logf("Access ganted to %d clients", len(gotClients))

			// then
			assert.ElementsMatch(t, tc.wantClientIDs, gotClients)
		})
	}
}

func TestGetClientByTag(t *testing.T) {
	// clients from data_test.go
	availableClients := []*Client{c1, c2, c3, c4, c5}
	cases := []struct {
		name              string
		tags              []string
		operator          string
		expectedClientIDs []string
	}{
		{
			name:     "single tag",
			tags:     []string{"Datacenter 4"},
			operator: "OR",
			expectedClientIDs: []string{
				"7d2e0e7b92115970d0aef41b8e23c080e3c41df10a042c5179c79973ae5bd235",
				"daflkdfjqlkerlkejrqlwedalfdfadfa",
			},
		},
		{
			name:     "more tags with OR",
			tags:     []string{"Datacenter 3", "Datacenter 4"},
			operator: "OR",
			expectedClientIDs: []string{
				"c1d3c6811e1282c675495c0b3149dfa3201883188c42727a318d4a0742564c96",
				"7d2e0e7b92115970d0aef41b8e23c080e3c41df10a042c5179c79973ae5bd235",
				"daflkdfjqlkerlkejrqlwedalfdfadfa",
			},
		},
		{
			name:     "more tags with AND",
			tags:     []string{"Datacenter 3", "Linux"},
			operator: "AND",
			expectedClientIDs: []string{
				"c1d3c6811e1282c675495c0b3149dfa3201883188c42727a318d4a0742564c96",
			},
		},
		{
			name:     "even more tags with AND",
			tags:     []string{"Datacenter 3", "Linux", "Datacenter 4"},
			operator: "AND",
			expectedClientIDs: []string{
				"7d2e0e7b92115970d0aef41b8e23c080e3c41df10a042c5179c79973ae5bd235",
				"c1d3c6811e1282c675495c0b3149dfa3201883188c42727a318d4a0742564c96",
				"daflkdfjqlkerlkejrqlwedalfdfadfa",
			},
		},
		{
			name:     "duplicate tags with AND",
			tags:     []string{"Datacenter 3", "Linux", "Linux"},
			operator: "AND",
			expectedClientIDs: []string{
				"c1d3c6811e1282c675495c0b3149dfa3201883188c42727a318d4a0742564c96",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var matchingClients []*Client
			if tc.operator == "AND" {
				matchingClients = findMatchingANDClients(availableClients, tc.tags)
			} else {
				matchingClients = findMatchingORClients(availableClients, tc.tags)
			}

			for idx, cl := range matchingClients {
				assert.Equal(t, tc.expectedClientIDs[idx], cl.GetID())
			}
		})
	}
}
