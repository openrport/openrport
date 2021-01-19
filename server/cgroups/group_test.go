package cgroups

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchesOneOf(t *testing.T) {
	testCases := []struct {
		name string

		groupParams  ParamValues
		clientParams []string

		wantRes bool
	}{
		{
			name: "exact match",

			groupParams:  ParamValues{"id-1"},
			clientParams: []string{"id-1"},

			wantRes: true,
		},
		{
			name: "group params contains exact client param",

			groupParams:  ParamValues{"id-1", "id-2", "id-3"},
			clientParams: []string{"id-2"},

			wantRes: true,
		},
		{
			name: "wildcard group param, min client param",

			groupParams:  ParamValues{"id*"},
			clientParams: []string{"id"},

			wantRes: true,
		},
		{
			name: "wildcard group param, short client param",

			groupParams:  ParamValues{"id-*"},
			clientParams: []string{"id-2"},

			wantRes: true,
		},
		{
			name: "wildcard group param, long client param",

			groupParams:  ParamValues{"id-*"},
			clientParams: []string{"id-12345678"},

			wantRes: true,
		},
		{
			name: "group params does no contain client param",

			groupParams:  ParamValues{"id-1", "id-2", "id-3"},
			clientParams: []string{"id-4"},

			wantRes: false,
		},
		{
			name: "group params including wildcard does not contain client param",

			groupParams:  ParamValues{"id-11", "id-22", "id-33", "id-1*", "id-2*"},
			clientParams: []string{"id-3"},

			wantRes: false,
		},
		{
			name: "empty client param",

			groupParams:  ParamValues{"id-1"},
			clientParams: []string{""},

			wantRes: true,
		},
		{
			name: "no group param",

			groupParams:  ParamValues{},
			clientParams: []string{"id-1"},

			wantRes: true,
		},
		{
			name: "plural client param, one match",

			groupParams:  ParamValues{"tag-a", "tag-2"},
			clientParams: []string{"tag-1", "tag-2", "tag-3"},

			wantRes: true,
		},
		{
			name: "plural client param, no match",

			groupParams:  ParamValues{"tag-a", "tag-b", "tag-c"},
			clientParams: []string{"tag-1", "tag-2", "tag-3"},

			wantRes: false,
		},
		{
			name: "plural client param, wildcard",

			groupParams:  ParamValues{"192.168.178.*", "10.10.10.*"},
			clientParams: []string{"192.168.178.2", "127.0.0.1"},

			wantRes: true,
		},
		{
			name: "plural client param, few wildcards",

			groupParams:  ParamValues{"tag-1*", "tag-2*", "tag-c"},
			clientParams: []string{"tag-1", "tag-22", "tag-3"},

			wantRes: true,
		},
		{
			name: "plural client param, no client params",

			groupParams:  ParamValues{"tag-1*", "tag-2*", "tag-c"},
			clientParams: []string{},

			wantRes: true,
		},
		{
			name: "plural client param, no group params",

			groupParams:  ParamValues{},
			clientParams: []string{"tag-1", "tag-2", "tag-3"},

			wantRes: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			gotRes := tc.groupParams.MatchesOneOf(tc.clientParams...)

			// then
			assert.Equal(t, tc.wantRes, gotRes)
		})
	}
}
