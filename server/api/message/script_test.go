package message_test

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/server/api/message"
)

func TestScriptServiceValidateReceiver(t *testing.T) {
	testCases := []struct {
		Name          string
		Validation    message.ValidationType
		Regex         *regexp.Regexp
		Receiver      string
		ExpectedError error
	}{
		{
			Name:          "no validation",
			Validation:    message.ValidationNone,
			Receiver:      "whatever",
			ExpectedError: nil,
		},
		{
			Name:          "email validation, valid",
			Validation:    message.ValidationEmail,
			Receiver:      "whatever@example.com",
			ExpectedError: nil,
		},
		{
			Name:          "email validation, invalid",
			Validation:    message.ValidationEmail,
			Receiver:      "not-an-email",
			ExpectedError: errors.New("invalid email format"),
		},
		{
			Name:          "regex validation, valid",
			Validation:    message.ValidationRegex,
			Regex:         regexp.MustCompile("[a-d]{4}"),
			Receiver:      "abcd",
			ExpectedError: nil,
		},
		{
			Name:          "regex validation, invalid",
			Validation:    message.ValidationRegex,
			Regex:         regexp.MustCompile("[a-d]{4}"),
			Receiver:      "invalid",
			ExpectedError: errors.New(`does not match "[a-d]{4}"`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			ss := message.NewScriptService("", tc.Validation, tc.Regex)

			err := ss.ValidateReceiver(context.Background(), tc.Receiver)

			assert.Equal(t, tc.ExpectedError, err)
		})
	}
}

func TestDataToEnv(t *testing.T) {
	ss := message.NewScriptService("", message.ValidationNone, nil)

	data := message.Data{
		SendTo: "whatever@example.com",
		Token:  "abcd12",
		TTL:    5 * time.Minute,
	}
	env := ss.DataToEnv(data)

	expected := []string{
		"2FA_TOKEN=abcd12",
		"2FA_SENDTO=whatever@example.com",
		"2FA_TOKEN_TTL=300",
	}
	assert.ElementsMatch(t, expected, env)
}
