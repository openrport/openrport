//+build windows

package chclient

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func (e *CmdExecutorImpl) New(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd {
	shellPath := execCtx.Shell
	absShellPath, err := getShellAbsolutePath(execCtx.Shell)
	if err != nil {
		e.Errorf(err.Error())
	} else {
		shellPath = absShellPath
	}

	switch execCtx.Shell {
	case cmdShell:
		return buildCmdShellCmd(ctx, execCtx, shellPath)
	case powerShell:
		return buildPowershellCmd(ctx, execCtx, shellPath)
	default:
		return buildDefaultCmd(ctx, execCtx, shellPath)
	}
}

func buildCmdShellCmd(ctx context.Context, execCtx *CmdExecutorContext, shellPath string) *exec.Cmd {
	// workaround for the issue with escaping args on windows for cmd shell https://github.com/golang/go/issues/1849
	cmd := exec.CommandContext(ctx, shellPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.CmdLine = fmt.Sprintf("/c %s", execCtx.Command)
	cmd.Dir = execCtx.WorkingDir

	return cmd
}

func buildPowershellCmd(ctx context.Context, execCtx *CmdExecutorContext, shellPath string) *exec.Cmd {
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

	cmd := exec.CommandContext(ctx, shellPath, args...)
	cmd.Dir = execCtx.WorkingDir

	return cmd
}

func buildDefaultCmd(ctx context.Context, execCtx *CmdExecutorContext, shellPath string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, shellPath, execCtx.Command)
	cmd.Dir = execCtx.WorkingDir

	return cmd
}

func getShellAbsolutePath(shell string) (absShellPath string, err error) {
	if !strings.HasSuffix(shell, ".exe") {
		shell += ".exe"
	}

	path := os.Getenv("path")
	for _, dir := range filepath.SplitList(path) {
		absShellPath := filepath.Join(dir, shell)
		d, err := os.Stat(absShellPath)
		if err != nil || d.IsDir() {
			continue
		}

		return absShellPath, nil
	}

	return "", fmt.Errorf("failed to find %s at %%PATH%%: %s: %w", shell, path, os.ErrNotExist)
}
