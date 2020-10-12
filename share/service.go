package chshare

import (
	"errors"
	"fmt"

	"github.com/kardianos/service"
)

// HandleServiceCommand handles string command and executes appropriate method.
// We do not use service.Control, because on uninstall it leaves the service running.
func HandleServiceCommand(svc service.Service, command string) error {
	switch command {
	case "install":
		if err := svc.Install(); err != nil {
			return err
		}
		fmt.Println("Service installed")
	case "uninstall":
		status, err := svc.Status()
		if err != nil {
			return err
		}
		if status == service.StatusRunning {
			if err := svc.Stop(); err != nil {
				return err
			}
		}
		if err := svc.Uninstall(); err != nil {
			return err
		}
		fmt.Println("Service uninstalled")
	case "start":
		if err := svc.Start(); err != nil {
			return err
		}
		fmt.Println("Service started")
	case "stop":
		if err := svc.Stop(); err != nil {
			return err
		}
		fmt.Println("Service stopped")
	default:
		return errors.New("Unknown service command")
	}
	return nil
}
