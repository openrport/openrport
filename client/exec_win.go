//+build windows

package chclient

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
)

func (e *CmdExecutorImpl) New(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd {
	switch execCtx.Shell {
	case cmdShell:
		return buildCmdShellCmd(ctx, execCtx)
	case powerShell:
		return buildPowershellCmd(ctx, execCtx)
	default:
		return buildDefaultCmd(ctx, execCtx)
	}
}

func buildCmdShellCmd(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd {
	// workaround for the issue with escaping args on windows for cmd shell https://github.com/golang/go/issues/1849
	cmd := exec.CommandContext(ctx, execCtx.Shell)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.CmdLine = fmt.Sprintf("/c %s", execCtx.Command)
	cmd.Dir = execCtx.WorkingDir

	return cmd
}

func buildPowershellCmd(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd {
	args := []string{
		"-Noninteractive", // Don't present an interactive prompt to the user.
		"-executionpolicy",
		"bypass",
	}

	if execCtx.IsScript {
		args = append(args, "-File")
	} else {
		args = append(args, "-Command")
	}

	args = append(args, execCtx.Command)

	cmd := exec.CommandContext(ctx, execCtx.Shell, args...)
	cmd.Dir = execCtx.WorkingDir

	return cmd
}

func buildDefaultCmd(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd {
	cmd := exec.CommandContext(ctx, execCtx.Shell, execCtx.Command)
	cmd.Dir = execCtx.WorkingDir

	return cmd
}
