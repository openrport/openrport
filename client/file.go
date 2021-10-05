package chclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/cloudradar-monitoring/rport/client/system"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
)

func (c *Client) HandleCreateFileRequest(ctx context.Context, reqPayload []byte) (*comm.CreateFileResponse, error) {
	if !c.config.RemoteScripts.Enabled {
		return nil, errors.New("remote scripts are disabled")
	}

	scriptDirName := c.config.GetScriptsDir()
	err := system.ValidateScriptDir(scriptDirName)
	if err != nil {
		return nil, err
	}

	fileInput := models.File{}

	fileContentBuf := bytes.NewBuffer(reqPayload)
	dec := json.NewDecoder(fileContentBuf)
	dec.DisallowUnknownFields()
	err = dec.Decode(&fileInput)
	if err != nil {
		return nil, err
	}

	if fileInput.Name == "" {
		return nil, errors.New("empty file name provided")
	}

	if fileInput.Mode == 0 {
		fileInput.Mode = system.DefaultFileMode
	}

	scriptFileName := filepath.Base(fileInput.Name)

	fileInput.Name = filepath.Join(scriptDirName, scriptFileName)

	err = ioutil.WriteFile(fileInput.Name, fileInput.Content, fileInput.Mode)
	if err != nil {
		return nil, err
	}

	hasher := sha256.New()
	_, err = io.Copy(hasher, bytes.NewBuffer(fileInput.Content))
	if err != nil {
		return nil, err
	}

	return &comm.CreateFileResponse{
		FilePath:   fileInput.Name,
		Sha256Hash: hex.EncodeToString(hasher.Sum(nil)),
		CreatedAt:  time.Now(),
	}, nil
}
