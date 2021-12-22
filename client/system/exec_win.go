//+build windows

package system

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func (e *CmdExecutorImpl) New(ctx context.Context, execCtx *CmdExecutorContext) *exec.Cmd {
	interpreterPath, err := e.getInterpreter(execCtx)
	if err != nil {
		e.Errorf(err.Error())
	} else {
		e.Debugf("resolved absolute interpreter path %s for interpreter %s", interpreterPath, execCtx.Interpreter)
	}

	switch execCtx.Interpreter {
	case chshare.CmdShell:
		return buildCmdInterpreterCmd(ctx, execCtx, interpreterPath)
	case chshare.PowerShell:
		return buildPowershellCmd(ctx, execCtx, interpreterPath)
	default:
		return buildDefaultCmd(ctx, execCtx, interpreterPath)
	}
}

func (e *CmdExecutorImpl) getInterpreter(execCtx *CmdExecutorContext) (string, error) {
	if execCtx.InterpreterAliases != nil && execCtx.Interpreter != "" {
		if mappedInterpreter, ok := execCtx.InterpreterAliases[execCtx.Interpreter]; ok {
			return mappedInterpreter, nil
		}
	}

	if execCtx.Interpreter == "" {
		execCtx.Interpreter = chshare.CmdShell
	}

	if execCtx.Interpreter == chshare.CmdShell ||
		execCtx.Interpreter == chshare.Tacoscript ||
		execCtx.Interpreter == chshare.PowerShell {
		interpreterWithAbsPath, err := getInterpreterAbsolutePath(execCtx.Interpreter)
		if err != nil {
			return "", err
		}

		return interpreterWithAbsPath, nil
	}

	return execCtx.Interpreter, nil
}

func buildCmdInterpreterCmd(ctx context.Context, execCtx *CmdExecutorContext, interpreterPath string) *exec.Cmd {
	// workaround for the issue with escaping args on windows for cmd interpreter https://github.com/golang/go/issues/1849
	cmd := exec.CommandContext(ctx, interpreterPath)
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

	args = append(args, execCtx.Command)

	cmd := exec.CommandContext(ctx, interpreterPath, args...)
	cmd.Dir = execCtx.WorkingDir

	return cmd
}

func buildDefaultCmd(ctx context.Context, execCtx *CmdExecutorContext, interpreterPath string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, interpreterPath, execCtx.Command)
	cmd.Dir = execCtx.WorkingDir

	return cmd
}

func getInterpreterAbsolutePath(interpreter string) (absInterpreterPath string, err error) {
	if !strings.HasSuffix(interpreter, ".exe") {
		interpreter += ".exe"
	}

	path := os.Getenv("path")
	for _, dir := range filepath.SplitList(path) {
		absInterpreterPath := filepath.Join(dir, interpreter)
		d, err := os.Stat(absInterpreterPath)
		if err != nil || d.IsDir() {
			continue
		}

		return absInterpreterPath, nil
	}

	return "", fmt.Errorf("failed to find %s at %%PATH%%: %s: %w", interpreter, path, os.ErrNotExist)
}
