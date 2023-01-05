//go:build windows
// +build windows

package system

import (
	"context"
	"fmt"
	"os/exec"

	"golang.org/x/text/encoding"
)

// DetectConsoleEncoding returns input and output encoding that interpreter is using. Returns nil, if it's UTF-8
func DetectConsoleEncoding(ctx context.Context, interpreter Interpreter) (*ShellEncoding, error) {
	commandInput, commandOutput := detectEncodingCommand(interpreter)

	// use utf-8 if detection is not supported for interpreter
	if len(commandInput) == 0 {
		return nil, nil
	}

	input, err := detectEncoding(ctx, interpreter, commandInput)
	if err != nil {
		return nil, err
	}

	// empty command for output encoding implies same encoding
	if len(commandOutput) == 0 {
		return &ShellEncoding{
			InputEncoding:  input,
			OutputEncoding: input,
		}, nil
	}

	output, err := detectEncoding(ctx, interpreter, commandOutput)
	if err != nil {
		return nil, err
	}

	return &ShellEncoding{
		InputEncoding:  input,
		OutputEncoding: output,
	}, nil
}

func detectEncoding(ctx context.Context, interpreter Interpreter, command []string) (encoding.Encoding, error) {
	cmd := exec.CommandContext(ctx, interpreter.Get(), command...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("could not detect active console Code Page: %w", err)
	}
	return detectEncodingByCHCPOutput(string(out))
}
