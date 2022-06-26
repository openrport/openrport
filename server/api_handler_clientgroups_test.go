package chserver

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/server/cgroups"
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
