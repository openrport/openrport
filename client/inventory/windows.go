//go:build windows
// +build windows

package inventory

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"

	"github.com/openrport/openrport/share/logger"
	"github.com/openrport/openrport/share/models"
)

type WindowsSoftwareInventoryManager struct{}

func NewWindowsSoftwareInventoryManager() *WindowsSoftwareInventoryManager {
	return &WindowsSoftwareInventoryManager{}
}

func (p *WindowsSoftwareInventoryManager) IsAvaliable(ctx context.Context) bool {
	return true
}

func (p *WindowsSoftwareInventoryManager) GetInventory(ctx context.Context, logger *logger.Logger) ([]models.SoftwareInventory, error) {
	keyPaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	var programs []models.SoftwareInventory

	for _, keyPath := range keyPaths {
		key, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.READ)
		if err != nil {
			continue
		}
		defer key.Close()

		names, err := key.ReadSubKeyNames(-1)
		if err != nil {
			continue
		}

		for _, name := range names {
			subKey, err := registry.OpenKey(key, name, registry.READ)
			if err != nil {
				continue
			}

			displayName, _, _ := subKey.GetStringValue("DisplayName")
			displayVersion, _, _ := subKey.GetStringValue("DisplayVersion")
			helpLink, _, _ := subKey.GetStringValue("HelpLink")
			comments, _, _ := subKey.GetStringValue("Comments")
			publisher, _, _ := subKey.GetStringValue("Publisher")

			description := strings.TrimSpace(fmt.Sprintf("%s %s", comments, publisher))
			if description == "" {
				description = helpLink
			}

			if displayName != "" {
				programs = append(programs, models.SoftwareInventory{
					Package: displayName,
					Version: displayVersion,
					Summary: description,
				})
			}
			subKey.Close()
		}
	}

	return programs, nil
}
