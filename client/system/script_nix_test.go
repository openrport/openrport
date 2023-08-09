//go:build !windows
// +build !windows

package system

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateScriptDirNix(t *testing.T) {
	tmp := t.TempDir()
	testCases := []struct {
		name             string
		dirToGive        string
		dirModeToGive    os.FileMode
		shouldCreateFile bool
		errToExpect      string
	}{
		{
			name:          "wrong_dir_mode",
			dirToGive:     tmp + "/wrong_dir_mode",
			dirModeToGive: os.FileMode(0755),
			errToExpect:   "wrong_dir_mode must be read-writable only by",
		},
		{
			name:          "not_writable_dir",
			dirToGive:     tmp + "/not_writable_dir",
			dirModeToGive: os.FileMode(0444),
			errToExpect:   "not_writable_dir is not writable",
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := os.MkdirAll(tc.dirToGive, tc.dirModeToGive)
			require.NoError(t, err)

			err = ValidateScriptDir(tc.dirToGive)
			if tc.errToExpect != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errToExpect)
			} else {
				require.NoError(t, err)
			}
		})
	}

	for _, testCase := range testCases {
		err := os.Remove(testCase.dirToGive)
		if err != nil {
			fmt.Println(err)
		}
	}
}
