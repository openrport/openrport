package chclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/random"
)

const DefaultFileMode = os.FileMode(0540)
const DefaultDirMode = os.FileMode(0700)

func (c *Client) HandleCreateFileRequest(ctx context.Context, reqPayload []byte) (*comm.CreateFileResponse, error) {
	if !c.config.RemoteScripts.Enabled {
		return nil, errors.New("remote scripts are disabled")
	}

	fileInput := models.ScriptFile{}

	fileContentBuf := bytes.NewBuffer(reqPayload)
	dec := json.NewDecoder(fileContentBuf)
	dec.DisallowUnknownFields()
	err := dec.Decode(&fileInput)
	if err != nil {
		return nil, err
	}

	filePath, fileHash, err := CreateScriptFile(c.config.GetScriptsDir(), fileInput.Interpreter, fileInput.Content)
	if err != nil {
		return nil, err
	}

	return &comm.CreateFileResponse{
		FilePath:   filePath,
		Sha256Hash: fileHash,
		CreatedAt:  time.Now(),
	}, nil
}

func CreateScriptFile(scriptDir, interpreter string, scriptContent []byte) (filePath, hash string, err error) {
	err = ValidateScriptDir(scriptDir)
	if err != nil {
		return "", "", err
	}

	scriptFileName, err := createScriptFileName(interpreter)
	if err != nil {
		return "", "", err
	}

	scriptFilePath := filepath.Join(scriptDir, scriptFileName)

	err = ioutil.WriteFile(scriptFilePath, scriptContent, DefaultFileMode)
	if err != nil {
		return "", "", err
	}

	hasher := sha256.New()
	_, err = io.Copy(hasher, bytes.NewBuffer(scriptContent))
	if err != nil {
		return "", "", err
	}

	return scriptFilePath, hex.EncodeToString(hasher.Sum(nil)), nil
}

func ValidateScriptDir(scriptDir string) error {
	if strings.TrimSpace(scriptDir) == "" {
		return errors.New("script directory cannot be empty")
	}

	dirStat, err := os.Stat(scriptDir)

	if os.IsNotExist(err) {
		return fmt.Errorf("script directory %s does not exist", scriptDir)
	}
	if err != nil {
		return err
	}
	if !dirStat.IsDir() {
		return fmt.Errorf("script directory %s is not a valid directory", scriptDir)
	}

	err = ValidateScriptDirOS(dirStat, scriptDir)
	if err != nil {
		return err
	}

	return nil
}

func createScriptFileName(interpreter string) (string, error) {
	scriptName, err := random.UUID4()
	if err != nil {
		return "", err
	}

	extension := getExtension(interpreter)

	return scriptName + extension, nil
}

func getExtension(interpreter string) string {
	if interpreter == chshare.Taco {
		return ".yml"
	}

	return GetScriptExtensionOS(interpreter)
}
