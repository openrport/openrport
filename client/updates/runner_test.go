package updates

import (
	"context"
	"strings"
)

type mockRunner struct {
	outputs map[string]string
	errors  map[string]error
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		outputs: make(map[string]string),
		errors:  make(map[string]error),
	}
}

func (r *mockRunner) Run(ctx context.Context, args ...string) (string, error) {
	key := strings.Join(args, " ")
	// try prefix commands if full is not registered
	for i := len(args) - 1; i > 0; i-- {
		if _, ok := r.outputs[key]; ok {
			break
		}
		key = strings.Join(args[:i], " ")
	}
	return r.outputs[key], r.errors[key]
}

func (r *mockRunner) Register(args []string, output string, err error) {
	key := strings.Join(args, " ")
	r.outputs[key] = output
	r.errors[key] = err
}
