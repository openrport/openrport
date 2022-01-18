package chclient

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/test"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type SourceFileProviderMock struct {
	mock.Mock
}

func (sfpm *SourceFileProviderMock) Open(path string) (io.ReadCloser, error) {
	args := sfpm.Called(path)

	f := args.Get(0)

	if f == nil {
		return nil, args.Error(1)
	}

	return f.(io.ReadCloser), args.Error(1)
}

type UploadOptionsProviderMock struct {
	mock.Mock
}

func (uopm *UploadOptionsProviderMock) GetUploadDir() string {
	args := uopm.Called()

	return args.String(0)
}

func (uopm *UploadOptionsProviderMock) GetFilePushDeny() []string {
	args := uopm.Called()

	denyGlobs := args.Get(0)
	if denyGlobs == nil {
		return nil
	}

	return denyGlobs.([]string)
}

func TestHandleUploadRequest(t *testing.T) {
	testCases := []struct {
		name                 string
		wantUploadedFile     *models.UploadedFile
		fsCallback           func(fs *test.FileAPIMock)
		optionsCallback      func(opts *UploadOptionsProviderMock)
		fileProviderCallback func(sfpm *SourceFileProviderMock)
		wantError            string
		wantResp             *models.UploadResponse
	}{
		{
			name:             "non existing file upload success",
			wantUploadedFile: getValidUploadFile(),
			fsCallback: func(fs *test.FileAPIMock) {
				fs.On("Exist", "/destination/file.txt").Return(false, nil)

				expectedTempFilePath := fmt.Sprintf("/data/%s/file_temp.txt", files.DefaultUploadTempFolder)
				fs.On("Exist", expectedTempFilePath).Return(false, nil)

				fs.On("CreateDirIfNotExists", "/data/"+files.DefaultUploadTempFolder, files.DefaultMode).Return(true, nil)
				fs.On("CreateDirIfNotExists", "/destination", files.DefaultMode).Return(true, nil)

				fileExpectation := func(f io.ReadCloser) bool {
					actualFileContent, err := ioutil.ReadAll(f)

					require.NoError(t, err)

					return string(actualFileContent) == "some content"
				}
				fs.On("CreateFile", expectedTempFilePath, mock.MatchedBy(fileExpectation)).Return(int64(10), []byte("md5_123"), nil)
				fs.On("Rename", expectedTempFilePath, "/destination/file.txt").Return(nil)
			},
			fileProviderCallback: buildDefaultFileProviderMock("/source/file_temp.txt", "some content"),
			optionsCallback:      defaultOptionsCallback,
			wantResp: &models.UploadResponse{
				UploadResponseShort: models.UploadResponseShort{
					ID:        "97e97cdd-135a-4620-ab50-d44025b8fe31",
					Filepath:  "/destination/file.txt",
					SizeBytes: 10,
				},
				Message: "file successfully copied to destination",
				Status:  "success",
			},
		},
		{
			name: "existing file forced success",
			wantUploadedFile: &models.UploadedFile{
				ID:                   "97e97cdd-135a-4620-ab50-d44025b8fe32",
				SourceFilePath:       "/source/file_temp2.txt",
				DestinationPath:      "/destination/file2.txt",
				DestinationFileMode:  0700,
				DestinationFileOwner: "admin",
				DestinationFileGroup: "group",
				ForceWrite:           true,
				Sync:                 false,
				Md5Checksum:          []byte("md5_124"),
			},
			fsCallback: func(fs *test.FileAPIMock) {
				fs.On("Exist", "/destination/file2.txt").Return(true, nil)

				expectedTempFilePath := fmt.Sprintf("/data/%s/file_temp2.txt", files.DefaultUploadTempFolder)
				fs.On("Exist", expectedTempFilePath).Return(false, nil)

				fs.On("CreateDirIfNotExists", "/data/"+files.DefaultUploadTempFolder, os.FileMode(0700)).Return(true, nil)
				fs.On("CreateDirIfNotExists", "/destination", os.FileMode(0700)).Return(true, nil)

				fs.On("CreateFile", expectedTempFilePath, mock.Anything).Return(int64(12), []byte("md5_124"), nil)
				fs.On("Remove", "/destination/file2.txt").Return(nil)
				fs.On("Rename", expectedTempFilePath, "/destination/file2.txt").Return(nil)
				fs.On("ChangeOwner", "/data/filepush/file_temp2.txt", "admin", "group").Return(nil)
				fs.On("ChangeMode", "/data/filepush/file_temp2.txt", os.FileMode(0700)).Return(nil)
			},
			fileProviderCallback: buildDefaultFileProviderMock("/source/file_temp2.txt", "some content2"),
			optionsCallback:      defaultOptionsCallback,
			wantResp: &models.UploadResponse{
				UploadResponseShort: models.UploadResponseShort{
					ID:        "97e97cdd-135a-4620-ab50-d44025b8fe32",
					Filepath:  "/destination/file2.txt",
					SizeBytes: 12,
				},
				Message: "file successfully copied to destination",
				Status:  "success",
			},
		},
		{
			name: "existing file not forced",
			wantUploadedFile: &models.UploadedFile{
				ID:              "97e97cdd-135a-4620-ab50-d44025b8fe33",
				SourceFilePath:  "/source/file_temp3.txt",
				DestinationPath: "/destination/file3.txt",
				Md5Checksum:     []byte("md5_124"),
			},
			fsCallback: func(fs *test.FileAPIMock) {
				fs.On("Exist", "/destination/file3.txt").Return(true, nil)
			},
			optionsCallback: func(opts *UploadOptionsProviderMock) {
				opts.On("GetFilePushDeny").Return([]string{})
			},
			wantResp: &models.UploadResponse{
				UploadResponseShort: models.UploadResponseShort{
					ID:       "97e97cdd-135a-4620-ab50-d44025b8fe33",
					Filepath: "/destination/file3.txt",
				},
				Message: "file /destination/file3.txt already exists and sync and force options were not enabled, will skip the request",
				Status:  "ignored",
			},
		},
		{
			name: "deny destination folder",
			wantUploadedFile: &models.UploadedFile{
				ID:              "97e97cdd-135a-4620-ab50-d44025b8fe34",
				SourceFilePath:  "/source/file_temp4.txt",
				DestinationPath: "/destination/file4.txt",
				Md5Checksum:     []byte("md5_125"),
			},
			optionsCallback: func(opts *UploadOptionsProviderMock) {
				opts.On("GetFilePushDeny").Return([]string{"/destination/*"})
			},
			wantError: "target path /destination/file4.txt matches file_push_deny pattern /destination/*, therefore the file push request is rejected",
		},
		{
			name:             "md5 checksum not matching",
			wantUploadedFile: getValidUploadFile(),
			fsCallback: func(fs *test.FileAPIMock) {
				fs.On("Exist", "/destination/file.txt").Return(false, nil)

				expectedTempFilePath := fmt.Sprintf("/data/%s/file_temp.txt", files.DefaultUploadTempFolder)
				fs.On("Exist", expectedTempFilePath).Return(false, nil)

				fs.On("CreateDirIfNotExists", "/data/"+files.DefaultUploadTempFolder, files.DefaultMode).Return(true, nil)

				fs.On("CreateFile", expectedTempFilePath, mock.Anything).Return(int64(12), []byte("md5"), nil)
				fs.On("Remove", expectedTempFilePath).Return(nil)
			},
			fileProviderCallback: buildDefaultFileProviderMock("/source/file_temp.txt", "some content"),
			optionsCallback:      defaultOptionsCallback,
			wantError:            "md5 check failed: checksum from server 6d64355f313233 doesn't equal the calculated checksum 6d6435",
		},
		{
			name: "file exists, sync on",
			wantUploadedFile: &models.UploadedFile{
				ID:                   "97e97cdd-135a-4620-ab50-d44025b8fe77",
				SourceFilePath:       "/source/file_temp7.txt",
				DestinationPath:      "/destination/file7.txt",
				DestinationFileMode:  0744,
				DestinationFileOwner: "admin",
				DestinationFileGroup: "group",
				Sync:                 true,
				Md5Checksum:          []byte("md5_1277"),
			},
			fsCallback: func(fs *test.FileAPIMock) {
				fs.On("Exist", "/destination/file7.txt").Return(true, nil)

				expectedTempFilePath := fmt.Sprintf("/data/%s/file_temp7.txt", files.DefaultUploadTempFolder)
				fs.On("Exist", expectedTempFilePath).Return(false, nil)

				fs.On("CreateDirIfNotExists", "/data/"+files.DefaultUploadTempFolder, os.FileMode(0744)).Return(true, nil)
				fs.On("CreateDirIfNotExists", "/destination", os.FileMode(0744)).Return(true, nil)

				fs.On("CreateFile", expectedTempFilePath, mock.Anything).Return(int64(12), []byte("md5_1277"), nil)

				existingFileMock := &test.ReadWriteCloserMock{}
				existingFileMock.Reader = strings.NewReader("some content")

				fs.On("Open", "/destination/file7.txt").Return(existingFileMock, nil)

				fs.On("Remove", "/destination/file7.txt").Return(nil)
				fs.On("Rename", expectedTempFilePath, "/destination/file7.txt").Return(nil)
				fs.On("ChangeOwner", "/destination/file7.txt", "admin", "group").Return(nil)
				fs.On("ChangeMode", "/destination/file7.txt", os.FileMode(0744)).Return(nil)
			},
			fileProviderCallback: buildDefaultFileProviderMock("/source/file_temp7.txt", "some"),
			optionsCallback:      defaultOptionsCallback,
			wantResp: &models.UploadResponse{
				UploadResponseShort: models.UploadResponseShort{
					ID:        "97e97cdd-135a-4620-ab50-d44025b8fe77",
					Filepath:  "/destination/file7.txt",
					SizeBytes: 12,
				},
				Message: "file successfully copied to destination",
				Status:  "success",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			uploadedFileBytes, err := tc.wantUploadedFile.ToBytes()
			require.NoError(t, err)

			fileAPIMock := test.NewFileAPIMock()
			if tc.fsCallback != nil {
				tc.fsCallback(fileAPIMock)
			}

			optionsProvMock := &UploadOptionsProviderMock{}
			if tc.optionsCallback != nil {
				tc.optionsCallback(optionsProvMock)
			}

			sourceFileProvider := &SourceFileProviderMock{}
			if tc.fileProviderCallback != nil {
				tc.fileProviderCallback(sourceFileProvider)
			}

			log := logger.NewLogger("client-upload-test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
			um := &UploadManager{
				FilesAPI:           fileAPIMock,
				OptionsProvider:    optionsProvMock,
				Logger:             log,
				SourceFileProvider: sourceFileProvider,
			}

			actualResp, err := um.HandleUploadRequest(uploadedFileBytes)
			fileAPIMock.AssertExpectations(t)
			optionsProvMock.AssertExpectations(t)
			sourceFileProvider.AssertExpectations(t)

			if tc.wantError != "" {
				require.EqualError(t, err, tc.wantError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantResp, actualResp)
		})
	}
}

func buildDefaultFileProviderMock(sourceFilePath, content string) func(f *SourceFileProviderMock) {
	return func(f *SourceFileProviderMock) {
		writerBuf := strings.NewReader(content)
		fileMock := &test.ReadCloserMock{
			Reader: writerBuf,
		}

		f.On("Open", sourceFilePath).Return(fileMock, nil)
		fileMock.On("Close").Return(nil)
	}
}

func getValidUploadFile() *models.UploadedFile {
	return &models.UploadedFile{
		ID:                   "97e97cdd-135a-4620-ab50-d44025b8fe31",
		SourceFilePath:       "/source/file_temp.txt",
		DestinationPath:      "/destination/file.txt",
		DestinationFileMode:  0,
		DestinationFileOwner: "",
		DestinationFileGroup: "",
		ForceWrite:           false,
		Sync:                 false,
		Md5Checksum:          []byte("md5_123"),
	}
}

func defaultOptionsCallback(opts *UploadOptionsProviderMock) {
	opts.On("GetUploadDir").Return(filepath.Join("/data", files.DefaultUploadTempFolder))
	opts.On("GetFilePushDeny").Return([]string{})
}
