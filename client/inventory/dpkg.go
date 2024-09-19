//go:build !windows
// +build !windows

package inventory

import (
	"context"
	"strings"

	"github.com/openrport/openrport/share/logger"
	"github.com/openrport/openrport/share/models"
)

type DPKGSoftwareInventoryManager struct {
	runner          Runner
	detectCmd       []string
	getInventoryCmd []string
}

func NewDPKGSoftwareInventoryManager() *DPKGSoftwareInventoryManager {
	return &DPKGSoftwareInventoryManager{
		runner:          &RunnerImpl{},
		detectCmd:       []string{"dpkg-query", "--help"},
		getInventoryCmd: []string{"sudo", "-n", "dpkg-query", "-W", "-f='${binary:Package} ${Version} ${binary:Summary}\n'"},
	}
}

func (p *DPKGSoftwareInventoryManager) IsAvaliable(ctx context.Context) bool {
	_, err := p.runner.Run(ctx, p.detectCmd...)
	return err == nil
}

func (p *DPKGSoftwareInventoryManager) GetInventory(ctx context.Context, logger *logger.Logger) ([]models.SoftwareInventory, error) {
	output, err := p.runner.Run(ctx, p.getInventoryCmd...)
	if err != nil {
		return nil, err
	}

	var result []models.SoftwareInventory
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}

		result = append(result, models.SoftwareInventory{
			Package: strings.ReplaceAll(parts[0], "'", ""),
			Version: parts[1],
			Summary: parts[2],
		})
	}

	return result, nil
}
