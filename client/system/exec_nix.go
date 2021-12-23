//+build !windows

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

	interpreter, err := e.getInterpreter(execCtx)
	if err != nil {
		e.Errorf(err.Error())
	}

	if interpreter != "" {
		args = append(args, interpreter)
		if execCtx.Interpreter != chshare.Tacoscript {
			args = append(args, "-c")
		}
	}

	cmdStr := execCtx.Command
	if strings.Contains(cmdStr, " ") && execCtx.Interpreter != chshare.Tacoscript {
		cmdStr = strings.ReplaceAll(cmdStr, " ", "\\ ")
	}

	args = append(args, cmdStr)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = execCtx.WorkingDir

	return cmd
}

func (e *CmdExecutorImpl) getInterpreter(execCtx *CmdExecutorContext) (string, error) {
	if execCtx.InterpreterAliases != nil && execCtx.Interpreter != "" {
		if mappedInterpreter, ok := execCtx.InterpreterAliases[execCtx.Interpreter]; ok {
			return mappedInterpreter, nil
		}
	}
	if execCtx.HasShebang {
		return "", nil
	}

	if execCtx.Interpreter == "" {
		return chshare.UnixShell, nil
	}

	return execCtx.Interpreter, nil
}
