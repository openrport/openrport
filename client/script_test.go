package chclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/models"

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

func TestHandleCreateFileRequest(t *testing.T) {
	inputFile := &models.ScriptFile{
		Content:     []byte("1234"),
		Interpreter: chshare.Taco,
	}

	inputFileBytes, err := json.Marshal(inputFile)
	require.NoError(t, err)

	scriptDirToCheck := filepath.Join(os.TempDir(), "TestHandleCreateFileRequest")
	config := &Config{
		Client: ClientConfig{
			DataDir: scriptDirToCheck,
		},
		RemoteScripts: ScriptsConfig{
			Enabled: true,
		},
	}

	err = os.MkdirAll(config.GetScriptsDir(), DefaultDirMode)
	require.NoError(t, err)
	defer os.Remove(scriptDirToCheck)

	client := NewClient(config)

	resp, err := client.HandleCreateFileRequest(context.Background(), inputFileBytes)
	require.NoError(t, err)

	logger := &chshare.Logger{}
	defer func() {
		err := os.Remove(resp.FilePath)
		if err != nil {
			logger.Errorf("failed to delete file: %v", err)
		}
	}()

	assert.True(t, strings.HasSuffix(resp.FilePath, ".yml"))
	assert.Equal(t, "03ac674216f3e15c761ee1a5e255f067953623c8b388b4459e13f978d7c846f4", resp.Sha256Hash)
	assert.True(t, resp.CreatedAt.Unix() > 0)

	assert.FileExists(t, resp.FilePath)
	actualFileContent, err := ioutil.ReadFile(resp.FilePath)
	require.NoError(t, err)

	assert.Equal(t, "1234", string(actualFileContent))
}

func TestCreateFileWhenScriptsDisabled(t *testing.T) {
	config := &Config{
		RemoteScripts: ScriptsConfig{
			Enabled: false,
		},
	}

	client := NewClient(config)

	_, err := client.HandleCreateFileRequest(context.Background(), []byte{})
	require.EqualError(t, err, "remote scripts are disabled")
}
