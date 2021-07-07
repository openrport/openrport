//+build windows

package chclient

import (
	"context"
	"strings"
)

func (s *realSystemInfo) virtualizationInfo(ctx context.Context) (virtSystem, virtRole string, err error) {
	cmd := s.cmdExec.New(ctx, "powerShell", "Get-Service", "", false)
	execRes, err := cmd.CombinedOutput()

	if err != nil {
		return "", "", err
	}

	sysInfo := strings.TrimSpace(string(execRes))

	virtSystem, virtRole = getVirtInfoFromPowershellServicesList(sysInfo)

	return virtSystem, virtRole, nil
}
