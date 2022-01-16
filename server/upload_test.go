package chserver

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandleFileUploads(t *testing.T) {
	cl := clients.New(t).Build()

	testCases := []struct {
		name                string
		wantStatus          int
		fsCallback          func(fs *test.FileAPIMock)
		wantResp            *models.UploadResponseShort
		wantClientInputFile *models.UploadedFile
		fileName            string
		fileContent         string
		formParts           map[string][]string
	}{
		{
			name:       "send file success",
			wantStatus: http.StatusOK,
			wantResp: &models.UploadResponseShort{
				ID:        "id-123",
				Filepath:  "/destination/myfile.txt",
				SizeBytes: 10,
			},
			fsCallback: func(fs *test.FileAPIMock) {
				fs.On("CreateDirIfNotExists", "/data/"+files.DefaultUploadTempFolder, files.DefaultMode).Return(true, nil)

				fileExpectation := func(f io.Reader) bool {
					actualFileContent, err := ioutil.ReadAll(f)

					require.NoError(t, err)

					return string(actualFileContent) == "some content"
				}
				fs.On("CreateFile", "/data/filepush/id-123_rport_filepush", mock.MatchedBy(fileExpectation)).Return(int64(10), []byte("md5-123"), nil)
				fs.On("Remove", "/data/filepush/id-123_rport_filepush").Return(nil)
			},
			fileName:    "file.txt",
			fileContent: "some content",
			formParts: map[string][]string{
				"client": {
					cl.ID,
				},
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
			wantClientInputFile: &models.UploadedFile{
				ID:                   "id-123",
				SourceFilePath:       "/data/filepush/id-123_rport_filepush",
				DestinationPath:      "/destination/myfile.txt",
				DestinationFileMode:  0744,
				DestinationFileOwner: "admin",
				DestinationFileGroup: "group",
				ForceWrite:           true,
				Sync:                 true,
				Md5Checksum:          []byte("md5-123"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			connMock := test.NewConnMock()

			connMock.ReturnOk = true

			done := make(chan bool)
			connMock.DoneChannel = done

			cl.Connection = connMock

			fileAPIMock := test.NewFileAPIMock()
			if tc.fsCallback != nil {
				tc.fsCallback(fileAPIMock)
			}

			al := APIListener{
				insecureForTests: true,
				Server: &Server{
					clientService: NewClientService(
						nil,
						nil,
						clients.NewClientRepository([]*clients.Client{cl}, &hour, testLog),
					),
					config: &Config{
						Server: ServerConfig{
							DataDir:         "/data",
							MaxFilePushSize: int64(10 << 20),
						},
					},
					filesAPI: fileAPIMock,
				},
				Logger: testLog,
			}
			al.initRouter()

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			part, err := writer.CreateFormFile("upload", tc.fileName)
			require.NoError(t, err)

			_, err = io.Copy(part, strings.NewReader(tc.fileContent))
			require.NoError(t, err)

			for key, vals := range tc.formParts {
				for _, val := range vals {
					err = writer.WriteField(key, val)
					require.NoError(t, err)
				}
			}

			err = writer.Close()
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
			req.Header.Add("Content-Type", writer.FormDataContentType())

			rec := httptest.NewRecorder()
			al.router.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)

			actualResponse := &models.UploadResponseShort{}
			err = json.Unmarshal(rec.Body.Bytes(), actualResponse)
			require.NoError(t, err)

			assert.Equal(t, tc.wantResp, actualResponse)

			select {
			case <-done:
				assertClientPayload(t, connMock, tc.wantClientInputFile)
			case <-time.After(time.Second * 2):
				assertClientPayload(t, connMock, tc.wantClientInputFile)
			}
		})
	}
}

func assertClientPayload(t *testing.T, connMock *test.ConnMock, wantClientInputFile *models.UploadedFile) {
	name, wantReply, payload := connMock.InputSendRequest()

	actualInputFile := &models.UploadedFile{}
	err := actualInputFile.FromBytes(payload)
	require.NoError(t, err)

	assert.Equal(t, "upload", name)
	assert.Equal(t, wantClientInputFile, actualInputFile)
	assert.True(t, wantReply)
}
