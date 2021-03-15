package cgroups

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchesOneOf(t *testing.T) {
	testCases := []struct {
		name string

		groupParams  *ParamValues
		clientParams []string

		wantRes bool
	}{
		{
			name: "exact match",

			groupParams:  &ParamValues{"id-1"},
			clientParams: []string{"id-1"},

			wantRes: true,
		},
		{
			name: "exact match, client param in upper case",

			groupParams:  &ParamValues{"id-1"},
			clientParams: []string{"ID-1"},

			wantRes: true,
		},
		{
			name: "exact match, group param in upper case",

			groupParams:  &ParamValues{"Name-1"},
			clientParams: []string{"name-1"},

			wantRes: true,
		},
		{
			name: "exact match but twice",

			groupParams:  &ParamValues{"123"},
			clientParams: []string{"123123"},

			wantRes: false,
		},
		{
			name: "start and end shares char",

			groupParams:  &ParamValues{"123*31"},
			clientParams: []string{"1231"},

			wantRes: false,
		},
		{
			name: "extra char at start",

			groupParams:  &ParamValues{"1123*31"},
			clientParams: []string{"123431"},

			wantRes: false,
		},
		{
			name: "extra char at end",

			groupParams:  &ParamValues{"113*21"},
			clientParams: []string{"1134121211"},

			wantRes: false,
		},
		{
			name: "group params contains exact client param",

			groupParams:  &ParamValues{"id-1", "id-2", "id-3"},
			clientParams: []string{"id-2"},

			wantRes: true,
		},
		{
			name: "wildcard group param, min client param",

			groupParams:  &ParamValues{"id*"},
			clientParams: []string{"id"},

			wantRes: true,
		},
		{
			name: "wildcard group param, short client param",

			groupParams:  &ParamValues{"id-*"},
			clientParams: []string{"id-2"},

			wantRes: true,
		},
		{
			name: "wildcard group param in upper case, client in lower",

			groupParams:  &ParamValues{"ID-*"},
			clientParams: []string{"id-2"},

			wantRes: true,
		},
		{
			name: "wildcard group param in lower case, client in upper",

			groupParams:  &ParamValues{"id-*"},
			clientParams: []string{"Id-2"},

			wantRes: true,
		},
		{
			name: "wildcard group param, long client param",

			groupParams:  &ParamValues{"id-*"},
			clientParams: []string{"id-12345678"},

			wantRes: true,
		},
		{
			name: "only wildcard",

			groupParams:  &ParamValues{"*"},
			clientParams: []string{"id-12345678"},

			wantRes: true,
		},
		{
			name: "wildcards at start end",

			groupParams:  &ParamValues{"*-inside-*"},
			clientParams: []string{"id-123-inside-45678"},

			wantRes: true,
		},
		{
			name: "wildcards, client value the same but without wildcards",

			groupParams:  &ParamValues{"*-*in***side**-*"},
			clientParams: []string{"-inside-"},

			wantRes: true,
		},
		{
			name: "few wildcards in the middle",

			groupParams:  &ParamValues{"prefix*1**2*35"},
			clientParams: []string{"prefix-id-1234567835"},

			wantRes: true,
		},
		{
			name: "few wildcards in the middle, no match",

			groupParams:  &ParamValues{"prefix*1**2*35"},
			clientParams: []string{"prefix-id-1234567836"},

			wantRes: false,
		},
		{
			name: "few wildcards in all positions",

			groupParams:  &ParamValues{"*p**re***fix*1*2*3*"},
			clientParams: []string{"preeeeeefix-id-111233456783"},

			wantRes: true,
		},
		{
			name: "group params does no contain client param",

			groupParams:  &ParamValues{"id-1", "id-2", "id-3"},
			clientParams: []string{"id-4"},

			wantRes: false,
		},
		{
			name: "group params including wildcard does not contain client param",

			groupParams:  &ParamValues{"id-11", "id-22", "id-33", "id-1*", "id-2*"},
			clientParams: []string{"id-3"},

			wantRes: false,
		},
		{
			name: "empty client param, nonempty group param",

			groupParams:  &ParamValues{"id-1"},
			clientParams: []string{""},

			wantRes: false,
		},
		{
			name: "no client param, nonempty group param",

			groupParams:  &ParamValues{"tag-1"},
			clientParams: []string{},

			wantRes: false,
		},
		{
			name: "no client param, no group param",

			groupParams:  &ParamValues{},
			clientParams: []string{},

			wantRes: true,
		},
		{
			name: "no client param, empty group param",

			groupParams:  &ParamValues{""},
			clientParams: []string{},

			wantRes: false,
		},
		{
			name: "empty client param, empty group param",

			groupParams:  &ParamValues{""},
			clientParams: []string{""},

			wantRes: true,
		},
		{
			name: "nonempty client param, empty group param",

			groupParams:  &ParamValues{""},
			clientParams: []string{"id-1"},

			wantRes: false,
		},
		{
			name: "nonempty client param, empty group params",

			groupParams:  &ParamValues{"", ""},
			clientParams: []string{"id-1"},

			wantRes: false,
		},
		{
			name: "empty client param, empty group params",

			groupParams:  &ParamValues{"", ""},
			clientParams: []string{"tag-1", ""},

			wantRes: true,
		},
		{
			name: "no client param, empty group params",

			groupParams:  &ParamValues{"", ""},
			clientParams: []string{},

			wantRes: false,
		},
		{
			name: "empty client param, no group param",

			groupParams:  &ParamValues{},
			clientParams: []string{""},

			wantRes: false,
		},
		{
			name: "empty client param, unset group param",

			groupParams:  nil,
			clientParams: []string{""},

			wantRes: true,
		},
		{
			name: "nonempty client param, unset group param",

			groupParams:  nil,
			clientParams: []string{"lala"},

			wantRes: true,
		},
		{
			name: "nonempty client param, no group param",

			groupParams:  &ParamValues{},
			clientParams: []string{"id-1"},

			wantRes: false,
		},
		{
			name: "plural client param, one match",

			groupParams:  &ParamValues{"tag-a", "tag-2"},
			clientParams: []string{"tag-1", "tag-2", "tag-3"},

			wantRes: true,
		},
		{
			name: "plural client param, no match",

			groupParams:  &ParamValues{"tag-a", "tag-b", "tag-c"},
			clientParams: []string{"tag-1", "tag-2", "tag-3"},

			wantRes: false,
		},
		{
			name: "plural client param, wildcard",

			groupParams:  &ParamValues{"192.168.178.*", "10.10.10.*"},
			clientParams: []string{"192.168.178.2", "127.0.0.1"},

			wantRes: true,
		},
		{
			name: "plural client param, few wildcards",

			groupParams:  &ParamValues{"tag-1*", "tag-2*", "tag-c"},
			clientParams: []string{"tag-1", "tag-22", "tag-3"},

			wantRes: true,
		},
		{
			name: "no client param, plural group params",

			groupParams:  &ParamValues{"tag-1*", "tag-2*", "tag-c"},
			clientParams: []string{},

			wantRes: false,
		},
		{
			name: "plural client param, no group params",

			groupParams:  &ParamValues{},
			clientParams: []string{"tag-1", "tag-2", "tag-3"},

			wantRes: false,
		},
		{
			name: "plural client param, unset group params",

			groupParams:  nil,
			clientParams: []string{"tag-1", "tag-2", "tag-3"},

			wantRes: true,
		},
		{
			name: "client param with no values, unset group params",

			groupParams:  nil,
			clientParams: nil,

			wantRes: true,
		},
		{
			name: "client param with no values, no group params",

			groupParams:  &ParamValues{},
			clientParams: nil,

			wantRes: true,
		},
		{
			name: "client param with no values, group param with empty value",

			groupParams:  &ParamValues{""},
			clientParams: nil,

			wantRes: false,
		},
		{
			name: "client param with no values, group param with nonempty value",

			groupParams:  &ParamValues{"123"},
			clientParams: nil,

			wantRes: false,
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
