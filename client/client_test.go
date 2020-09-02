package chclient

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Foo") != "Bar" {
			t.Fatal("expected header Foo to be 'Bar'")
		}
	}))
	// Close the server when test finishes
	defer server.Close()

	config := Config{
		Fingerprint: "",
		Auth:        "",
		Connection: ConnectionOptions{
			KeepAlive:        time.Second,
			MaxRetryCount:    0,
			MaxRetryInterval: time.Second,
			HeadersRaw:       []string{"Foo: Bar"},
		},
		Server:  server.URL,
		Remotes: []string{"192.168.0.5:3000:google.com:80"},
	}
	err := config.ParseAndValidate()
	if err != nil {
		log.Fatal(err)
	}
	c, err := NewClient(&config)
	if err != nil {
		log.Fatal(err)
	}
	if err = c.Run(); err != nil {
		log.Fatal(err)
	}
}
