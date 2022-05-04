//go:build windows
// +build windows

package system

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

func (s *realSystemInfo) virtualizationInfo(ctx context.Context) (virtSystem, virtRole string, err error) {
	ps, err := exec.LookPath("powershell.exe")
	if err != nil {
		return "", "", err
	}
	pscmd := "Get-Service"
	args := []string{"-NoProfile", "-NonInteractive", pscmd}
	cmd := exec.Command(ps, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("error on %s %s: %s", ps, pscmd, stderr.String())
	}

	virtSystem, virtRole = getVirtInfoFromPowershellServicesList(stdout.String())

	return virtSystem, virtRole, nil
}
