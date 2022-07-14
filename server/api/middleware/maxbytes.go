package middleware

import (
	"fmt"
	"net/http"
)

func MaxBytes(f http.Handler, maxBytes int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		if err := r.ParseForm(); err != nil {
			http.Error(w, fmt.Sprintf("Request data exceeds the limit of %d bytes: %s", maxBytes, err), http.StatusBadRequest)
			return
		}
		f.ServeHTTP(w, r)
	}
}
