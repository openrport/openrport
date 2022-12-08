package system

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func TestDetectCmdOutputEncoding(t *testing.T) {
	testCases := []struct {
		Name         string
		CmdOutput    string
		WantEncoding encoding.Encoding
		WantErr      error
	}{
		{
			Name:         "Code page 850",
			CmdOutput:    "Aktive Codepage: 850.",
			WantEncoding: charmap.CodePage850,
		},
		{
			Name:      "utf-7",
			CmdOutput: "Active code page: 65000.",
			WantErr:   fmt.Errorf("encoding with Code Page %s is not supported", "65000"),
		},
		{
			Name:      "not supported",
			CmdOutput: "Active code page: 869.",
			WantErr:   fmt.Errorf("encoding with Code Page %s is not supported", "869"),
		},
		{
			Name:         "utf-8",
			CmdOutput:    "Active code page: 65001.",
			WantEncoding: nil,
		},
		{
			Name:         "Code page 437",
			CmdOutput:    "Active code page: 437.",
			WantEncoding: charmap.CodePage437,
		},
		{
			Name:         "Code page 1252",
			CmdOutput:    "Active Codepage: 1252.",
			WantEncoding: charmap.Windows1252,
		},
		{
			Name:      "unknown",
			CmdOutput: "Active code page: 936.",
			WantErr:   fmt.Errorf("could not get Encoding by IANA name using detected Code Page %s: %v", "936", errors.New("ianaindex: invalid encoding name")),
		},
		{
			Name:      "invalid",
			CmdOutput: "some unknown output",
			WantErr:   fmt.Errorf("could not parse 'chcp' command output: could not find Code Page number in: %q", "some unknown output"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			gotEnc, gotErr := detectEncodingByCHCPOutput(tc.CmdOutput)
			assert.Equal(t, tc.WantErr, gotErr)
			assert.Equal(t, tc.WantEncoding, gotEnc)
		})
	}
}

func TestDetectEncodingCommand(t *testing.T) {
	testCases := []struct {
		Interpreter string
		Want        []string
	}{
		{
			Interpreter: chshare.CmdShell,
			Want:        detectEncodingCmd,
		},
		{
			Interpreter: chshare.PowerShell,
			Want:        detectEncodingPowershell,
		},
		{
			Interpreter: chshare.UnixShell,
			Want:        nil,
		},
		{
			Interpreter: chshare.Tacoscript,
			Want:        nil,
		},
		{
			Interpreter: `C:\Program Files\PowerShell\7\pwsh.exe`,
			Want:        detectEncodingPowershell,
		},
		{
			Interpreter: `C:\Program Files\Git\bin\bash.exe`,
			Want:        nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Interpreter, func(t *testing.T) {
			t.Parallel()

			interpreter := Interpreter{
				InterpreterNameFromInput: tc.Interpreter,
			}

			got := detectEncodingCommand(interpreter)
			assert.Equal(t, tc.Want, got)
		})
	}
}
