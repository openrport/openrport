package comm_test

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openrport/openrport/share/comm"
	"github.com/openrport/openrport/share/logger"
)

type retryTestInfo struct {
	shouldTry bool
}

var retryInfo *retryTestInfo

func shouldRetryFn(err error) (shouldRetry bool) {
	if retryInfo != nil {
		if retryInfo.shouldTry {
			if strings.Contains(err.Error(), "timeout") {
				return true
			}
		}
	}
	return false
}

func TestShouldSucceedWhenNoError(t *testing.T) {
	retryInfo = nil

	testLog := logger.NewLogger("retries", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	_, err := comm.WithRetry(func() (result any, err error) {
		return nil, nil
	}, shouldRetryFn, 50*time.Millisecond, "test", testLog)

	assert.NoError(t, err)
}

func TestShouldSucceedAfterRetries(t *testing.T) {
	retryInfo = &retryTestInfo{
		shouldTry: true,
	}

	testLog := logger.NewLogger("retries", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
	attempts := 0

	_, err := comm.WithRetry(func() (result any, err error) {
		if attempts < 2 {
			attempts++
			return nil, errors.New("timeout")
		}
		return nil, nil
	}, shouldRetryFn, 50*time.Millisecond, "test", testLog)

	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestShouldFailWhenMaxBusyErrors(t *testing.T) {
	retryInfo = &retryTestInfo{
		shouldTry: true,
	}

	testLog := logger.NewLogger("retries", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
	attempts := 0

	_, err := comm.WithRetry(func() (result any, err error) {
		attempts++
		return nil, errors.New("timeout")
	}, shouldRetryFn, 50*time.Millisecond, "test", testLog)

	assert.EqualError(t, err, "timeout")
	assert.Equal(t, comm.DefaultMaxRetryAttempts, attempts)
}

func TestShouldFailImmediatelyWhenNonBusyError(t *testing.T) {
	retryInfo = &retryTestInfo{
		shouldTry: true,
	}

	testLog := logger.NewLogger("retries", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
	attempts := 0

	_, err := comm.WithRetry(func() (result any, err error) {
		attempts++
		return nil, errors.New("not a time out")
	}, shouldRetryFn, 50*time.Millisecond, "test", testLog)

	assert.EqualError(t, err, "not a time out")
	assert.Equal(t, 1, attempts)
}
