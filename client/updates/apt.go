package updates

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type AptPackageManager struct {
	runner                 Runner
	descriptionRegex       *regexp.Regexp
	rebootRequiredFilename string
	detectCmd              []string
	updateCacheCmd         []string
	getSummariesCmd        []string
	getCountsCmd           []string
}

type getCountsCmdError error

func NewAptPackageManager() *AptPackageManager {
	return &AptPackageManager{
		runner:                 &RunnerImpl{},
		descriptionRegex:       regexp.MustCompile(`\((.*?)\)`),
		rebootRequiredFilename: "/var/run/reboot-required",
		detectCmd:              []string{"apt-get", "help"},
		updateCacheCmd:         []string{"sudo", "-n", "apt-get", "update", "-o", "Debug::NoLocking=true"},
		getSummariesCmd:        []string{"apt-get", "-s", "-o", "Debug::NoLocking=true", "upgrade"},
		getCountsCmd:           []string{"/usr/lib/update-notifier/apt-check"},
	}
}

func (p *AptPackageManager) IsAvailable(ctx context.Context) bool {
	_, err := p.runner.Run(ctx, p.detectCmd...)
	return err == nil
}

func (p *AptPackageManager) GetUpdatesStatus(ctx context.Context, logger *chshare.Logger) (*models.UpdatesStatus, error) {
	err := p.updatePackageCache(ctx)
	if err != nil {
		return nil, err
	}

	summaries, err := p.getSummaries(ctx)
	if err != nil {
		return nil, err
	}

	rebootPending, err := p.checkRebootRequired()
	if err != nil {
		return nil, err
	}

	// Get the number of available updates and the number of security updates
	// from the output of the command, if is available. Ubuntu does some magic to calculate the updates.
	// As this number is displayed on the system, we give it preference over our own sums.
	// If not available, we count the summaries.
	available, security, err := p.getCounts(ctx)
	if err != nil {
		if _, ok := err.(getCountsCmdError); !ok {
			logger.Errorf("Getting update counts using apt-check failed: %v", err)
		}
		available = len(summaries)
		security = p.countSecurityUpdates(summaries)
	}

	return &models.UpdatesStatus{
		UpdatesAvailable:         available,
		SecurityUpdatesAvailable: security,
		UpdateSummaries:          summaries,
		RebootPending:            rebootPending,
	}, nil
}

func (p *AptPackageManager) getCounts(ctx context.Context) (availableUpdates int, securityUpdates int, err error) {
	output, err := p.runner.Run(ctx, p.getCountsCmd...)
	if err != nil {
		return 0, 0, getCountsCmdError(err)
	}

	parts := strings.SplitN(output, ";", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid output of counts cmd: %v", output)
	}

	available, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	security, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return available, security, nil
}

func (p *AptPackageManager) countSecurityUpdates(summaries []models.UpdateSummary) int {
	count := 0
	for _, s := range summaries {
		if s.IsSecurityUpdate {
			count++
		}
	}
	return count
}

func (p *AptPackageManager) checkRebootRequired() (bool, error) {
	if _, err := os.Stat(p.rebootRequiredFilename); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func (p *AptPackageManager) getSummaries(ctx context.Context) ([]models.UpdateSummary, error) {
	output, err := p.runner.Run(ctx, p.getSummariesCmd...)
	if err != nil {
		return nil, err
	}

	var result []models.UpdateSummary
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "Inst") {
			continue
		}

		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}
		title := parts[1]
		descriptionMatch := p.descriptionRegex.FindStringSubmatch(line)
		description := title
		if len(descriptionMatch) > 1 {
			description = fmt.Sprintf(
				"%s %s",
				title,
				descriptionMatch[1],
			)
		}

		isSecurity := strings.Contains(description, "-security")

		result = append(result, models.UpdateSummary{
			Title:            title,
			Description:      description,
			IsSecurityUpdate: isSecurity,
		})
	}

	return result, nil
}

func (p *AptPackageManager) updatePackageCache(ctx context.Context) error {
	_, err := p.runner.Run(ctx, p.updateCacheCmd...)
	return err
}
