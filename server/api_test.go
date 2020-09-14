package chserver

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/sessions"
)

func TestGetCorrespondingSortFuncPositive(t *testing.T) {
	testCases := []struct {
		sortStr string

		wantFunc func(a []*sessions.ClientSession, desc bool)
		wantDesc bool
	}{
		{
			sortStr:  "",
			wantFunc: sessions.SortByID,
			wantDesc: false,
		},
		{
			sortStr:  "-",
			wantFunc: sessions.SortByID,
			wantDesc: true,
		},
		{
			sortStr:  "id",
			wantFunc: sessions.SortByID,
			wantDesc: false,
		},
		{
			sortStr:  "-id",
			wantFunc: sessions.SortByID,
			wantDesc: true,
		},
		{
			sortStr:  "name",
			wantFunc: sessions.SortByName,
			wantDesc: false,
		},
		{
			sortStr:  "-name",
			wantFunc: sessions.SortByName,
			wantDesc: true,
		},
		{
			sortStr:  "hostname",
			wantFunc: sessions.SortByHostname,
			wantDesc: false,
		},
		{
			sortStr:  "-hostname",
			wantFunc: sessions.SortByHostname,
			wantDesc: true,
		},
		{
			sortStr:  "os",
			wantFunc: sessions.SortByOS,
			wantDesc: false,
		},
		{
			sortStr:  "-os",
			wantFunc: sessions.SortByOS,
			wantDesc: true,
		},
	}

	for _, tc := range testCases {
		// when
		gotFunc, gotDesc, gotErr := getCorrespondingSortFunc(tc.sortStr)

		// then
		// workaround to compare func vars, see https://github.com/stretchr/testify/issues/182
		wantFuncName := runtime.FuncForPC(reflect.ValueOf(tc.wantFunc).Pointer()).Name()
		gotFuncName := runtime.FuncForPC(reflect.ValueOf(gotFunc).Pointer()).Name()
		msg := fmt.Sprintf("getCorrespondingSortFunc(%q) = (%s, %v, %v), expected: (%s, %v, %v)", tc.sortStr, gotFuncName, gotDesc, gotErr, wantFuncName, tc.wantDesc, nil)

		assert.NoErrorf(t, gotErr, msg)
		assert.Equalf(t, wantFuncName, gotFuncName, msg)
		assert.Equalf(t, tc.wantDesc, gotDesc, msg)
	}
}

func TestGetCorrespondingSortFuncNegative(t *testing.T) {
	// when
	_, _, gotErr := getCorrespondingSortFunc("unknown")

	// then
	require.Error(t, gotErr)
	assert.Contains(t, gotErr.Error(), "incorrect format")
}
