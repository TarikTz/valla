package dashboard

import (
	"net/http"
	"strings"
	"time"
)

// responseRecorder captures the HTTP status code written by a handler.
type responseRecorder struct {
	http.ResponseWriter
	code int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

// WrapHandler wraps next to capture each request's method, path, status, and
// latency, sending the result to logCh in a non-blocking fashion.
func WrapHandler(next http.Handler, logCh chan<- RequestEntry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseRecorder{ResponseWriter: w, code: http.StatusOK}

		// Extract subdomain from Host header (strip port first).
		host := r.Host
		if i := strings.LastIndex(host, ":"); i >= 0 {
			host = host[:i]
		}
		subdomain := host
		if parts := strings.SplitN(host, ".", 2); len(parts) == 2 {
			subdomain = parts[0]
		}

		next.ServeHTTP(rw, r)

		// Non-blocking send; drop the entry if the channel buffer is full.
		select {
		case logCh <- RequestEntry{
			Method:     r.Method,
			Subdomain:  subdomain,
			Path:       r.URL.Path,
			StatusCode: rw.code,
			Latency:    time.Since(start),
		}:
		default:
		}
	})
}
