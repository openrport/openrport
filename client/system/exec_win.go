//go:build windows
// +build windows

package system

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	chshare "github.com/openrport/openrport/share"
)

func (e *CmdExecutorImpl) New(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd {
	interpreterPath := execCtx.Interpreter.Get()
	e.Debugf("resolved interpreter %s for input %s", interpreterPath, execCtx.Interpreter.InterpreterNameFromInput)

	if execCtx.Interpreter.Matches(chshare.CmdShell, true) {
		return buildCmdInterpreterCmd(ctx, execCtx, interpreterPath)
	} else if execCtx.Interpreter.Matches(chshare.PowerShell, false) {
		return buildPowershellCmd(ctx, execCtx, interpreterPath)
	} else {
		return buildDefaultCmd(ctx, execCtx, interpreterPath)
	}
}

func buildCmdInterpreterCmd(ctx context.Context, execCtx *CmdExecutorContext, interpreterPath string) *exec.Cmd {
	// workaround for the issue with escaping args on windows for cmd interpreter https://github.com/golang/go/issues/1849
	var cmd *exec.Cmd
	if interpreterPath != "" {
		cmd = exec.CommandContext(ctx, interpreterPath)
	} else {
		cmd = exec.CommandContext(ctx, chshare.CmdShell+".exe")
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{}

	cmdStr := execCtx.Command
	if strings.Contains(cmdStr, " ") {
		cmdStr = `"` + strings.Trim(cmdStr, `"`) + `"`
	}

	cmd.SysProcAttr.CmdLine = fmt.Sprintf("/c %s", cmdStr)
	cmd.Dir = execCtx.WorkingDir

	return cmd
}

func buildPowershellCmd(ctx context.Context, execCtx *CmdExecutorContext, interpreterPath string) *exec.Cmd {
	args := []string{
		"-Noninteractive", // Don't present an interactive prompt to the user.
		"-executionpolicy",
		"bypass",
	}

	args = append(args, "-File")

	var cmd *exec.Cmd
	if interpreterPath != "" {
		args = append(args, execCtx.Command)
		cmd = exec.CommandContext(ctx, interpreterPath, args...)
	} else {
		cmd = exec.CommandContext(ctx, chshare.PowerShell+".exe", args...)
	}
	cmd.Dir = execCtx.WorkingDir

	return cmd
}

func buildDefaultCmd(ctx context.Context, execCtx *CmdExecutorContext, interpreterPath string) *exec.Cmd {
	var cmd *exec.Cmd
	if interpreterPath != "" {
		cmd = exec.CommandContext(ctx, interpreterPath, execCtx.Command)
	} else {
		cmd = exec.CommandContext(ctx, execCtx.Command)
	}
	cmd.Dir = execCtx.WorkingDir

	return cmd
}
