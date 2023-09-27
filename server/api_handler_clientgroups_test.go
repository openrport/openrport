package chserver

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openrport/openrport/server/cgroups"
)

func TestValidateInputClientGroup(t *testing.T) {
	testCases := []struct {
		name    string
		groupID string
		wantErr error
	}{
		{
			name:    "empty group ID",
			groupID: "",
			wantErr: errors.New("group ID cannot be empty"),
		},
		{
			name:    "group ID only with whitespaces",
			groupID: " ",
			wantErr: errors.New("group ID cannot be empty"),
		},
		{
			name:    "group ID with invalid char '?'",
			groupID: "?",
			wantErr: errors.New(`invalid group ID "?": can contain only "A-Za-z0-9_-*"`),
		},
		{
			name:    "group ID with invalid char '.'",
			groupID: "2.1",
			wantErr: errors.New(`invalid group ID "2.1": can contain only "A-Za-z0-9_-*"`),
		},
		{
			name:    "group ID with extra whitespaces",
			groupID: " id ",
			wantErr: errors.New(`invalid group ID " id ": can contain only "A-Za-z0-9_-*"`),
		},
		{
			name:    "group ID with invalid char '/'",
			groupID: "2/1",
			wantErr: errors.New(`invalid group ID "2/1": can contain only "A-Za-z0-9_-*"`),
		},
		{
			name:    "valid group ID with all available chars",
			groupID: "*abc-XYZ_09_ABC-xyz*",
			wantErr: nil,
		},
		{
			name:    "valid group ID with one char",
			groupID: "a",
			wantErr: nil,
		},
		{
			name:    "valid group ID with one char '*'",
			groupID: "*",
			wantErr: nil,
		},
		{
			name:    "valid group ID with max number of chars",
			groupID: "012345678901234567890123456789",
			wantErr: nil,
		},
		{
			name:    "invalid group ID with too many chars",
			groupID: "0123456789012345678901234567890",
			wantErr: errors.New("invalid group ID: max length 30, got 31"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			gotErr := validateInputClientGroup(cgroups.ClientGroup{ID: tc.groupID})

			// then
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}

func TestValidateInputClientGroupParamsTag(t *testing.T) {
	testCases := []struct {
		name     string
		jsonData *json.RawMessage
		wantRes  bool
		wantErr  error
	}{
		{
			name:     "Tags json parsing ok 1",
			jsonData: jsonData(` ["Linux", "Datacenter 3"] `),
			wantErr:  nil,
		},
		{
			name:     "Tags json parsing ok 2",
			jsonData: jsonData(` { "AND": [ "Linux", "Datacenter 3" ] } `),
			wantErr:  nil,
		},
		{
			name:     "Tags json parsing ok 2",
			jsonData: jsonData(` { "amd": [ "Linux", "Datacenter 3" ] } `),
			wantErr:  errors.New("error, only and/or is allowed for tags group definitions"),
		},
		{
			name:     "Tags json parsing ok 2",
			jsonData: jsonData(` { "": [ "Linux", "Datacenter 3" ] } `),
			wantErr:  errors.New("error, only and/or is allowed for tags group definitions"),
		},
		{
			name:     "Tags json parsing error 1",
			jsonData: jsonData(` [Linux", "Datacenter 3"] `),
			wantErr:  errors.New("error parsing tags group definitions"),
		},
		{
			name:     "Tags json parsing error 2",
			jsonData: jsonData(` { "and": ["T*", "Datacenter 2, "Datacenter 5"] } `),
			wantErr:  errors.New("error parsing tags group definitions"),
		},
		{
			name:     "both and and or are present",
			jsonData: jsonData(` { [ { "and": [ "Linux", "Datacenter 3" ] }, { "or": [ "Linux", "Datacenter 3" ] }    ] } `),
			wantErr:  errors.New("error parsing tags group definitions"),
		},
		{
			name:     "no map keys",
			jsonData: jsonData(` `),
			wantErr:  errors.New("error parsing tags group definitions"),
		},
		{
			name:     "no map keys",
			jsonData: jsonData(` { } `),
			wantErr:  errors.New("error parsing tags group definitions"),
		},
		{
			name:     "no map keys 2 ",
			jsonData: jsonData(` { [] } `),
			wantErr:  errors.New("error parsing tags group definitions"),
		},
		{
			name:     "no map keys 3 ",
			jsonData: jsonData(` [] `),
			wantErr:  errors.New("error parsing tags group definitions"),
		},
		{
			name:     "no map keys 4 ",
			jsonData: jsonData(` { "and" : [] } `),
			wantErr:  errors.New("error parsing tags group definitions"),
		},
		{
			name:     "no tags",
			jsonData: nil,
			wantErr:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var group cgroups.ClientGroup
			group.ID = "testg1"
			group.Params = &cgroups.ClientParams{
				Tag: tc.jsonData,
			}

			gotErr := validateInputClientGroup(group)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}

func jsonData(data string) *json.RawMessage {
	bytes := []byte(data)
	return (*json.RawMessage)(&bytes)
}
