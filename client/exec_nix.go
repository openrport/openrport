//+build !windows

package chclient

import (
	"context"
	"os/exec"
)

func (e *CmdExecutorImpl) New(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd {
	return e.newCmd(ctx, execCtx)
}
