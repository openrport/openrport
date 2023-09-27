package updates

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openrport/openrport/share/models"
)

func TestZypperPackageMangerIsAvailable(t *testing.T) { //nolint:dupl
	ctx := context.Background()
	testCases := []struct {
		Name           string
		DetectCmdError error
		ExpectedResult bool
	}{
		{
			Name:           "Zypper detected",
			DetectCmdError: nil,
			ExpectedResult: true,
		},
		{
			Name:           "Zypper not detected",
			DetectCmdError: errors.New("zypper: command not found"),
			ExpectedResult: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			mr := newMockRunner()
			pm := NewZypperPackageManager()
			pm.runner = mr
			mr.Register(pm.detectCmd, "", tc.DetectCmdError)

			result := pm.IsAvailable(ctx)

			assert.Equal(t, tc.ExpectedResult, result)
		})
	}
}

func TestZypperPackageMangerGetUpdatesStatus(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		Name string

		UpdateCacheCmdErr          error
		ListUpdatesCmdOutput       string
		ListUpdatesCmdErr          error
		ListPatchesCmdOutput       string
		ListPatchesCmdErr          error
		PatchInfoCmdOutput         string
		PatchInfoCmdErr            error
		PatchInfoRebootCmdOutput   string
		PatchInfoSecurityCmdOutput string
		NeedsRebootCmdOutput       string

		ExpectedResult *models.UpdatesStatus
		ExpectedError  error
	}{
		{
			Name:              "Update package cache error",
			UpdateCacheCmdErr: errors.New("some error"),
			ExpectedError:     errors.New("some error"),
		},
		{
			Name:              "List updates error",
			ListUpdatesCmdErr: errors.New("some error"),
			ExpectedError:     errors.New("some error"),
		},
		{
			Name:              "List patches error",
			ListPatchesCmdErr: errors.New("some error"),
			ExpectedError:     errors.New("some error"),
		},
		{
			Name: "Patch info error",
			ListPatchesCmdOutput: `

Repository                 | Name                                | Category    | Severity  | Interactive | Status     | Summary
---------------------------+-------------------------------------+-------------+-----------+-------------+------------+------------------------------------
openSUSE-Tumbleweed-Update | update-test-32bit-pkg               | recommended | moderate  | ---         | not needed | Test-update for openSUSE Tumbleweed

`,
			PatchInfoCmdErr: errors.New("some error"),
			ExpectedError:   errors.New("some error"),
		},
		{
			Name: "Updates with no patches",
			ListUpdatesCmdOutput: `
S | Repository              | Name                              | Current Version      | Available Version      | Arch
--+-------------------------+-----------------------------------+----------------------+------------------------+-------
v | openSUSE-Tumbleweed-Oss | evince                            | 40.2-1.1             | 40.4-1.1               | x86_64
v | openSUSE-Tumbleweed-Oss | ruby-common                       | 2.6-4.2              | 2.6-5.1                | noarch
`,
			NeedsRebootCmdOutput: `
No core libraries or services have been updated since the last system boot.
Reboot is probably not necessary.
`,
			ExpectedResult: &models.UpdatesStatus{
				UpdatesAvailable:         2,
				SecurityUpdatesAvailable: 0,
				UpdateSummaries: []models.UpdateSummary{
					{
						Title:            "evince",
						Description:      "evince 40.4-1.1 [x86_64]",
						IsSecurityUpdate: false,
					},
					{
						Title:            "ruby-common",
						Description:      "ruby-common 2.6-5.1 [noarch]",
						IsSecurityUpdate: false,
					},
				},
			},
		},
		{
			Name: "Updates with security and reboot patches",
			ListUpdatesCmdOutput: `
S | Repository              | Name                              | Current Version      | Available Version      | Arch
--+-------------------------+-----------------------------------+----------------------+------------------------+-------
v | openSUSE-Tumbleweed-Oss | glibc                             | 2.33-8.1             | 2.33-9.1               | x86_64
v | openSUSE-Tumbleweed-Oss | kernel-default                    | 5.13.2-1.1           | 5.14.2-1.1             | x86_64
`,
			ListPatchesCmdOutput: `

Repository                 | Name                                | Category    | Severity  | Interactive | Status     | Summary
---------------------------+-------------------------------------+-------------+-----------+-------------+------------+------------------------------------
openSUSE-Tumbleweed-Update | update-test-reboot-needed           | recommended | important | reboot      | not needed | Test-update for openSUSE Tumbleweed
openSUSE-Tumbleweed-Update | update-test-security                | security    | critical  | ---         | not needed | Test-update for openSUSE Tumbleweed

`,
			PatchInfoRebootCmdOutput: `


Information for patch update-test-reboot-needed:
------------------------------------------------
Repository  : openSUSE-Tumbleweed-Update
Name        : update-test-reboot-needed
Version     : 1
Arch        : noarch
Vendor      : BenniBrunner
Status      : not needed
Category    : recommended
Severity    : important
Created On  : Mon 23 May 2016 03:27:48 PM CEST
Interactive : reboot
Summary     : Test-update for openSUSE Tumbleweed
Description :
    This is a recommended test-update with a needed reboot for openSUSE Tumbleweed.
Provides    : patch:update-test-reboot-needed = 1
Conflicts   : [2]
    kernel-default.x86_64 < 3-2.1
    kernel-default.i586 < 3-2.1

`,
			PatchInfoSecurityCmdOutput: `


Information for patch update-test-security:
-------------------------------------------
Repository  : openSUSE-Tumbleweed-Update
Name        : update-test-security
Version     : 1
Arch        : noarch
Vendor      : BenniBrunner
Status      : not needed
Category    : security
Severity    : critical
Created On  : Mon 23 May 2016 03:27:46 PM CEST
Interactive : ---
Summary     : Test-update for openSUSE Tumbleweed
Description :
    This is a critical security-test-update for openSUSE Tumbleweed.
Provides    : patch:update-test-security = 1
Conflicts   : [2]
    glibc.x86_64 < 3-2.1
    glibc.i586 < 3-2.1

`,
			NeedsRebootCmdOutput: `
No core libraries or services have been updated since the last system boot.
Reboot is probably not necessary.
`,
			ExpectedResult: &models.UpdatesStatus{
				UpdatesAvailable:         2,
				SecurityUpdatesAvailable: 1,
				UpdateSummaries: []models.UpdateSummary{
					{
						Title:            "glibc",
						Description:      "glibc 2.33-9.1 [x86_64]",
						IsSecurityUpdate: true,
						RebootRequired:   false,
					},
					{
						Title:            "kernel-default",
						Description:      "kernel-default 5.14.2-1.1 [x86_64]",
						IsSecurityUpdate: false,
						RebootRequired:   true,
					},
				},
			},
		},
		{
			Name: "Reboot pending",
			NeedsRebootCmdOutput: `
Core libraries or services have been updated since the last system boot.
Reboot is required.
`,
			ExpectedResult: &models.UpdatesStatus{
				RebootPending:   true,
				UpdateSummaries: []models.UpdateSummary{},
			},
		},
		{
			Name: "Reboot not pending",
			NeedsRebootCmdOutput: `
No core libraries or services have been updated since the last system boot.
Reboot is probably not necessary.
`,
			ExpectedResult: &models.UpdatesStatus{
				RebootPending:   false,
				UpdateSummaries: []models.UpdateSummary{},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			mr := newMockRunner()
			pm := NewZypperPackageManager()
			pm.runner = mr

			mr.Register(pm.updateCacheCmd, "", tc.UpdateCacheCmdErr)
			mr.Register(pm.listUpdatesCmd, tc.ListUpdatesCmdOutput, tc.ListUpdatesCmdErr)
			mr.Register(pm.listPatchesCmd, tc.ListPatchesCmdOutput, tc.ListPatchesCmdErr)
			mr.Register(pm.patchInfoCmd, tc.PatchInfoCmdOutput, tc.PatchInfoCmdErr)
			mr.Register(pm.needsRebootCmd, tc.NeedsRebootCmdOutput, nil)
			mr.Register(append(pm.patchInfoCmd, "update-test-reboot-needed"), tc.PatchInfoRebootCmdOutput, nil)
			mr.Register(append(pm.patchInfoCmd, "update-test-security"), tc.PatchInfoSecurityCmdOutput, nil)

			result, err := pm.GetUpdatesStatus(ctx, nil /* logger */)

			assert.Equal(t, tc.ExpectedError, err)
			assert.Equal(t, tc.ExpectedResult, result)
		})
	}
}
