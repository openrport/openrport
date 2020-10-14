package chserver

import (
	"os"
	"path"
	"testing"

	"github.com/magiconair/properties/assert"
)

func TestGetJobsDirectory(t *testing.T) {
	wantDir := path.Join("path", "to", "dir", "jobs")
	for _, tc := range []struct {
		name     string
		inputDir string
		wantRes  string
	}{
		{
			name:     "empty",
			inputDir: "",
			wantRes:  "",
		},
		{
			name:     "without path separator at the end",
			inputDir: path.Join("path", "to", "dir"),
			wantRes:  wantDir,
		},
		{
			name:     "with path separator at the end",
			inputDir: path.Join("path", "to", "dir") + string(os.PathSeparator),
			wantRes:  wantDir,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantRes, getJobsDirectory(tc.inputDir))
		})
	}
}
