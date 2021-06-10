//+build !windows

package chclient

import (
	"context"
	"os/exec"
)

func (e *CmdExecutorImpl) New(ctx context.Context, shell, command, cwd string, isSudo bool) *exec.Cmd {
	return e.newCmd(ctx, shell, command, cwd, isSudo)
}
