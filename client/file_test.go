package chclient

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/models"
)

func TestHandleCreateFileRequest(t *testing.T) {
	inputFile := &models.File{
		Name:      "some.file.txt",
		Content:   []byte("1234"),
		CreateDir: false,
		Mode:      os.FileMode(0755),
	}

	inputFileBytes, err := json.Marshal(inputFile)
	require.NoError(t, err)

	config := &Config{}

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

	assert.True(t, strings.HasSuffix(resp.FilePath, "some.file.txt"))
	assert.Equal(t, "03ac674216f3e15c761ee1a5e255f067953623c8b388b4459e13f978d7c846f4", resp.Sha256Hash)
	assert.True(t, resp.CreatedAt.Unix() > 0)

	assert.FileExists(t, resp.FilePath)
	actualFileContent, err := ioutil.ReadFile(resp.FilePath)
	require.NoError(t, err)

	assert.Equal(t, "1234", string(actualFileContent))
}
