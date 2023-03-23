package models

import (
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/realvnc-labs/rport/share/logger"

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
		name            string
		fileInput       UploadedFile
		destinationPath string
		wantErr         string
		globPatterns    []string
	}{
		{
			name:            "dir_exact_match",
			destinationPath: "/test/dfasjdfldjf.txt",
			wantErr:         "target path /test matches protected pattern /test, therefore the file push request is rejected",
			globPatterns:    []string{"/lele", "/test"},
		},
		{
			name:            "dir_exact_no_match",
			destinationPath: "/mele",
			globPatterns:    []string{"/lele", "/pele"},
		},
		{
			name:            "dir_wildcard_match",
			destinationPath: "/test/lala/sdjfaldjfla.txt",
			wantErr:         "target path /test/lala matches protected pattern /test/*, therefore the file push request is rejected",
			globPatterns:    []string{"/test/*"},
		},
		{
			name:            "dir_wildcard_no_match",
			destinationPath: "/test/lala/daldjfdfasd.txt",
			globPatterns:    []string{"/test/*/*.csv"},
		},
		{
			name:            "dir_any_symbol_match",
			destinationPath: "/test/lala/feqrer13231.txt",
			wantErr:         "target path /test/lala matches protected pattern /test/?ala, therefore the file push request is rejected",
			globPatterns:    []string{"/test/?ala"},
		},
		{
			name:            "dir_any_symbol_no_match",
			destinationPath: "/test/lala/3142343da.txt",
			globPatterns:    []string{"/test/?"},
		},
		{
			name:            "dir_symbol_set_match",
			destinationPath: "/test/34123dfasdfaqw.txt",
			wantErr:         "target path /test matches protected pattern /[tabc][lae][ts]t, therefore the file push request is rejected",
			globPatterns:    []string{"/[tabc][lae][ts]t"},
		},
		{
			name:            "dir_symbol_set_no_match",
			destinationPath: "/test/213daff1r31.txt",
			globPatterns:    []string{"/[abc]est"},
		},
		{
			name:            "dir_symbol_range_match",
			destinationPath: "/test/3123fdfadfwqefqew.txt",
			wantErr:         "target path /test matches protected pattern /[a-z]est, therefore the file push request is rejected",
			globPatterns:    []string{"/[a-z]est"},
		},
		{
			name:            "dir_symbol_range_no_match",
			destinationPath: "/1test/e123efddafqer2§fr.txt",
			globPatterns:    []string{"/[a-z]est"},
		},
		{
			name:            "file_full_match",
			destinationPath: "/etc/er122f1fefgtu7i.txt",
			wantErr:         "target path /etc/er122f1fefgtu7i.txt matches protected pattern /etc/er122f1fefgtu7i.txt, therefore the file push request is rejected",
			globPatterns:    []string{"/etc/*.csv", "/etc/er122f1fefgtu7i.txt"},
		},
		{
			name:            "file_wildcard_match",
			destinationPath: "/etc/34123oi40.txt",
			wantErr:         "target path /etc/34123oi40.txt matches protected pattern /etc/*.txt, therefore the file push request is rejected",
			globPatterns:    []string{"/etc/*.txt"},
		},
		{
			name:            "file_wildcard_not_match",
			destinationPath: "/etc/rqewjrjva'.txt",
			globPatterns:    []string{"/etc/*.csv"},
		},
		{
			name:            "file_any_symbol_match",
			destinationPath: "/etc/pi093899.txt",
			wantErr:         "target path /etc/pi093899.txt matches protected pattern /etc/?i093899.txt, therefore the file push request is rejected",
			globPatterns:    []string{"/etc/?i093899.txt"},
		},
		{
			name:            "file_any_symbol_not_match",
			destinationPath: "/etc/ma.txt",
			globPatterns:    []string{"/etc/?.txt"},
		},
		{
			name:            "file_symbol_set_match",
			destinationPath: "/etc/abc.txt",
			wantErr:         "target path /etc/abc.txt matches protected pattern /etc/[abc][b][bac].txt, therefore the file push request is rejected",
			globPatterns:    []string{"/etc/[abc][b][bac].txt"},
		},
		{
			name:            "file_symbol_set_not_match",
			destinationPath: "/etc/abd.txt",
			globPatterns:    []string{"/etc/[abc][b][bac].txt"},
		},
		{
			name:            "file_symbol_range_match",
			destinationPath: "/etc/z1c.txt",
			wantErr:         "target path /etc/z1c.txt matches protected pattern /etc/[a-z][0-9][a-c].txt, therefore the file push request is rejected",
			globPatterns:    []string{"/etc/[a-z][0-9][a-c].txt"},
		},
		{
			name:            "file_symbol_range_not_match",
			destinationPath: "/etc/z1c.txt",
			globPatterns:    []string{"/etc/[a-y][0-9][a-c].txt"},
		},
	}

	log := logger.NewLogger("file-validation", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fileInput := UploadedFile{
				DestinationPath: tc.destinationPath,
			}
			actualErr := fileInput.ValidateDestinationPath(tc.globPatterns, log)
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
