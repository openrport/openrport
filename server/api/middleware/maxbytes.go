package middleware

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func MaxBytes(f http.Handler, maxBytes int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dec := json.NewDecoder(r.Body)
		fmt.Printf(" MaxBytes decoded data: %v\n", dec)

		fmt.Printf(" MaxBytes Request data - limit of bytes: %d\n", maxBytes)
		b, _ := ioutil.ReadAll(r.Body)
		fmt.Printf(" MaxBytes Request data - len: %d\n", len(b))
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		if err := r.ParseForm(); err != nil {
			http.Error(w, fmt.Sprintf("Request data exceeds the limit of %d bytes: %s", maxBytes, err), http.StatusBadRequest)
			return
		}
		f.ServeHTTP(w, r)
		fmt.Printf(" MaxBytes REENTERED")
	}
}
