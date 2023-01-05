//go:build !windows
// +build !windows

package system

import (
	"context"
)

// DetectConsoleEncoding returns encoding that interpreter is using. Returns nil, if it's UTF-8
func DetectConsoleEncoding(ctx context.Context, interpreter Interpreter) (*ShellEncoding, error) {
	// impl only for windows
	return nil, nil
}
