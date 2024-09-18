package inventory

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

type Runner interface {
	Run(context.Context, ...string) (string, error)
}

type RunnerImpl struct{}

func (r *RunnerImpl) Run(ctx context.Context, args ...string) (string, error) {
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...) //nolint:gosec

	cmd.Stderr = stderr
	cmd.Stdout = stdout
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			err = errors.New(stderr.String())
		}
		return stdout.String(), err
	}

	return stdout.String(), nil
}
