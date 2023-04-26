package message_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/server/api/message"
)

func TestURLService(t *testing.T) {
	var form url.Values
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		require.NoError(t, err)

		form = r.PostForm
	}))
	defer ts.Close()
	service := message.NewURLService(ts.URL, "https://test.example.com")

	err := service.Send(context.Background(), message.Data{
		SendTo:        "test@example.com",
		Token:         "test-token",
		UserAgent:     "test-user-agent",
		RemoteAddress: "test-remote-address",
		TTL:           time.Minute * 10,
	})
	require.NoError(t, err)

	assert.Equal(t, "test@example.com", form.Get("email"))
	assert.Equal(t, "test-token", form.Get("token"))
	assert.Equal(t, "600", form.Get("ttl"))
	assert.Equal(t, "test-user-agent", form.Get("user_agent"))
	assert.Equal(t, "test-remote-address", form.Get("remote_address"))
	assert.Equal(t, "https://test.example.com", form.Get("url"))
}
