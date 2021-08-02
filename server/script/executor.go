package script

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/clients"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/random"
)

const DefaultScriptFileMode = os.FileMode(0744)

type Executor struct {
	logger *chshare.Logger
}

func NewExecutor(logger *chshare.Logger) *Executor {
	return &Executor{
		logger: logger,
	}
}

func (e *Executor) CreateScriptOnClient(scriptInput *api.ExecuteInput, cl *clients.Client) (scriptPath string, err error) {
	fileName := e.createClientScriptPath(cl.OSKernel, scriptInput.Shell)
	fileInput := &models.File{
		Name:      fileName,
		Content:   []byte(scriptInput.Script),
		CreateDir: true,
		Mode:      DefaultScriptFileMode,
	}

	sshResp := &comm.CreateFileResponse{}
	err = comm.SendRequestAndGetResponse(cl.Connection, comm.RequestTypeCreateFile, fileInput, sshResp)
	if err != nil {
		return scriptPath, err
	}

	hasher := sha256.New()
	_, err = io.Copy(hasher, bytes.NewBufferString(scriptInput.Script))
	if err != nil {
		return scriptPath, err
	}

	e.logger.Debugf("script successfully copied to the client: %+v", sshResp)

	expectedHash := hex.EncodeToString(hasher.Sum(nil))
	if expectedHash != sshResp.Sha256Hash {
		return scriptPath, fmt.Errorf("mismatch of request %s and response %s script hashes", expectedHash, sshResp.Sha256Hash)
	}

	return sshResp.FilePath, nil
}

func (e *Executor) createClientScriptPath(os, shell string) string {
	scriptName := random.UUID4()
	if os == "windows" {
		if shell == "powershell" {
			return scriptName + ".ps1"
		}
		return scriptName + ".bat"
	}

	return scriptName + ".sh"
}
