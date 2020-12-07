package sessions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetInitState(t *testing.T) {
	ctx := context.Background()
	s1 := New(t).Build()
	wantS1 := shallowCopy(s1)
	wantS1.Disconnected = &nowMock
	s2 := New(t).DisconnectedDuration(5 * time.Minute).Build()
	s3 := New(t).DisconnectedDuration(2 * time.Hour).Build()

	testCases := []struct {
		name string

		dbSessions []*ClientSession
		expiration time.Duration
		wantRes    []*ClientSession
	}{
		{
			name:       "no sessions",
			dbSessions: nil,
			wantRes:    nil,
		},
		{
			name:       "1 connected, 1 disconnected, 1 obsolete",
			dbSessions: []*ClientSession{s1, s2, s3},
			wantRes:    []*ClientSession{wantS1, s2},
			expiration: hour,
		},
		{
			name:       "1 connected, 2 disconnected, 0 expiration",
			dbSessions: []*ClientSession{s1, s2, s3},
			wantRes:    []*ClientSession{wantS1},
			expiration: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			p := newFakeSessionProvider(t, tc.expiration, tc.dbSessions...)
			defer p.Close()

			// when
			gotSessions, gotErr := GetInitState(ctx, p)

			// then
			assert.NoError(t, gotErr)
			assert.Len(t, gotSessions, len(tc.wantRes))
			assert.ElementsMatch(t, gotSessions, tc.wantRes)
		})
	}
}
