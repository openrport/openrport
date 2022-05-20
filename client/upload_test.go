package chclient

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudradar-monitoring/rport/share/errors"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const defaultUID = uint32(123)
const defaultGID = uint32(456)

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

func (uopm *UploadOptionsProviderMock) GetProtectedUploadDirs() []string {
	args := uopm.Called()

	denyGlobs := args.Get(0)
	if denyGlobs == nil {
		return nil
	}

	return denyGlobs.([]string)
}

func (uopm *UploadOptionsProviderMock) IsFileReceptionEnabled() bool {
	args := uopm.Called()

	return args.Bool(0)
}

func TestHandleUploadRequest(t *testing.T) {
	testCases := []struct {
		name                  string
		wantUploadedFile      *models.UploadedFile
		fsCallback            func(fs *test.FileAPIMock)
		optionsCallback       func(opts *UploadOptionsProviderMock)
		fileProviderCallback  func(sfpm *SourceFileProviderMock)
		sysUserLookupCallback func(sysUsrLookup *test.SysUserProviderMock)
		wantError             string
		wantResp              *models.UploadResponse
	}{
		{
			name:             "non existing file upload success",
			wantUploadedFile: getValidUploadFile("some content"),
			fsCallback: func(fs *test.FileAPIMock) {
				fs.On("Exist", filepath.Join("destination", "file.txt")).Return(false, nil)

				expectedTempFilePath := filepath.Join("data", files.DefaultUploadTempFolder, "file_temp.txt")
				fs.On("Exist", expectedTempFilePath).Return(false, nil)

				fs.On("CreateDirIfNotExists", filepath.Join("data", files.DefaultUploadTempFolder), files.DefaultMode).Return(true, nil)
				fs.On("CreateDirIfNotExists", "destination", files.DefaultMode).Return(true, nil)

				fileExpectation := func(f io.ReadCloser) bool {
					actualFileContent, err := ioutil.ReadAll(f)

					require.NoError(t, err)

					return string(actualFileContent) == "some content"
				}
				fs.On("CreateFile", expectedTempFilePath, mock.MatchedBy(fileExpectation)).Return(int64(10), nil)

				fileMock := &test.ReadWriteCloserMock{}
				fileMock.Reader = strings.NewReader("some content")
				fileMock.On("Close").Return(nil)

				fs.On("Open", expectedTempFilePath).Return(fileMock, nil)

				fs.On("Rename", expectedTempFilePath, filepath.Join("destination", "file.txt")).Return(nil)
			},
			fileProviderCallback: buildDefaultFileProviderMock(filepath.Join("source", "file_temp.txt"), "some content"),
			optionsCallback:      defaultOptionsCallback,
			wantResp: &models.UploadResponse{
				UploadResponseShort: models.UploadResponseShort{
					ID:        "97e97cdd-135a-4620-ab50-d44025b8fe31",
					Filepath:  filepath.Join("destination", "file.txt"),
					SizeBytes: 10,
				},
				Message: "file successfully copied to destination " + filepath.Join("destination", "file.txt"),
				Status:  "success",
			},
		},
		{
			name: "existing file forced success",
			wantUploadedFile: &models.UploadedFile{
				ID:                   "97e97cdd-135a-4620-ab50-d44025b8fe32",
				SourceFilePath:       filepath.Join("source", "file_temp2.txt"),
				DestinationPath:      filepath.Join("destination", "file2.txt"),
				DestinationFileMode:  0700,
				DestinationFileOwner: "admin",
				DestinationFileGroup: "group",
				ForceWrite:           true,
				Sync:                 false,
				Md5Checksum:          test.Md5Hash("some content"),
			},
			sysUserLookupCallback: func(sysUsrLookup *test.SysUserProviderMock) {
				usr := &user.User{
					Username: "some",
				}
				gr := &user.Group{
					Name: "gr",
				}
				sysUsrLookup.On("GetCurrentUserAndGroup").Return(usr, gr, nil)
			},
			fsCallback: func(fs *test.FileAPIMock) {
				fs.On("Exist", filepath.Join("destination", "file2.txt")).Return(true, nil)

				expectedTempFilePath := filepath.Join("data", files.DefaultUploadTempFolder, "file_temp2.txt")
				fs.On("Exist", expectedTempFilePath).Return(false, nil)

				fs.On("CreateDirIfNotExists", filepath.Join("data", files.DefaultUploadTempFolder), os.FileMode(0700)).Return(true, nil)
				fs.On("CreateDirIfNotExists", "destination", os.FileMode(0700)).Return(true, nil)

				fileMock := &test.ReadWriteCloserMock{}
				fileMock.Reader = strings.NewReader("some content")
				fileMock.On("Close").Return(nil)

				fs.On("Open", expectedTempFilePath).Return(fileMock, nil)

				fs.On("CreateFile", expectedTempFilePath, mock.Anything).Return(int64(12), nil)
				fs.On("Remove", filepath.Join("destination", "file2.txt")).Return(nil)
				fs.On("Rename", expectedTempFilePath, filepath.Join("destination", "file2.txt")).Return(nil)
				fs.On("ChangeOwner", filepath.Join("data", "filepush", "file_temp2.txt"), "admin", "group").Return(nil)
				fs.On("ChangeMode", filepath.Join("data", "filepush", "file_temp2.txt"), os.FileMode(0700)).Return(nil)
			},
			fileProviderCallback: buildDefaultFileProviderMock(filepath.Join("source", "file_temp2.txt"), "some content2"),
			optionsCallback:      defaultOptionsCallback,
			wantResp: &models.UploadResponse{
				UploadResponseShort: models.UploadResponseShort{
					ID:        "97e97cdd-135a-4620-ab50-d44025b8fe32",
					Filepath:  filepath.Join("destination", "file2.txt"),
					SizeBytes: 12,
				},
				Message: "file successfully copied to destination " + filepath.Join("destination", "file2.txt"),
				Status:  "success",
			},
		},
		{
			name: "existing file not forced",
			wantUploadedFile: &models.UploadedFile{
				ID:              "97e97cdd-135a-4620-ab50-d44025b8fe33",
				SourceFilePath:  filepath.Join("source", "file_temp3.txt"),
				DestinationPath: filepath.Join("destination", "file3.txt"),
				Md5Checksum:     []byte("md5_124"),
			},
			fsCallback: func(fs *test.FileAPIMock) {
				fs.On("Exist", filepath.Join("destination", "file3.txt")).Return(true, nil)
			},
			optionsCallback: func(opts *UploadOptionsProviderMock) {
				opts.On("GetProtectedUploadDirs").Return([]string{})
				opts.On("IsFileReceptionEnabled").Return(true)
			},
			wantResp: &models.UploadResponse{
				UploadResponseShort: models.UploadResponseShort{
					ID:       "97e97cdd-135a-4620-ab50-d44025b8fe33",
					Filepath: filepath.Join("destination", "file3.txt"),
				},
				Message: fmt.Sprintf("file %s already exists, should not be synched or overwritten with force", filepath.Join("destination", "file3.txt")),
				Status:  "ignored",
			},
		},
		{
			name: "deny destination folder",
			wantUploadedFile: &models.UploadedFile{
				ID:              "97e97cdd-135a-4620-ab50-d44025b8fe34",
				SourceFilePath:  filepath.Join("source", "file_temp4.txt"),
				DestinationPath: filepath.Join("destination", "file4.txt"),
				Md5Checksum:     []byte("md5_125"),
			},
			optionsCallback: func(opts *UploadOptionsProviderMock) {
				opts.On("GetProtectedUploadDirs").Return([]string{filepath.Join("destination", "*")})
				opts.On("IsFileReceptionEnabled").Return(true)
			},
			wantError: fmt.Sprintf(
				"target path %s matches protected pattern %s, therefore the file push request is rejected",
				filepath.Join("destination", "file4.txt"),
				filepath.Join("destination", "*"),
			),
		},
		{
			name:             "md5 checksum not matching",
			wantUploadedFile: getValidUploadFile("some content non matching"),
			fsCallback: func(fs *test.FileAPIMock) {
				fs.On("Exist", filepath.Join("destination", "file.txt")).Return(false, nil)

				expectedTempFilePath := filepath.Join("data", files.DefaultUploadTempFolder, "file_temp.txt")
				fs.On("Exist", expectedTempFilePath).Return(false, nil)

				fs.On("CreateDirIfNotExists", filepath.Join("data", files.DefaultUploadTempFolder), files.DefaultMode).Return(true, nil)

				fileMock := &test.ReadWriteCloserMock{}
				fileMock.Reader = strings.NewReader("some content")
				fileMock.On("Close").Return(nil)

				fs.On("Open", expectedTempFilePath).Return(fileMock, nil)

				fs.On("CreateFile", expectedTempFilePath, mock.Anything).Return(int64(12), nil)
				fs.On("Remove", expectedTempFilePath).Return(nil)
			},
			fileProviderCallback: buildDefaultFileProviderMock(filepath.Join("source", "file_temp.txt"), "some content"),
			optionsCallback:      defaultOptionsCallback,
			wantError:            "md5 check failed: checksum from server 260c194bdd86828158fda34d0fbd5fcd doesn't equal the calculated checksum",
		},
		{
			name: "file exists, sync on",
			wantUploadedFile: &models.UploadedFile{
				ID:                   "97e97cdd-135a-4620-ab50-d44025b8fe77",
				SourceFilePath:       filepath.Join("source", "file_temp7.txt"),
				DestinationPath:      filepath.Join("destination", "file7.txt"),
				DestinationFileMode:  0744,
				DestinationFileOwner: "admin",
				DestinationFileGroup: "group",
				Sync:                 true,
				Md5Checksum:          test.Md5Hash("some content"),
			},
			fsCallback: func(fs *test.FileAPIMock) {
				fs.On("Exist", filepath.Join("destination", "file7.txt")).Return(true, nil)

				expectedTempFilePath := filepath.Join("data", files.DefaultUploadTempFolder, "file_temp7.txt")
				fs.On("Exist", expectedTempFilePath).Return(false, nil)

				fs.On("CreateDirIfNotExists", filepath.Join("data", files.DefaultUploadTempFolder), os.FileMode(0744)).Return(true, nil)
				fs.On("CreateDirIfNotExists", "destination", os.FileMode(0744)).Return(true, nil)

				fs.On("CreateFile", expectedTempFilePath, mock.Anything).Return(int64(12), nil)

				existingFileMock := &test.ReadWriteCloserMock{}
				existingFileMock.Reader = strings.NewReader("some content")

				fs.On("Open", expectedTempFilePath).Return(existingFileMock, nil)

				existingFileMock2 := &test.ReadWriteCloserMock{}
				existingFileMock2.Reader = strings.NewReader("some content")
				fs.On("Open", filepath.Join("destination", "file7.txt")).Return(existingFileMock2, nil)

				fs.On("GetFileMode", filepath.Join("destination", "file7.txt")).Return(os.FileMode(0744), nil)

				fs.On("GetFileOwnerAndGroup", filepath.Join("destination", "file7.txt")).Return(defaultUID, defaultGID, nil)

				fs.On("Remove", filepath.Join("destination", "file7.txt")).Return(nil)
				fs.On("Rename", expectedTempFilePath, filepath.Join("destination", "file7.txt")).Return(nil)
				fs.On("ChangeOwner", filepath.Join("data", "filepush", "file_temp7.txt"), "admin", "group").Return(nil)
				fs.On("ChangeMode", filepath.Join("data", "filepush", "file_temp7.txt"), os.FileMode(0744)).Return(nil)
			},
			sysUserLookupCallback: func(sysUsrLookup *test.SysUserProviderMock) {
				sysUsrLookup.On("GetUIDByName", "admin").Return(defaultUID, nil)
				sysUsrLookup.On("GetGidByName", "group").Return(defaultGID+1, nil)

				usr := &user.User{
					Username: "some",
				}
				gr := &user.Group{
					Name: "gr",
				}
				sysUsrLookup.On("GetCurrentUserAndGroup").Return(usr, gr, nil)
			},
			fileProviderCallback: buildDefaultFileProviderMock(filepath.Join("source", "file_temp7.txt"), "some"),
			optionsCallback:      defaultOptionsCallback,
			wantResp: &models.UploadResponse{
				UploadResponseShort: models.UploadResponseShort{
					ID:        "97e97cdd-135a-4620-ab50-d44025b8fe77",
					Filepath:  filepath.Join("destination", "file7.txt"),
					SizeBytes: 12,
				},
				Message: "file successfully copied to destination " + filepath.Join("destination", "file7.txt"),
				Status:  "success",
			},
		},
		{
			name: "file not exists, chown ignored",
			wantUploadedFile: &models.UploadedFile{
				ID:                   "97e97cdd-135a-4620-ab50-d44025b8fe78",
				SourceFilePath:       filepath.Join("source", "file_temp8.txt"),
				DestinationPath:      filepath.Join("destination", "file8.txt"),
				DestinationFileMode:  os.FileMode(0744),
				DestinationFileOwner: "admin",
				DestinationFileGroup: "group",
				Md5Checksum:          test.Md5Hash("some content"),
			},
			fsCallback: func(fs *test.FileAPIMock) {
				fs.On("Exist", filepath.Join("destination", "file8.txt")).Return(false, nil)

				expectedTempFilePath := filepath.Join("data", files.DefaultUploadTempFolder, "file_temp8.txt")
				fs.On("Exist", expectedTempFilePath).Return(false, nil)

				fs.On("CreateDirIfNotExists", filepath.Join("data", files.DefaultUploadTempFolder), os.FileMode(0744)).Return(true, nil)
				fs.On("CreateDirIfNotExists", "destination", os.FileMode(0744)).Return(true, nil)

				fs.On("CreateFile", expectedTempFilePath, mock.Anything).Return(int64(12), nil)
				fs.On("ChangeMode", expectedTempFilePath, os.FileMode(0744)).Return(nil)

				existingFileMock := &test.ReadWriteCloserMock{}
				existingFileMock.Reader = strings.NewReader("some content")

				fs.On("Open", expectedTempFilePath).Return(existingFileMock, nil)

				existingFileMock2 := &test.ReadWriteCloserMock{}
				existingFileMock2.Reader = strings.NewReader("some content")
				fs.On("Rename", expectedTempFilePath, filepath.Join("destination", "file8.txt")).Return(nil)
			},
			sysUserLookupCallback: func(sysUsrLookup *test.SysUserProviderMock) {
				sysUsrLookup.On("GetUIDByName", "admin").Return(defaultUID, nil)
				sysUsrLookup.On("GetGidByName", "group").Return(defaultGID+1, nil)

				usr := &user.User{
					Username: "admin",
				}
				gr := &user.Group{
					Name: "group",
				}
				sysUsrLookup.On("GetCurrentUserAndGroup").Return(usr, gr, nil)
			},
			fileProviderCallback: buildDefaultFileProviderMock(filepath.Join("source", "file_temp8.txt"), "some"),
			optionsCallback:      defaultOptionsCallback,
			wantResp: &models.UploadResponse{
				UploadResponseShort: models.UploadResponseShort{
					ID:        "97e97cdd-135a-4620-ab50-d44025b8fe78",
					Filepath:  filepath.Join("destination", "file8.txt"),
					SizeBytes: 12,
				},
				Message: "file successfully copied to destination " + filepath.Join("destination", "file8.txt"),
				Status:  "success",
			},
		},
		{
			name:             "uploads disabled",
			wantUploadedFile: getValidUploadFile(""),
			optionsCallback: func(opts *UploadOptionsProviderMock) {
				opts.On("IsFileReceptionEnabled").Return(false)
			},
			wantError: errors.ErrUploadsDisabled.Error(),
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

			sysUsrLookup := &test.SysUserProviderMock{}
			if tc.sysUserLookupCallback != nil {
				tc.sysUserLookupCallback(sysUsrLookup)
			}

			log := logger.NewLogger("client-upload-test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
			um := &UploadManager{
				FilesAPI:           fileAPIMock,
				OptionsProvider:    optionsProvMock,
				Logger:             log,
				SourceFileProvider: sourceFileProvider,
				SysUserLookup:      sysUsrLookup,
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
			assert.Equal(t, tc.wantResp, actualResp)
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

func getValidUploadFile(content string) *models.UploadedFile {
	return &models.UploadedFile{
		ID:                   "97e97cdd-135a-4620-ab50-d44025b8fe31",
		SourceFilePath:       filepath.Join("source", "file_temp.txt"),
		DestinationPath:      filepath.Join("destination", "file.txt"),
		DestinationFileMode:  0,
		DestinationFileOwner: "",
		DestinationFileGroup: "",
		ForceWrite:           false,
		Sync:                 false,
		Md5Checksum:          test.Md5Hash(content),
	}
}

func defaultOptionsCallback(opts *UploadOptionsProviderMock) {
	opts.On("GetUploadDir").Return(filepath.Join("data", files.DefaultUploadTempFolder))
	opts.On("GetProtectedUploadDirs").Return([]string{})
	opts.On("IsFileReceptionEnabled").Return(true)
}
