package csr

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInitState(t *testing.T) {
	now = nowMockF
	hour := time.Hour

	testCases := []struct {
		descr string // Test Case Description

		csrJSONBytes string
		expiration   *time.Duration

		wantRes         []*ClientSession
		wantErrContains string // part of an expected error
	}{
		{
			descr:           "empty file",
			csrJSONBytes:    emptyFile,
			wantRes:         nil,
			wantErrContains: "",
		},
		{
			descr:           "empty JSON array",
			csrJSONBytes:    jsonEmptyArray,
			wantRes:         nil,
			wantErrContains: "",
		},
		{
			descr:           "1 connected, 1 disconnected, 1 obsolete",
			csrJSONBytes:    jsonOneEach,
			wantRes:         []*ClientSession{s1, s2},
			wantErrContains: "",
			expiration:      &hour,
		},
		{
			descr:           "corrupted json",
			csrJSONBytes:    `afsdf saf234 sdfe4r`,
			wantRes:         nil,
			wantErrContains: "failed to parse CSR data",
			expiration:      nil,
		},
		{
			descr:           "partially corrupted json at the end, valid 1 connected client",
			csrJSONBytes:    jsonCorruptedWithOneClient,
			wantRes:         []*ClientSession{s1},
			wantErrContains: "failed to parse client session",
			expiration:      &hour,
		},
	}

	for _, tc := range testCases {
		msg := fmt.Sprintf("test case: %q", tc.descr)

		// given
		fileMock := strings.NewReader(tc.csrJSONBytes)

		// when
		gotSessions, gotErr := getInitState(fileMock, tc.expiration)

		// then
		if len(tc.wantErrContains) > 0 {
			require.Errorf(t, gotErr, msg)
			assert.Containsf(t, gotErr.Error(), tc.wantErrContains, msg)
		} else {
			assert.NoErrorf(t, gotErr, msg)
		}
		assert.Lenf(t, gotSessions, len(tc.wantRes), msg)
		assert.ElementsMatch(t, gotSessions, tc.wantRes, msg)
	}
}
