package chserver

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/ptr"
	"github.com/cloudradar-monitoring/rport/share/ws"
)

func TestHandleOutputChannel(t *testing.T) {
	log := logger.NewLogger("client-listener-test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
	cl := &ClientListener{Server: &Server{uiJobWebSockets: ws.NewWebSocketCache()}}
	mockConn := &connMock{}
	ws := ws.NewConcurrentWebSocket(mockConn, log)
	cl.Server.uiJobWebSockets.Set("test-jid", ws)

	testCases := []struct {
		Name     string
		Job      models.Job
		Type     string
		Expected interface{}
	}{
		{
			Name: "no ws",
			Job: models.Job{
				JID: "other-jid",
			},
			Type:     models.ChannelStdout,
			Expected: nil,
		},
		{
			Name: "jid",
			Job: models.Job{
				JID: "test-jid",
			},
			Type: models.ChannelStdout,
			Expected: outputChannelData{
				JID: "test-jid",
				Result: &models.JobResult{
					StdOut: "test-output",
				},
			},
		},
		{
			Name: "multi job jid",
			Job: models.Job{
				MultiJobID: ptr.String("test-jid"),
				JID:        "job-jid",
			},
			Type: models.ChannelStdout,
			Expected: outputChannelData{
				JID: "job-jid",
				Result: &models.JobResult{
					StdOut: "test-output",
				},
			},
		},
		{
			Name: "stderr",
			Job: models.Job{
				JID: "test-jid",
			},
			Type: models.ChannelStderr,
			Expected: outputChannelData{
				JID: "test-jid",
				Result: &models.JobResult{
					StdErr: "test-output",
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			// t.Parallel()

			jobData, err := json.Marshal(tc.Job)
			require.NoError(t, err)

			reader, writer := io.Pipe()

			wg := sync.WaitGroup{}
			wg.Add(1)

			go func() {
				defer wg.Done()
				err := cl.handleOutputChannel(tc.Type, jobData, log, reader)
				require.NoError(t, err)
			}()

			_, err = writer.Write([]byte("test-output"))
			require.NoError(t, err)

			writer.Close()

			wg.Wait()
			assert.Equal(t, tc.Expected, mockConn.LastWrite)
		})
	}
}

type connMock struct {
	ws.Conn

	LastWrite interface{}
}

func (c *connMock) WriteJSON(data interface{}) error {
	c.LastWrite = data
	return nil
}
