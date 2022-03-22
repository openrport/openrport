package middleware

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewrite404(t *testing.T) {
	testCases := []struct {
		Path string
	}{
		{
			Path: "/not_found_path",
		}, {
			Path: "/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Path, func(t *testing.T) {
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/" {
					w.Header().Add("test", "redirect")
					http.NotFound(w, r)
					return
				}
				w.Header().Add("test", "success")
				_, err := w.Write([]byte("success"))
				require.NoError(t, err)
			}
			h := Rewrite404(http.HandlerFunc(mockHandler), "/")

			req, err := http.NewRequest("GET", tc.Path, nil)
			require.NoError(t, err)
			rw := httptest.NewRecorder()

			h.ServeHTTP(rw, req)

			result := rw.Result()
			resultBody, err := ioutil.ReadAll(result.Body)
			require.NoError(t, err)
			expectedHeader := []string{"success"}

			assert.Equal(t, 200, result.StatusCode)
			assert.Equal(t, expectedHeader, result.Header.Values("test"))
			assert.Equal(t, "success", string(resultBody))
		})
	}
}

func TestRewrite404ForVueJs(t *testing.T) {
	testCases := []struct {
		Path           string
		ExpectedStatus int
	}{
		{
			Path:           "/not_found_path",
			ExpectedStatus: 404,
		}, {
			Path:           "/dashboard",
			ExpectedStatus: 200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Path, func(t *testing.T) {
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/" {
					http.NotFound(w, r)
					return
				}
				_, err := w.Write([]byte("hello world"))
				require.NoError(t, err)
			}
			vueHistoryPaths := []string{"auth", "dashboard"}
			h := Rewrite404ForVueJs(http.HandlerFunc(mockHandler), vueHistoryPaths)

			req, err := http.NewRequest("GET", tc.Path, nil)
			require.NoError(t, err)
			rw := httptest.NewRecorder()
			h.ServeHTTP(rw, req)
			result := rw.Result()
			assert.Equal(t, tc.ExpectedStatus, result.StatusCode)
		})
	}
}
