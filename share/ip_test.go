package chshare

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoteIP(t *testing.T) {

	testCases := []struct {
		Name          string
		XForwardedFor string
		ExpectedIP    string
	}{
		{
			Name:       "no header",
			ExpectedIP: "192.168.0.13",
		},
		{
			Name:          "public ip",
			XForwardedFor: "8.8.8.8",
			ExpectedIP:    "8.8.8.8",
		},
		{
			Name:          "private ip",
			XForwardedFor: "192.168.88.23",
			ExpectedIP:    "192.168.88.23",
		},
		{
			Name:          "public and private ip",
			XForwardedFor: "192.168.88.23,8.8.8.8",
			ExpectedIP:    "8.8.8.8",
		},
		{
			Name:          "comma with space",
			XForwardedFor: "192.168.88.23, 8.8.8.8",
			ExpectedIP:    "8.8.8.8",
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.168.0.13:1234"
			if tc.XForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tc.XForwardedFor)
			}
			ip := RemoteIP(req)

			assert.Equal(t, tc.ExpectedIP, ip)
		})
	}
}
