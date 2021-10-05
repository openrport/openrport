//+build !windows

package system

import (
	"context"
	"os/exec"
	"strings"
)

func (e *CmdExecutorImpl) New(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd {
	var args []string
	if execCtx.IsSudo {
		args = append(args, "sudo", "-n")
	}

	interpreter := execCtx.Interpreter
	if interpreter != "" {
		args = append(args, interpreter, "-c")
	}

	commandStr := execCtx.Command
	if execCtx.IsScript {
		commandStr = strings.ReplaceAll(commandStr, " ", "\\ ")
	}

	args = append(args, commandStr)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = execCtx.WorkingDir

	return cmd
}
