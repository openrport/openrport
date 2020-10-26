//+build windows

package chclient

import (
	"context"
	"os/exec"
	"strings"
	"syscall"
)

func (e *CmdExecutorImpl) New(ctx context.Context, shell, command string) *exec.Cmd {
	// workaround for the issue with escaping args on windows for cmd shell https://github.com/golang/go/issues/1849
	if shell == cmdShell {
		cmd := exec.CommandContext(ctx, shell)
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.SysProcAttr.CmdLine = strings.Join(shellOptions[shell], " ") + command
		return cmd
	}

	return e.newCmd(ctx, shell, command)
}
