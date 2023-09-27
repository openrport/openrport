package acme

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openrport/openrport/share/logger"
)

func TestHostPolicy(t *testing.T) {
	log := logger.NewLogger("acme-test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
	ctx := context.Background()
	acme := New(log, "", 0)

	acme.AddHost("test1.example.com", "test2.example.com")
	acme.AddHost("https://test3.example.com:443")
	acme.AddHost("https://test4.example.com")

	assert.NoError(t, acme.hostPolicy(ctx, "test1.example.com"))
	assert.NoError(t, acme.hostPolicy(ctx, "test2.example.com"))
	assert.NoError(t, acme.hostPolicy(ctx, "test3.example.com"))
	assert.NoError(t, acme.hostPolicy(ctx, "test4.example.com"))
	assert.Error(t, acme.hostPolicy(ctx, "not-allowed.example.com"))
	assert.Error(t, acme.hostPolicy(ctx, "example.com"))
}
