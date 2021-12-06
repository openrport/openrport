//go:build !windows
// +build !windows

package system

import (
	"context"

	"golang.org/x/text/encoding"
)

// DetectConsoleEncoding returns encoding that system console is using. Returns nil, if it's UTF-8
func DetectConsoleEncoding(ctx context.Context) (encoding.Encoding, error) {
	// impl only for windows
	return nil, nil
}
