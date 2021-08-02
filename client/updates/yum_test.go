package updates

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/share/models"
)

func TestYumPackageMangerIsAvailable(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		Name           string
		DnfError       error
		YumError       error
		ExpectedResult bool
		ExpectedCmd    string
	}{
		{
			Name:           "Only dnf",
			DnfError:       nil,
			YumError:       errors.New("yum: command not found"),
			ExpectedResult: true,
			ExpectedCmd:    "dnf",
		},
		{
			Name:           "Only yum",
			DnfError:       errors.New("dnf: command not found"),
			YumError:       nil,
			ExpectedResult: true,
			ExpectedCmd:    "yum",
		},
		{
			Name:           "Prioritize dnf when both are available",
			DnfError:       nil,
			YumError:       nil,
			ExpectedResult: true,
			ExpectedCmd:    "dnf",
		},
		{
			Name:           "Not available",
			DnfError:       errors.New("dnf: command not found"),
			YumError:       errors.New("yum: command not found"),
			ExpectedResult: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			mr := newMockRunner()
			pm := NewYumPackageManager()
			pm.runner = mr
			mr.Register([]string{"dnf", "help"}, "", tc.DnfError)
			mr.Register([]string{"yum", "help"}, "", tc.YumError)

			result := pm.IsAvailable(ctx)

			assert.Equal(t, tc.ExpectedResult, result)
			if tc.ExpectedCmd != "" {
				assert.Equal(t, tc.ExpectedCmd, pm.cmd)
			}
		})
	}
}

func TestYumPackageMangerGetUpdatesStatus(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		Name                      string
		RebootRequiredFilename    string
		ListUpdatesOutput         string
		ListUpdatesErr            error
		ListSecurityUpdatesOutput string
		ListSecurityUpdatesErr    error
		NeedsRestartingOutput     string
		NeedsRestartingErr        error
		ExpectedResult            *models.UpdatesStatus
		ExpectedError             error
	}{
		{
			Name:           "List updates error",
			ListUpdatesErr: errors.New("some error"),
			ExpectedError:  errors.New("some error"),
		},
		{
			Name:                   "List security updates error",
			ListSecurityUpdatesErr: errors.New("some error"),
			ExpectedError:          errors.New("some error"),
		},
		{
			Name: "No security updates",
			ListUpdatesOutput: `
anaconda-core.x86_64                                               33.16.5.2-1.el8.0.1                       appstream
glibc.x86_64                                                       2.28-161.el8                              baseos
			`,
			ListSecurityUpdatesOutput: "",
			ExpectedResult: &models.UpdatesStatus{
				UpdatesAvailable:         2,
				SecurityUpdatesAvailable: 0,
				UpdateSummaries: []models.UpdateSummary{
					{
						Title:            "anaconda-core",
						Description:      "anaconda-core 33.16.5.2-1.el8.0.1 [x86_64] (appstream)",
						IsSecurityUpdate: false,
					},
					{
						Title:            "glibc",
						Description:      "glibc 2.28-161.el8 [x86_64] (baseos)",
						IsSecurityUpdate: false,
					},
				},
			},
		},
		{
			Name: "With security updates",
			ListUpdatesOutput: `
anaconda-core.x86_64                                               33.16.5.2-1.el8.0.1                       appstream
glibc.x86_64                                                       2.28-161.el8                              baseos
			`,
			ListSecurityUpdatesOutput: `
glibc.x86_64                                                       2.28-161.el8                              baseos
`,
			ExpectedResult: &models.UpdatesStatus{
				UpdatesAvailable:         2,
				SecurityUpdatesAvailable: 1,
				UpdateSummaries: []models.UpdateSummary{
					{
						Title:            "anaconda-core",
						Description:      "anaconda-core 33.16.5.2-1.el8.0.1 [x86_64] (appstream)",
						IsSecurityUpdate: false,
					},
					{
						Title:            "glibc",
						Description:      "glibc 2.28-161.el8 [x86_64] (baseos)",
						IsSecurityUpdate: true,
					},
				},
			},
		},
		{
			Name: "Reboot pending",
			NeedsRestartingOutput: `
Core libraries or services have been updated since boot-up:
  * glibc

  Reboot is required to fully utilize these updates.
  More information: https://access.redhat.com/solutions/27943`,
			NeedsRestartingErr: errors.New("exit status 1"),
			ExpectedResult: &models.UpdatesStatus{
				RebootPending:   true,
				UpdateSummaries: []models.UpdateSummary{},
			},
		},
		{
			Name: "Reboot not pending",
			NeedsRestartingOutput: `
No core libraries or services have been updated since boot-up.
Reboot should not be necessary.`,
			ExpectedResult: &models.UpdatesStatus{
				RebootPending:   false,
				UpdateSummaries: []models.UpdateSummary{},
			},
		},
		{
			Name: "Reboot check not available",
			NeedsRestartingOutput: `
No such command: needs-restarting. Please use /usr/bin/dnf --help
It could be a DNF plugin command, try: "dnf install 'dnf-command(needs-restarting)'"`,
			NeedsRestartingErr: errors.New("exit status 1"),
			ExpectedResult: &models.UpdatesStatus{
				RebootPending:   false,
				UpdateSummaries: []models.UpdateSummary{},
			},
		},
	}

	for _, cmd := range []string{"yum", "dnf"} {
		cmd := cmd
		for _, tc := range testCases {
			tc := tc
			t.Run(fmt.Sprintf("%v/%v", cmd, tc.Name), func(t *testing.T) {
				t.Parallel()

				mr := newMockRunner()
				pm := NewYumPackageManager()
				pm.runner = mr
				pm.cmd = cmd

				mr.Register([]string{cmd, "check-update", "--quiet", "--refresh"}, tc.ListUpdatesOutput, tc.ListUpdatesErr)
				mr.Register([]string{cmd, "check-update", "--quiet", "--security"}, tc.ListSecurityUpdatesOutput, tc.ListSecurityUpdatesErr)
				mr.Register([]string{cmd, "needs-restarting", "-r"}, tc.NeedsRestartingOutput, tc.NeedsRestartingErr)

				result, err := pm.GetUpdatesStatus(ctx, nil /* logger */)

				assert.Equal(t, tc.ExpectedError, err)
				assert.Equal(t, tc.ExpectedResult, result)
			})
		}
	}
}
