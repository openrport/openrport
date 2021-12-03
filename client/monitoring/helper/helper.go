package helper

import (
	"context"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var ErrCommandExecutionTimeout = errors.New("command execution timeout exceeded")

// RunCommandWithTimeout runs command and returns it's standard output. If timeout exceeded the returned error is ErrCommandExecutionTimeout
func RunCommandWithTimeout(timeout time.Duration, name string, arg ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, arg...)

	result, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		err = ErrCommandExecutionTimeout
	}
	return result, err
}

func RoundToTwoDecimalPlaces(v float64) float64 {
	return math.Round(v*100) / 100
}

func FloatToIntRoundUP(f float64) int {
	return int(f + 0.5)
}

// StrInSlice returns true if search string found in slice
func StrInSlice(search string, slice []string) bool {
	for _, str := range slice {
		if str == search {
			return true
		}
	}
	return false
}

// GetEnv retrieves the environment variable key. If it does not exist it returns the default.
func GetEnv(key string, dfault string, combineWith ...string) string {
	value := os.Getenv(key)
	if value == "" {
		value = dfault
	}

	switch len(combineWith) {
	case 0:
		return value
	case 1:
		return filepath.Join(value, combineWith[0])
	default:
		all := make([]string, len(combineWith)+1)
		all[0] = value
		copy(all[1:], combineWith)
		return filepath.Join(all...)
	}
}

func HostProc(combineWith ...string) string {
	return GetEnv("HOST_PROC", "/proc", combineWith...)
}
