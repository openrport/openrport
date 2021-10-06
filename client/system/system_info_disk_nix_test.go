package system

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiskUsage(t *testing.T) {
	err := DiskUsage(context.Background())
	assert.Nil(t, err)
}
