package helpers

import (
	"os"
	"testing"
)

func CleanUp(t *testing.T, pathsToRemove ...string) {
	t.Log("cleaning before test")
	for _, path := range pathsToRemove {
		t.Log("removing", path, os.RemoveAll(path))
	}
}
