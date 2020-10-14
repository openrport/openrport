package chclient

import (
	"os/exec"
)

type CmdExecutor interface {
	Start(cmd *exec.Cmd) error
	Wait(cmd *exec.Cmd) error
}

type CmdExecutorImpl struct {
}

func NewCmdExecutor() *CmdExecutorImpl {
	return &CmdExecutorImpl{}
}

func (e *CmdExecutorImpl) Start(cmd *exec.Cmd) error {
	return cmd.Start()
}

func (e *CmdExecutorImpl) Wait(cmd *exec.Cmd) error {
	return cmd.Wait()
}
