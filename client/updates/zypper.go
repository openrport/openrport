package updates

import (
	"context"
	"fmt"
	"strings"

	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/models"
)

type ZypperPackageManager struct {
	runner Runner

	detectCmd      []string
	updateCacheCmd []string
	listUpdatesCmd []string
	needsRebootCmd []string
	listPatchesCmd []string
	patchInfoCmd   []string
}

func NewZypperPackageManager() *ZypperPackageManager {
	return &ZypperPackageManager{
		runner: &RunnerImpl{},

		detectCmd:      []string{"zypper", "help"},
		updateCacheCmd: []string{"sudo", "-n", "zypper", "refresh"},
		listUpdatesCmd: []string{"zypper", "--terse", "--quiet", "list-updates"},
		needsRebootCmd: []string{"zypper", "needs-rebooting"},
		listPatchesCmd: []string{"zypper", "--terse", "--quiet", "list-patches"},
		patchInfoCmd:   []string{"zypper", "--terse", "--quiet", "patch-info"},
	}
}

func (p *ZypperPackageManager) IsAvailable(ctx context.Context) bool {
	_, err := p.runner.Run(ctx, p.detectCmd...)
	return err == nil
}

func (p *ZypperPackageManager) GetUpdatesStatus(ctx context.Context, logger *logger.Logger) (*models.UpdatesStatus, error) {
	err := p.updatePackageCache(ctx)
	if err != nil {
		return nil, err
	}

	updates, err := p.listUpdates(ctx)
	if err != nil {
		return nil, err
	}

	info, err := p.getSecurityAndRebootInfo(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]models.UpdateSummary, len(updates))
	securityCount := 0
	for i, update := range updates {
		key := fmt.Sprintf("%s.%s", update.name, update.arch)

		summaries[i] = models.UpdateSummary{
			Title: update.name,
			Description: fmt.Sprintf(
				"%s %s [%s]",
				update.name,
				update.version,
				update.arch,
			),
			IsSecurityUpdate: info[key].isSecurity,
			RebootRequired:   info[key].needsReboot,
		}

		if info[key].isSecurity {
			securityCount++
		}
	}

	rebootPending := p.checkRebootRequired(ctx)

	return &models.UpdatesStatus{
		UpdatesAvailable:         len(updates),
		SecurityUpdatesAvailable: securityCount,
		UpdateSummaries:          summaries,
		RebootPending:            rebootPending,
	}, nil
}

func (p *ZypperPackageManager) listUpdates(ctx context.Context) ([]zypperUpdate, error) {
	output, err := p.runner.Run(ctx, p.listUpdatesCmd...)
	if err != nil {
		return nil, err
	}

	var result []zypperUpdate
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "v |") {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != 6 {
			continue
		}

		result = append(result, zypperUpdate{
			name:    strings.TrimSpace(parts[2]),
			version: strings.TrimSpace(parts[4]),
			arch:    strings.TrimSpace(parts[5]),
		})
	}

	return result, nil
}

type zypperInfo struct {
	isSecurity  bool
	needsReboot bool
}

type zypperUpdate struct {
	name    string
	arch    string
	version string
}

// getSecurityAndRebootInfo uses list-patches to get reboot and security information from patches and maps it to packages
func (p *ZypperPackageManager) getSecurityAndRebootInfo(ctx context.Context) (map[string]zypperInfo, error) {
	output, err := p.runner.Run(ctx, p.listPatchesCmd...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]zypperInfo)
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "Repository") {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != 7 {
			continue
		}

		name := strings.TrimSpace(parts[1])
		category := strings.TrimSpace(parts[2])
		interactive := strings.TrimSpace(parts[4])

		updates, err := p.listUpdatesForPatch(ctx, name)
		if err != nil {
			return nil, err
		}

		for _, update := range updates {
			result[update] = zypperInfo{
				isSecurity:  category == "security",
				needsReboot: interactive == "reboot",
			}
		}
	}

	return result, nil
}

func (p *ZypperPackageManager) listUpdatesForPatch(ctx context.Context, patchName string) ([]string, error) {
	fullCmd := append(p.patchInfoCmd, patchName)
	output, err := p.runner.Run(ctx, fullCmd...)
	if err != nil {
		return nil, err
	}

	result := []string{}
	skipUntilConflicts := true
	for _, line := range strings.Split(output, "\n") {
		if skipUntilConflicts {
			if strings.HasPrefix(line, "Conflicts") {
				skipUntilConflicts = false
			}
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}

		result = append(result, parts[0])
	}

	return result, nil
}

func (p *ZypperPackageManager) checkRebootRequired(ctx context.Context) bool {
	// The error is ignored here, since exit code is the same for reboot required and command does not exist.
	// So we cannot use it to determine if reboot is required.
	output, _ := p.runner.Run(ctx, p.needsRebootCmd...)
	return !strings.Contains(output, "Reboot is probably not necessary")
}

func (p *ZypperPackageManager) updatePackageCache(ctx context.Context) error {
	_, err := p.runner.Run(ctx, p.updateCacheCmd...)
	return err
}
