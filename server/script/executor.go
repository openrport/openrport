package script

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/clients"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/random"
)

const DefaultScriptFileMode = os.FileMode(0744)

type ExecutionInput struct {
	Client       *clients.Client
	ScriptBody   []byte
	IsSudo       bool
	IsPowershell bool
	Cwd          string
	Timeout      time.Duration
}

type Executor struct {
	logger *chshare.Logger
}

func NewExecutor(logger *chshare.Logger) *Executor {
	return &Executor{
		logger: logger,
	}
}

func (e *Executor) ConvertScriptInputToCmdInput(ei *ExecutionInput, scriptPath string) *api.ExecuteCommandInput {
	return &api.ExecuteCommandInput{
		Command:    scriptPath,
		Shell:      e.createShell(ei.Client, ei.IsPowershell),
		Cwd:        ei.Cwd,
		IsSudo:     ei.IsSudo,
		TimeoutSec: int(ei.Timeout.Seconds()),
		ClientID:   ei.Client.ID,
		IsScript:   true,
	}
}

func (e *Executor) CreateScriptOnClient(scriptInput *ExecutionInput) (scriptPath string, err error) {
	fileInput := &models.File{
		Name:      e.createClientScriptPath(scriptInput.Client, scriptInput.IsPowershell),
		Content:   scriptInput.ScriptBody,
		CreateDir: true,
		Mode:      DefaultScriptFileMode,
	}

	sshResp := &comm.CreateFileResponse{}
	err = comm.SendRequestAndGetResponse(scriptInput.Client.Connection, comm.RequestTypeCreateFile, fileInput, sshResp)
	if err != nil {
		return "", err
	}

	hasher := sha256.New()
	_, err = io.Copy(hasher, bytes.NewBuffer(scriptInput.ScriptBody))
	if err != nil {
		return "", err
	}

	e.logger.Debugf("script successfully copied to the client: %+v", sshResp)

	expectedHash := hex.EncodeToString(hasher.Sum(nil))
	if expectedHash != sshResp.Sha256Hash {
		return "", fmt.Errorf("mismatch of request %s and response %s script hashes", expectedHash, sshResp.Sha256Hash)
	}

	return sshResp.FilePath, nil
}

func (e *Executor) createClientScriptPath(cl *clients.Client, isPowershell bool) string {
	scriptName := random.UUID4()
	if e.isWindowsClient(cl) {
		if isPowershell {
			return scriptName + ".ps1"
		}
		return scriptName + ".bat"
	}

	return scriptName + ".sh"
}

func (e *Executor) isWindowsClient(cl *clients.Client) bool {
	return cl.OSKernel == "windows"
}

func (e *Executor) createShell(cl *clients.Client, isPowershell bool) string {
	if e.isWindowsClient(cl) {
		if isPowershell {
			return "powershell"
		}

		return "cmd"
	}

	return ""
}
