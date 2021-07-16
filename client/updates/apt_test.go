package updates

import (
	"context"
	"errors"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRunner struct {
	outputs map[string]string
	errors  map[string]error
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		outputs: make(map[string]string),
		errors:  make(map[string]error),
	}
}

func (r *mockRunner) Run(ctx context.Context, args ...string) (string, error) {
	key := strings.Join(args, " ")
	return r.outputs[key], r.errors[key]
}

func (r *mockRunner) Register(args []string, output string, err error) {
	key := strings.Join(args, " ")
	r.outputs[key] = output
	r.errors[key] = err
}

func TestAptPackageMangerIsAvailable(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		Name           string
		DetectCmdError error
		ExpectedResult bool
	}{
		{
			Name:           "Apt detected",
			DetectCmdError: nil,
			ExpectedResult: true,
		},
		{
			Name:           "Apt not detected",
			DetectCmdError: errors.New("Command 'apt-get' not found"),
			ExpectedResult: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			mr := newMockRunner()
			apt := NewAptPackageManager()
			apt.runner = mr
			mr.Register(apt.detectCmd, "", tc.DetectCmdError)

			result := apt.IsAvailable(ctx)

			assert.Equal(t, tc.ExpectedResult, result)
		})
	}
}

func TestAptPackageMangerGetUpdatesStatus(t *testing.T) {
	ctx := context.Background()
	tmpFile, err := ioutil.TempFile("", "reboot-required")
	require.NoError(t, err)

	testCases := []struct {
		Name                   string
		RebootRequiredFilename string
		UpdateCacheCmdErr      error
		GetSummariesCmdOutput  string
		GetSummariesCmdErr     error
		GetCountsCmdOutput     string
		GetCountsCmdErr        error
		ExpectedResult         *Status
		ExpectedError          error
	}{
		{
			Name:              "Update package cache error",
			UpdateCacheCmdErr: errors.New("some error"),
			ExpectedError:     errors.New("some error"),
		},
		{
			Name:               "Get summaries error",
			GetSummariesCmdErr: errors.New("some error"),
			ExpectedError:      errors.New("some error"),
		},
		{
			Name: "Summaries with manual counts no security updates",
			GetSummariesCmdOutput: `
NOTE: This is only a simulation!
      apt-get needs root privileges for real execution.
      Keep also in mind that locking is deactivated,
      so don't depend on the relevance to the real current situation!
Reading package lists... Done
Building dependency tree
Reading state information... Done
Calculating upgrade... Done
The following packages have been kept back:
  ubuntu-advantage-tools
The following packages will be upgraded:
  libc6 libc6-dev
2 upgraded, 0 newly installed, 0 to remove and 1 not upgraded.
Inst libc6 [2.31-0ubuntu9.1] (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64])
Inst libc6-dev [2.31-0ubuntu9.1] (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64]) []
Conf libc6 (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64])
Conf libc6-dev (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64])
`,
			GetCountsCmdErr: errors.New("command not found"),
			ExpectedResult: &Status{
				UpdatesAvailable:         2,
				SecurityUpdatesAvailable: 0,
				UpdateSummaries: []Summary{
					{
						Title:            "libc6",
						Description:      "libc6 2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64]",
						IsSecurityUpdate: false,
					},
					{
						Title:            "libc6-dev",
						Description:      "libc6-dev 2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64]",
						IsSecurityUpdate: false,
					},
				},
			},
		},
		{
			Name: "Summaries with manual counts with security updates",
			GetSummariesCmdOutput: `
NOTE: This is only a simulation!
      apt-get needs root privileges for real execution.
      Keep also in mind that locking is deactivated,
      so don't depend on the relevance to the real current situation!
Reading package lists... Done
Building dependency tree
Reading state information... Done
Calculating upgrade... Done
The following packages have been kept back:
  ubuntu-advantage-tools
The following packages will be upgraded:
  libc6 openvpn
2 upgraded, 0 newly installed, 0 to remove and 1 not upgraded.
Inst libc6 [2.31-0ubuntu9.1] (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64])
Inst openvpn [2.4.7-1ubuntu2] (2.4.7-1ubuntu2.20.04.2 Ubuntu:20.04/focal-updates, Ubuntu:20.04/focal-security [amd64])
Conf libc6 (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64])
Conf openvpn [2.4.7-1ubuntu2] (2.4.7-1ubuntu2.20.04.2 Ubuntu:20.04/focal-updates, Ubuntu:20.04/focal-security [amd64])
`,
			GetCountsCmdErr: errors.New("command not found"),
			ExpectedResult: &Status{
				UpdatesAvailable:         2,
				SecurityUpdatesAvailable: 1,
				UpdateSummaries: []Summary{
					{
						Title:            "libc6",
						Description:      "libc6 2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64]",
						IsSecurityUpdate: false,
					},
					{
						Title:            "openvpn",
						Description:      "openvpn 2.4.7-1ubuntu2.20.04.2 Ubuntu:20.04/focal-updates, Ubuntu:20.04/focal-security [amd64]",
						IsSecurityUpdate: true,
					},
				},
			},
		},
		{
			Name: "Summaries with count cmd",
			GetSummariesCmdOutput: `
NOTE: This is only a simulation!
      apt-get needs root privileges for real execution.
      Keep also in mind that locking is deactivated,
      so don't depend on the relevance to the real current situation!
Reading package lists... Done
Building dependency tree
Reading state information... Done
Calculating upgrade... Done
The following packages have been kept back:
  ubuntu-advantage-tools
The following packages will be upgraded:
  libc6 libc6-dev
2 upgraded, 0 newly installed, 0 to remove and 1 not upgraded.
Inst libc6 [2.31-0ubuntu9.1] (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64])
Inst libc6-dev [2.31-0ubuntu9.1] (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64]) []
Conf libc6 (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64])
Conf libc6-dev (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64])
`,
			GetCountsCmdOutput: "3;1",
			ExpectedResult: &Status{
				UpdatesAvailable:         3,
				SecurityUpdatesAvailable: 1,
				UpdateSummaries: []Summary{
					{
						Title:            "libc6",
						Description:      "libc6 2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64]",
						IsSecurityUpdate: false,
					},
					{
						Title:            "libc6-dev",
						Description:      "libc6-dev 2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64]",
						IsSecurityUpdate: false,
					},
				},
			},
		},
		{
			Name: "Summaries with invalid count cmd output",
			GetSummariesCmdOutput: `
NOTE: This is only a simulation!
      apt-get needs root privileges for real execution.
      Keep also in mind that locking is deactivated,
      so don't depend on the relevance to the real current situation!
Reading package lists... Done
Building dependency tree
Reading state information... Done
Calculating upgrade... Done
The following packages have been kept back:
  ubuntu-advantage-tools
The following packages will be upgraded:
  libc6 libc6-dev
2 upgraded, 0 newly installed, 0 to remove and 1 not upgraded.
Inst libc6 [2.31-0ubuntu9.1] (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64])
Inst libc6-dev [2.31-0ubuntu9.1] (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64]) []
Conf libc6 (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64])
Conf libc6-dev (2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64])
`,
			GetCountsCmdOutput: "invalid",
			ExpectedResult: &Status{
				UpdatesAvailable:         2,
				SecurityUpdatesAvailable: 0,
				UpdateSummaries: []Summary{
					{
						Title:            "libc6",
						Description:      "libc6 2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64]",
						IsSecurityUpdate: false,
					},
					{
						Title:            "libc6-dev",
						Description:      "libc6-dev 2.31-0ubuntu9.2 Ubuntu:20.04/focal-updates [amd64]",
						IsSecurityUpdate: false,
					},
				},
			},
		},
		{
			Name:                   "Reboot pending",
			RebootRequiredFilename: tmpFile.Name(),
			ExpectedResult: &Status{
				RebootPending: true,
			},
		},
		{
			Name:                   "Reboot not pending",
			RebootRequiredFilename: "/some/non/existent/file",
			ExpectedResult: &Status{
				RebootPending: false,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			mr := newMockRunner()
			apt := NewAptPackageManager()
			apt.runner = mr

			apt.rebootRequiredFilename = tc.RebootRequiredFilename
			mr.Register(apt.updateCacheCmd, "", tc.UpdateCacheCmdErr)
			mr.Register(apt.getSummariesCmd, tc.GetSummariesCmdOutput, tc.GetSummariesCmdErr)
			mr.Register(apt.getCountsCmd, tc.GetCountsCmdOutput, tc.GetCountsCmdErr)

			result, err := apt.GetUpdatesStatus(ctx, nil /* logger */)

			assert.Equal(t, tc.ExpectedError, err)
			assert.Equal(t, tc.ExpectedResult, result)
		})
	}
}
