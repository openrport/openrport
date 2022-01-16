package models

import (
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/cloudradar-monitoring/rport/share/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validUploadedFile = &UploadedFile{
	ID:                   "123",
	SourceFilePath:       "/source/test.txt",
	DestinationPath:      "/dest/test.txt",
	DestinationFileMode:  0744,
	DestinationFileOwner: "admin",
	DestinationFileGroup: "gr",
	ForceWrite:           true,
	Sync:                 true,
	Md5Checksum:          []byte("213"),
}

var validUploadedFileRaw = `{"ID":"123","SourceFilePath":"/source/test.txt","DestinationPath":"/dest/test.txt","DestinationFileMode":484,"DestinationFileOwner":"admin","DestinationFileGroup":"gr","ForceWrite":true,"Sync":true,"Md5Checksum":"MjEz"}`

func TestValidateUploadedFile(t *testing.T) {
	testCases := []struct {
		name      string
		fileInput UploadedFile
		wantErr   string
	}{
		{
			name: "empty source path",
			fileInput: UploadedFile{
				SourceFilePath:  "",
				DestinationPath: "lala",
			},
			wantErr: "empty source file name",
		},
		{
			name: "empty destination path",
			fileInput: UploadedFile{
				SourceFilePath:  "lala",
				DestinationPath: "",
			},
			wantErr: "empty destination file path",
		},
		{
			name: "valid",
			fileInput: UploadedFile{
				SourceFilePath:  "lala",
				DestinationPath: "mama",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			actualErr := tc.fileInput.Validate()
			if tc.wantErr == "" {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, tc.wantErr)
			}
		})
	}
}

func TestValidateDestinationPathUploadedFile(t *testing.T) {
	testCases := []struct {
		name         string
		fileInput    UploadedFile
		wantErr      string
		globPatterns []string
	}{
		{
			name: "glob_exact_match",
			fileInput: UploadedFile{
				DestinationPath: "/test/mama.txt",
			},
			wantErr:      "target path /test/mama.txt matches file_push_deny pattern /test, therefore the file push request is rejected",
			globPatterns: []string{"/lele", "/test"},
		},
		{
			name: "glob_exact_no_match",
			fileInput: UploadedFile{
				DestinationPath: "/mele",
			},
			globPatterns: []string{"/lele", "/pele"},
		},
		{
			name: "glob_wildcard_match",
			fileInput: UploadedFile{
				DestinationPath: "/test/lala/mama.txt",
			},
			wantErr:      "target path /test/lala/mama.txt matches file_push_deny pattern /test/*, therefore the file push request is rejected",
			globPatterns: []string{"/test/*"},
		},
		{
			name: "glob_wildcard_no_match",
			fileInput: UploadedFile{
				DestinationPath: "/test/lala/mama.txt",
			},
			globPatterns: []string{"/test/*/*"},
		},
		{
			name: "glob_any_symbol_match",
			fileInput: UploadedFile{
				DestinationPath: "/test/lala/mama.txt",
			},
			wantErr:      "target path /test/lala/mama.txt matches file_push_deny pattern /test/?ala, therefore the file push request is rejected",
			globPatterns: []string{"/test/?ala"},
		},
		{
			name: "glob_any_symbol_no_match",
			fileInput: UploadedFile{
				DestinationPath: "/test/lala/mama.txt",
			},
			globPatterns: []string{"/test/?"},
		},
		{
			name: "glob_symbol_set_match",
			fileInput: UploadedFile{
				DestinationPath: "/test/mama.txt",
			},
			wantErr:      "target path /test/mama.txt matches file_push_deny pattern /[tabc][lae][ts]t, therefore the file push request is rejected",
			globPatterns: []string{"/[tabc][lae][ts]t"},
		},
		{
			name: "glob_symbol_set_no_match",
			fileInput: UploadedFile{
				DestinationPath: "/test/mama.txt",
			},
			globPatterns: []string{"/[abc]est"},
		},
		{
			name: "glob_symbol_range_match",
			fileInput: UploadedFile{
				DestinationPath: "/test/mama.txt",
			},
			wantErr:      "target path /test/mama.txt matches file_push_deny pattern /[a-z]est, therefore the file push request is rejected",
			globPatterns: []string{"/[a-z]est"},
		},
		{
			name: "glob_symbol_range_no_match",
			fileInput: UploadedFile{
				DestinationPath: "/1test/mama.txt",
			},
			globPatterns: []string{"/[a-z]est"},
		},
	}

	log := logger.NewLogger("file-validation", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			actualErr := tc.fileInput.ValidateDestinationPath(tc.globPatterns, log)
			if tc.wantErr == "" {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, tc.wantErr)
			}
		})
	}
}

func TestUploadedFileFromMultipartRequest(t *testing.T) {
	testCases := []struct {
		name             string
		formParts        map[string][]string
		wantUploadedFile *UploadedFile
		wantErr          string
	}{
		{
			name: "valid",
			formParts: map[string][]string{
				"dest": {
					"/destination/myfile.txt",
				},
				"id": {
					"id-123",
				},
				"user": {
					"admin",
				},
				"group": {
					"group",
				},
				"mode": {
					"0744",
				},
				"force": {
					"1",
				},
				"sync": {
					"1",
				},
			},
			wantUploadedFile: &UploadedFile{
				ID:                   "id-123",
				DestinationPath:      "/destination/myfile.txt",
				DestinationFileMode:  os.FileMode(0744),
				DestinationFileOwner: "admin",
				DestinationFileGroup: "group",
				ForceWrite:           true,
				Sync:                 true,
			},
		},
		{
			name: "invalid_file_mode",
			formParts: map[string][]string{
				"mode": {
					"dfasdf",
				},
			},
			wantErr: `failed to parse file mode value dfasdf: strconv.ParseInt: parsing "dfasdf": invalid syntax`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := &http.Request{}
			form := make(url.Values)

			for key, vals := range tc.formParts {
				for _, val := range vals {
					form.Add(key, val)
				}
			}

			req.MultipartForm = &multipart.Form{
				Value: form,
			}

			actualUploadedFile := &UploadedFile{}
			err := actualUploadedFile.FromMultipartRequest(req)

			if tc.wantErr != "" {
				assert.EqualError(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantUploadedFile, actualUploadedFile)
			}
		})
	}
}

func TestToBytes(t *testing.T) {
	bytes, err := validUploadedFile.ToBytes()
	require.NoError(t, err)
	assert.Equal(t, validUploadedFileRaw, string(bytes))
}

func TestFromBytes(t *testing.T) {
	actualUploadedFile := &UploadedFile{}

	err := actualUploadedFile.FromBytes([]byte(validUploadedFileRaw))
	require.NoError(t, err)
	assert.Equal(t, validUploadedFile, actualUploadedFile)
}
