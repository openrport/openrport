//go:build windows
// +build windows

package system

import (
	"context"
	"fmt"
	"os/exec"

	"golang.org/x/text/encoding"
)

// DetectConsoleEncoding returns encoding that interpreter is using. Returns nil, if it's UTF-8
func DetectConsoleEncoding(ctx context.Context, interpreter Interpreter) (encoding.Encoding, error) {
	command := detectEncodingCommand(interpreter)

	// use utf-8 if detection is not supported
	if len(command) == 0 {
		return nil, nil
	}

	cmd := exec.CommandContext(ctx, interpreter.Get(), command...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("could not detect active console Code Page: %w", err)
	}
	return detectEncodingByCHCPOutput(string(out))
}
