package clients

import (
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/share/query"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCRWithExpiration(t *testing.T) {
	now = nowMockF

	exp := 2 * time.Hour
	repo := NewClientRepository([]*Client{c1, c2}, &exp, testLog)

	assert := assert.New(t)
	assert.NoError(repo.Save(c3))
	assert.NoError(repo.Save(c4))

	gotCount, err := repo.Count()
	assert.NoError(err)
	assert.Equal(3, gotCount)

	gotCountActive, err := repo.CountActive()
	assert.NoError(err)
	assert.Equal(1, gotCountActive)

	gotCountDisconnected, err := repo.CountDisconnected()
	assert.NoError(err)
	assert.Equal(2, gotCountDisconnected)

	gotClients, err := repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*Client{c1, c2, c3}, gotClients)

	// active
	gotClient, err := repo.GetActiveByID(c1.ID)
	assert.NoError(err)
	assert.Equal(c1, gotClient)

	// disconnected
	gotClient, err = repo.GetActiveByID(c2.ID)
	assert.NoError(err)
	assert.Nil(gotClient)

	deleted, err := repo.DeleteObsolete()
	assert.NoError(err)
	require.Len(t, deleted, 1)
	assert.Equal(c4, deleted[0])
	gotClients, err = repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*Client{c1, c2, c3}, gotClients)

	assert.NoError(repo.Delete(c3))
	gotClients, err = repo.GetAll()
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

	gotCount, err := repo.Count()
	assert.NoError(err)
	assert.Equal(4, gotCount)

	gotCountActive, err := repo.CountActive()
	assert.NoError(err)
	assert.Equal(2, gotCountActive)

	gotCountDisconnected, err := repo.CountDisconnected()
	assert.NoError(err)
	assert.Equal(2, gotCountDisconnected)

	gotClients, err := repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*Client{c1, c2, c3, c4Active}, gotClients)

	// active
	gotClient, err := repo.GetActiveByID(c1.ID)
	assert.NoError(err)
	assert.Equal(c1, gotClient)

	// disconnected
	gotClient, err = repo.GetActiveByID(c2.ID)
	assert.NoError(err)
	assert.Nil(gotClient)

	deleted, err := repo.DeleteObsolete()
	assert.NoError(err)
	assert.Len(deleted, 0)

	assert.NoError(repo.Delete(c4Active))
	gotClients, err = repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*Client{c1, c2, c3}, gotClients)
}

func TestCRWithFilter(t *testing.T) {
	testCases := []struct {
		filters           []query.FilterOption
		expectedClientIDs []string
	}{
		{
			filters: []query.FilterOption{
				{
					Column: "os_full_name",
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
			filters: []query.FilterOption{
				{
					Column: "os_full_name",
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
			filters: []query.FilterOption{
				{
					Column: "os_full_name",
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
			filters: []query.FilterOption{
				{
					Column: "os_virtualization_system",
					Values: []string{
						"KVM",
						"Microsoft Windows Server 2016 Standard",
					},
				},
				{
					Column: "os_virtualization_role",
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
			filters: []query.FilterOption{
				{
					Column: "os_full_name",
					Values: []string{
						"Oracle",
					},
				},
			},
			expectedClientIDs: []string{},
		},
		{
			filters: []query.FilterOption{
				{
					Column: "os_full_name",
					Values: []string{
						"Microsoft Windows Server 2016 Standard",
					},
				},
				{
					Column: "os_version",
					Values: []string{
						"10.0.14393 Build 14393",
					},
				},
				{
					Column: "cpu_family",
					Values: []string{
						"1",
					},
				},
				{
					Column: "cpu_model",
					Values: []string{
						"4",
					},
				},
				{
					Column: "cpu_model_name",
					Values: []string{
						"Intel*",
					},
				},
				{
					Column: "num_cpus",
					Values: []string{
						"2",
					},
				},
				{
					Column: "timezone",
					Values: []string{
						"UTC*",
					},
				},
			},
			expectedClientIDs: []string{
				"daflkdfjqlkerlkejrqlwedalfdfadfa",
			},
		},
	}

	for _, testcase := range testCases {
		repo := NewClientRepository([]*Client{c1, c2, c5}, nil, testLog)

		actualClients, err := repo.GetFiltered(testcase.filters)
		require.NoError(t, err)

		actualClientIDs := make([]string, 0, len(actualClients))

		for _, actualClient := range actualClients {
			actualClientIDs = append(actualClientIDs, actualClient.ID)
		}

		assert.ElementsMatch(t, testcase.expectedClientIDs, actualClientIDs)
	}
}

func TestCRWithUnsupportedFilter(t *testing.T) {
	repo := NewClientRepository([]*Client{c1}, nil, testLog)
	_, err := repo.GetFiltered([]query.FilterOption{
		{
			Column: "unknown_field",
			Values: []string{
				"1",
			},
		},
	})
	require.EqualError(t, err, "unsupported filter column: unknown_field")
}
