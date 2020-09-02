package sessions

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var emptyFile = ``
var jsonEmptyArray = `[]`
var jsonWithThreeClients = fmt.Sprintf("[%s,%s,%s]", s1JSON, s2JSON, s3JSON)
var jsonCorruptedWithOneClient = fmt.Sprintf("[%s,%s", s1JSON, `
 {
   "id": "2fb5eca74d7bdf5f5b879ebadb446af7c113b076354d74e1882d8101e9f4b918",
   "name": "Random Rport Client 2",
   "os": "Linux alpine-3-10-tk-02 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
   "host`)

func TestGetInitState(t *testing.T) {
	now = nowMockF

	wantS1 := shallowCopy(s1)
	wantS1.Disconnected = &nowMock

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
			csrJSONBytes:    jsonWithThreeClients,
			wantRes:         []*ClientSession{wantS1, s2},
			wantErrContains: "",
			expiration:      &hour,
		},
		{
			descr:           "1 connected, 2 disconnected with unset expiration",
			csrJSONBytes:    jsonWithThreeClients,
			wantRes:         []*ClientSession{wantS1, s2, s3},
			wantErrContains: "",
			expiration:      nil,
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
			wantRes:         []*ClientSession{wantS1},
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
