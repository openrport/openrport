package script

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/cloudradar-monitoring/rport/server/clients"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/random"
)

type Executor struct {
	logger *chshare.Logger
}

func NewExecutor(logger *chshare.Logger) *Executor {
	return &Executor{
		logger: logger,
	}
}

//todo implement
func (e *Executor) createScriptCommand(scriptPath string, cl *clients.Client, isSudo bool, cwd string) string {
	return ""
}

//todo implement
func (e *Executor) createShell(cl *clients.Client, isPowershell bool) string {
	return ""
}

func (e *Executor) RunScriptOnClient(curUser string, cl *clients.Client, scriptBody []byte, isSudo, isPowershell bool, cwd string) error {
	scriptPath, err := e.createScriptOnClient(cl, isPowershell, scriptBody)
	if err != nil {
		return err
	}

	command := e.createScriptCommand(scriptPath, cl, isSudo, cwd)

	curJob := models.Job{
		JobSummary: models.JobSummary{
			JID:        random.UUID4(),
			FinishedAt: nil,
		},
		ClientID:   cl.ID,
		ClientName: cl.Name,
		Command:    command,
		Shell:      e.createShell(cl, isPowershell),
		CreatedBy:  curUser,
		Result:     nil,
		Cwd:        cwd,
	}
	sshResp := &comm.RunCmdResponse{}
	err = comm.SendRequestAndGetResponse(cl.Connection, comm.RequestTypeRunCmd, curJob, sshResp)
	if err != nil {
		return err
	}

	return nil
}

func (e *Executor) createScriptOnClient(cl *clients.Client, isPowershell bool, scriptBody []byte) (scriptPath string, err error) {
	fileInput := &models.File{
		Name:      e.createClientScriptPath(cl, isPowershell),
		Content:   scriptBody,
		CreateDir: true,
		Mode:      0744,
	}

	sshResp := &comm.CreateFileResponse{}
	err = comm.SendRequestAndGetResponse(cl.Connection, comm.RequestTypeCreateFile, fileInput, sshResp)
	if err != nil {
		return "", err
	}

	hasher := sha256.New()
	_, err = io.Copy(hasher, bytes.NewBuffer(scriptBody))
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

func (e *Executor) isWindowsClient(cl *clients.Client) bool {
	return cl.OSKernel == "windows"
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
