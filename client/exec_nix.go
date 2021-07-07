//+build !windows

package chclient

import (
	"context"
	"os/exec"
)

func (e *CmdExecutorImpl) New(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd {
	var args []string
	prefix := execCtx.Shell
	if execCtx.IsSudo {
		prefix = "sudo"
		args = append(args, "-n", execCtx.Shell)
	}

	args = append(args, "-c", execCtx.Command)

	cmd := exec.CommandContext(ctx, prefix, args...)
	cmd.Dir = execCtx.WorkingDir

	return cmd
}
