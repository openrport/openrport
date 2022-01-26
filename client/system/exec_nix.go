//go:build !windows
// +build !windows

package system

import (
	"context"
	"os/exec"
	"strings"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func (e *CmdExecutorImpl) New(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd {
	var args []string
	if execCtx.IsSudo {
		args = append(args, "sudo", "-n")
	}

	var interpreter string
	if execCtx.HasShebang {
		interpreter = ""
	} else {
		interpreter = execCtx.Interpreter.Get()
	}

	if interpreter != "" {
		args = append(args, interpreter)
		if interpreter != chshare.Tacoscript {
			args = append(args, "-c")
		}
	}

	cmdStr := execCtx.Command
	if strings.Contains(cmdStr, " ") && interpreter != chshare.Tacoscript {
		cmdStr = strings.ReplaceAll(cmdStr, " ", "\\ ")
	}

	args = append(args, cmdStr)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = execCtx.WorkingDir

	return cmd
}
