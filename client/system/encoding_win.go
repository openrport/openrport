//go:build windows
// +build windows

package system

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
)

var (
	codePageRegexp = regexp.MustCompile(`(\d+)`)
	// mapping for code pages that do not match IANA names
	codePageToIANAMapping = map[string]string{
		"65000": "utf-7",
		"65001": "utf-8",
		"1252":  "windows-1252",
	}
)

// DetectConsoleEncoding returns encoding that system console is using. Returns nil, if it's UTF-8
func DetectConsoleEncoding(ctx context.Context) (encoding.Encoding, error) {
	cmd := exec.CommandContext(ctx, "cmd", "/c", "chcp")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("could not detect active console Code Page: error running 'chcp' command: %w", err)
	}

	return detectEncodingByCHCPOutput(string(out))
}

func detectEncodingByCHCPOutput(chcpOut string) (encoding.Encoding, error) {
	m := codePageRegexp.FindStringSubmatch(chcpOut)
	if len(m) < 2 {
		return nil, fmt.Errorf("could not parse 'chcp' command output: could not find Code Page number in: %q", chcpOut)
	}

	codePage := m[1]
	iana := getIANAByCodePage(codePage)

	// utf-8 is used by default, no need to return encoding
	if iana == "utf-8" {
		return nil, nil
	}

	enc, err := ianaindex.IANA.Encoding(iana)
	if err != nil {
		return nil, fmt.Errorf("could not get Encoding by IANA name using detected Code Page %s: %v", codePage, err)
	}

	if enc == nil {
		return nil, fmt.Errorf("encoding with Code Page %s is not supported", codePage)
	}

	return enc, nil
}

func getIANAByCodePage(codePage string) string {
	if v, ok := codePageToIANAMapping[codePage]; ok {
		return v
	}

	return codePage
}
