//go:build windows
// +build windows

package system

import (
	"context"
	"fmt"
	"os/exec"

	"golang.org/x/text/encoding"
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
