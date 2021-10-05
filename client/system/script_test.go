package system

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateScriptDir(t *testing.T) {
	testCases := []struct {
		name             string
		dirToGive        string
		dirModeToGive    os.FileMode
		shouldCreateDir  bool
		shouldCreateFile bool
		errToExpect      string
	}{
		{
			name:            "directory not exists",
			dirToGive:       "non_existing_dir",
			shouldCreateDir: false,
			errToExpect:     "script directory non_existing_dir does not exist",
		},
		{
			name:            "empty dir",
			shouldCreateDir: false,
			errToExpect:     "script directory cannot be empty",
		},
		{
			name:            "empty dir with spaces",
			dirToGive:       "     ",
			shouldCreateDir: false,
			errToExpect:     "script directory cannot be empty",
		},
		{
			name:            "working_dir",
			dirToGive:       "working_dir",
			dirModeToGive:   DefaultDirMode,
			shouldCreateDir: true,
		},
		{
			name:            "working dir with spaces",
			dirToGive:       "  working_dir_space",
			dirModeToGive:   DefaultDirMode,
			shouldCreateDir: true,
		},
		{
			name:             "file as dir name",
			dirToGive:        "some_file",
			shouldCreateFile: true,
			errToExpect:      "script directory some_file is not a valid directory",
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.shouldCreateFile {
				emptyFile, err := os.Create(tc.dirToGive)
				require.NoError(t, err)

				err = emptyFile.Close()
				require.NoError(t, err)
			}

			if testCase.shouldCreateDir {
				err := os.MkdirAll(tc.dirToGive, tc.dirModeToGive)
				require.NoError(t, err)
			}

			err := ValidateScriptDir(tc.dirToGive)
			if tc.errToExpect != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errToExpect)
			} else {
				require.NoError(t, err)
			}
		})
	}

	for _, testCase := range testCases {
		if !testCase.shouldCreateDir && !testCase.shouldCreateFile {
			continue
		}

		err := os.Remove(testCase.dirToGive)
		if err != nil {
			fmt.Println(err)
		}
	}
}
