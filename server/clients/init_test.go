package clients

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetInitState(t *testing.T) {
	ctx := context.Background()
	c1 := New(t).ID("client-1").Logger(testLog).Build()
	wantC1 := shallowCopy(c1)

	wantC1.SetDisconnectedAt(&nowMock)
	c2 := New(t).ID("client-2").DisconnectedDuration(5 * time.Minute).Logger(testLog).Build()
	c3 := New(t).ID("client-3").DisconnectedDuration(2 * time.Hour).Logger(testLog).Build()

	testCases := []struct {
		name string

		dbClients  []*Client
		expiration time.Duration
		wantRes    []*Client
	}{
		{
			name:      "no clients",
			dbClients: nil,
			wantRes:   nil,
		},
		{
			name:       "1 connected, 1 disconnected, 1 obsolete",
			dbClients:  []*Client{c1, c2, c3},
			wantRes:    []*Client{wantC1, c2},
			expiration: hour,
		},
		{
			name:       "1 connected, 2 disconnected, 0 expiration",
			dbClients:  []*Client{c1, c2, c3},
			wantRes:    []*Client{wantC1},
			expiration: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewFakeClientProvider(t, &tc.expiration, tc.dbClients...)
			defer p.Close()

			gotClients, gotErr := LoadInitialClients(ctx, p, testLog)
			assert.NoError(t, gotErr)
			assert.Len(t, gotClients, len(tc.wantRes))

			// patch the client logger for the ElementsMatch check
			for _, c := range gotClients {
				c.Logger = testLog
			}

			assert.ElementsMatch(t, gotClients, tc.wantRes)
		})
	}
}
