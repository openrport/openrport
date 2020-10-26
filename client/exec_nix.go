//+build !windows

package chclient

import (
	"context"
	"os/exec"
)

func (e *CmdExecutorImpl) New(ctx context.Context, shell, command string) *exec.Cmd {
	return e.newCmd(ctx, shell, command)
}
