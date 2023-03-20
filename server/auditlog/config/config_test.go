package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		Name   string
		Config Config
		Want   error
	}{
		{
			Name: "ok",
			Config: Config{
				Enable:   true,
				Rotation: RotationMonthly,
			},
			Want: nil,
		}, {
			Name: "disabled",
			Config: Config{
				Enable:   false,
				Rotation: "not-checked",
			},
			Want: nil,
		}, {
			Name: "invalid rotation",
			Config: Config{
				Enable:   true,
				Rotation: "invalid",
			},
			Want: errors.New(`invalid api.audit_log_rotation: "invalid"`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			err := tc.Config.Validate()

			if tc.Want == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tc.Want.Error(), err.Error())
			}
		})
	}
}
