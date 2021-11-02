package auditlog

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/share/query"
)

func TestEntry(t *testing.T) {
	auditLog := enabledAuditLog()
	e := auditLog.Entry(ApplicationClient, ActionCreate)

	assert.WithinDuration(t, time.Now(), e.Timestamp, time.Millisecond)
	assert.Equal(t, ApplicationClient, e.Application)
	assert.Equal(t, ActionCreate, e.Action)
}

func TestWithID(t *testing.T) {
	testCases := []struct {
		Name   string
		ID     interface{}
		wantID string
	}{
		{
			Name:   "string",
			ID:     "abc",
			wantID: "abc",
		}, {
			Name:   "uuid",
			ID:     uuid.MustParse("11236310-6cad-408e-b372-a0f04d68d2df"),
			wantID: "11236310-6cad-408e-b372-a0f04d68d2df",
		}, {
			Name:   "number",
			ID:     123,
			wantID: "123",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			e := emptyEntry().WithID(tc.ID)

			assert.Equal(t, tc.wantID, e.ID)
		})
	}
}

func TestWithHTTPRequest(t *testing.T) {
	ctx := api.WithUser(context.Background(), "test-user")
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(ctx)

	e := emptyEntry().WithHTTPRequest(req)

	assert.Equal(t, "test-user", e.Username)
	assert.Equal(t, "192.0.2.1", e.RemoteIP)
}

func TestWithRequest(t *testing.T) {
	e := emptyEntry().WithRequest(map[string]interface{}{
		"k1": "v1",
		"k2": 2,
	})

	assert.JSONEq(t, `{"k1": "v1", "k2": 2}`, e.Request)
}

func TestWithResponse(t *testing.T) {
	e := emptyEntry().WithResponse(map[string]interface{}{
		"k1": "v1",
		"k2": 2,
	})

	assert.JSONEq(t, `{"k1": "v1", "k2": 2}`, e.Response)
}

func TestWithClient(t *testing.T) {
	e := emptyEntry().WithClient(&clients.Client{
		ID:      "11236310-6cad-408e-b372-a0f04d68d2df",
		Address: "127.0.0.1",
	})

	assert.Equal(t, "11236310-6cad-408e-b372-a0f04d68d2df", e.ClientID)
	assert.Equal(t, "127.0.0.1", e.ClientHostName)
}

func TestWithClientID(t *testing.T) {
	auditLog := enabledAuditLog()
	auditLog.clientGetter = &mockClientGetter{}

	t.Run("client exists", func(t *testing.T) {
		e := auditLog.Entry("", "").WithClientID("11236310-6cad-408e-b372-a0f04d68d2df")
		assert.Equal(t, "11236310-6cad-408e-b372-a0f04d68d2df", e.ClientID)
		assert.Equal(t, "127.0.0.1", e.ClientHostName)
	})

	t.Run("client does not exist", func(t *testing.T) {
		e := auditLog.Entry("", "").WithClientID("2e7bf232-b4aa-4cdb-a7bb-d28f63712c2d")
		assert.Equal(t, "2e7bf232-b4aa-4cdb-a7bb-d28f63712c2d", e.ClientID)
		assert.Equal(t, "", e.ClientHostName)
	})
}

func TestSave(t *testing.T) {
	mockProvider := &mockProvider{}
	auditLog := enabledAuditLog()

	t.Run("with provider", func(t *testing.T) {
		auditLog.provider = mockProvider

		e := auditLog.Entry(ApplicationClient, ActionCreate)
		e.Save()

		assert.Equal(t, []Entry{*e}, mockProvider.entries)
	})

	t.Run("with nil provider", func(t *testing.T) {
		auditLog.provider = nil

		auditLog.Entry("", "").Save()
	})
}

func TestSaveForMultipleClients(t *testing.T) {
	mockProvider := &mockProvider{}
	auditLog := enabledAuditLog()
	auditLog.provider = mockProvider

	auditLog.Entry("", "").SaveForMultipleClients([]*clients.Client{
		{
			ID:      "c1",
			Address: "c1.com",
		},
		{
			ID:      "c2",
			Address: "c2.com",
		},
	})

	assert.Len(t, mockProvider.entries, 2)
	assert.Equal(t, "c1", mockProvider.entries[0].ClientID)
	assert.Equal(t, "c1.com", mockProvider.entries[0].ClientHostName)
	assert.Equal(t, "c2", mockProvider.entries[1].ClientID)
	assert.Equal(t, "c2.com", mockProvider.entries[1].ClientHostName)
}

func enabledAuditLog() *AuditLog {
	return &AuditLog{
		config: Config{
			Enable: true,
		},
	}
}

func emptyEntry() *Entry {
	return enabledAuditLog().Entry("", "")
}

type mockClientGetter struct {
}

func (mockClientGetter) GetByID(id string) (*clients.Client, error) {
	if id == "11236310-6cad-408e-b372-a0f04d68d2df" {
		return &clients.Client{
			ID:      "11236310-6cad-408e-b372-a0f04d68d2df",
			Address: "127.0.0.1",
		}, nil
	}
	return nil, nil
}

type mockProvider struct {
	entries []Entry
}

func (p *mockProvider) Save(e *Entry) error {
	p.entries = append(p.entries, *e)
	return nil
}

func (p *mockProvider) List(ctx context.Context, opts *query.ListOptions) ([]*Entry, error) {
	return nil, nil
}

func (p mockProvider) Close() error { return nil }
