package system

import (
	"context"
	"os/exec"

	chshare "github.com/cloudradar-monitoring/rport/share/logger"
)

type CmdExecutorContext struct {
	Interpreter string
	Command     string
	WorkingDir  string
	IsSudo      bool
	IsScript    bool
}

type CmdExecutor interface {
	New(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd
	Start(cmd *exec.Cmd) error
	Wait(cmd *exec.Cmd) error
}

type CmdExecutorImpl struct {
	*chshare.Logger
}

func NewCmdExecutor(l *chshare.Logger) *CmdExecutorImpl {
	return &CmdExecutorImpl{
		Logger: l,
	}
}

func (e *CmdExecutorImpl) Start(cmd *exec.Cmd) error {
	return cmd.Start()
}

func (e *CmdExecutorImpl) Wait(cmd *exec.Cmd) error {
	return cmd.Wait()
}
