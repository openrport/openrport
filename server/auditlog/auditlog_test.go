package auditlog

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/db/migration/auditlog"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/server/clients"
)

var DataSourceOptions = sqlite.DataSourceOptions{WALEnabled: false}

func TestNotEnabled(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	mockProvider := &mockProvider{}
	auditLog, err := New(nil, nil, "", Config{Enable: false}, DataSourceOptions)
	require.NoError(t, err)
	auditLog.provider = mockProvider

	// Call with all methods to make sure it doesn't panic if not initialized
	e := auditLog.Entry(ApplicationAuthUser, ActionCreate).
		WithID(123).
		WithHTTPRequest(req).
		WithRequest(map[string]interface{}{}).
		WithResponse(map[string]interface{}{}).
		WithClient(&clients.Client{}).
		WithClientID("123")

	e.Save()
	e.SaveForMultipleClients([]*clients.Client{&clients.Client{}})

	assert.Equal(t, 0, len(mockProvider.entries))
}

func TestIPObfuscation(t *testing.T) {
	testCases := []struct {
		RemoteIP            string
		ExpectedObfuscation string
	}{
		{
			RemoteIP:            "192.0.2.123",
			ExpectedObfuscation: "192.0.2.x",
		}, {
			RemoteIP:            "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			ExpectedObfuscation: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		}, {
			RemoteIP:            "example.com",
			ExpectedObfuscation: "example.com",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.RemoteIP, func(t *testing.T) {
			t.Parallel()

			t.Run("with obfuscation", func(t *testing.T) {
				mockProvider := &mockProvider{}
				auditLog := &AuditLog{
					config: Config{
						Enable:           true,
						UseIPObfuscation: true,
					},
					provider: mockProvider,
				}

				e := auditLog.Entry("", "")
				e.RemoteIP = tc.RemoteIP
				e.Save()

				assert.Equal(t, tc.ExpectedObfuscation, mockProvider.entries[0].RemoteIP)
			})

			t.Run("without obfuscation", func(t *testing.T) {
				mockProvider := &mockProvider{}
				auditLog := &AuditLog{
					config: Config{
						Enable:           true,
						UseIPObfuscation: false,
					},
					provider: mockProvider,
				}

				e := auditLog.Entry("", "")
				e.RemoteIP = tc.RemoteIP
				e.Save()

				assert.Equal(t, tc.RemoteIP, mockProvider.entries[0].RemoteIP)
			})
		})
	}
}

func TestList(t *testing.T) {
	db, err := sqlite.New(":memory:", auditlog.AssetNames(), auditlog.Asset, DataSourceOptions)
	require.NoError(t, err)
	dbProv := &SQLiteProvider{
		db: db,
	}
	auditLog := &AuditLog{
		config: Config{
			Enable: true,
		},
		provider: dbProv,
	}
	defer auditLog.Close()

	auditLog.Entry(ApplicationLibraryScript, ActionCreate).Save()
	auditLog.Entry(ApplicationLibraryScript, ActionUpdate).Save()
	auditLog.Entry(ApplicationLibraryCommand, ActionCreate).Save()

	r := httptest.NewRequest("GET", "/auditlog?filter[application]=library.script&sort=-timestamp&page[limit]=1&page[offset]=1", nil)
	result, err := auditLog.List(r)
	require.NoError(t, err)

	assert.Equal(t, 2, result.Meta.Count)
	entries := result.Data.([]*Entry)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, ApplicationLibraryScript, entries[0].Application)
	assert.Equal(t, ActionCreate, entries[0].Action)
}
