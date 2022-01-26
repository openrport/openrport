//go:build !windows
// +build !windows

package chserver

// Contains constants applicable only to non windows OS.
const (
	DefaultDataDirectory = "/var/lib/rport"
)
