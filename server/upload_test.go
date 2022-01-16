package chserver

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/test"
)

func TestHandleFileUploads(t *testing.T) {
	cl := clients.New(t).Build()

	testCases := []struct {
		name           string
		expectedStatus int
		fsCallback     func(fs *test.FileAPIMock)
	}{
		{
			name:           "send file success",
			expectedStatus: http.StatusOK,
			fsCallback: func(fs *test.FileAPIMock) {
				fs.On("CreateDirIfNotExists", "/data/"+files.DefaultUploadTempFolder, files.DefaultMode).Return(true, nil)
				fs.On("CreateFile", mock.Anything, mock.Anything).Return(int64(10), []byte("md5-123"), nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			connMock := test.NewConnMock()
			connMock.ReturnOk = true
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

			part, err := writer.CreateFormFile("upload", "file.txt")
			require.NoError(t, err)

			_, err = io.Copy(part, strings.NewReader("some file"))
			require.NoError(t, err)

			err = writer.WriteField("client", cl.ID)
			require.NoError(t, err)

			err = writer.WriteField("dest", "/destination/myfile.txt")
			require.NoError(t, err)

			err = writer.Close()
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
			req.Header.Add("Content-Type", writer.FormDataContentType())

			rec := httptest.NewRecorder()
			al.router.ServeHTTP(rec, req)

			fileAPIMock.AssertExpectations(t)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			name, _, _ := connMock.InputSendRequest()
			assert.Equal(t, "", name)
		})
	}
}
