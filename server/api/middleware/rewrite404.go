package middleware

import (
	"net/http"
	"strings"
)

type NotFoundRewriteResponseWriter struct {
	http.ResponseWriter
	headerWritten bool
	status        int
	header        http.Header
}

func (w *NotFoundRewriteResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = w.ResponseWriter.Header().Clone()
	}
	return w.header
}

func (w *NotFoundRewriteResponseWriter) WriteHeader(status int) {
	w.status = status
	if status != http.StatusNotFound {
		for key, values := range w.header {
			w.ResponseWriter.Header()[key] = values
		}
		w.headerWritten = true
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *NotFoundRewriteResponseWriter) Write(p []byte) (int, error) {
	if w.status != http.StatusNotFound {
		if !w.headerWritten {
			for key, values := range w.header {
				w.ResponseWriter.Header()[key] = values
			}
		}
		return w.ResponseWriter.Write(p)
	}
	return len(p), nil // lie that it was successfully written
}

func Rewrite404(h http.Handler, rewritePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		newW := &NotFoundRewriteResponseWriter{ResponseWriter: w}
		h.ServeHTTP(newW, r)
		if newW.status == http.StatusNotFound {
			r.URL.Path = rewritePath
			h.ServeHTTP(w, r)
		}
	}
}

// Rewrite404ForVueJs Implement VueJS HTML5 history mode by rewriting specific URLs to /
func Rewrite404ForVueJs(h http.Handler, vueHistoryPaths []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		newW := &NotFoundRewriteResponseWriter{ResponseWriter: w}
		h.ServeHTTP(newW, r)
		if newW.status == http.StatusNotFound {
			for _, v := range vueHistoryPaths {
				if strings.HasPrefix(r.URL.Path, "/"+v) {
					r.URL.Path = "/"
				}
			}
			h.ServeHTTP(w, r)
		}
	}
}

func Handle404(next http.Handler, error404Handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		newW := &NotFoundRewriteResponseWriter{ResponseWriter: w}
		next.ServeHTTP(newW, r)
		if newW.status == http.StatusNotFound {
			error404Handler.ServeHTTP(w, r)
		}
	}
}
