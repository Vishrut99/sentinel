package middleware

import (
	"log"
	"net/http"
	"time"
)

// writer wraps http.ResponseWriter to capture the status code.
// The logger runs outside the handler — by the time the handler writes
// the status code to the real writer, it's already sent. We sit in the
// middle (handler → wrapper → real writer) to save it before it's gone.
//
// All three methods (Header, Write, WriteHeader) are required because
// http.ResponseWriter is an interface — Go won't compile unless every
// method is present, even if we only care about WriteHeader.
type writer struct {
	w          http.ResponseWriter
	statusCode int
}

func (w *writer) WriteHeader(code int) {
	w.statusCode = code
	w.w.WriteHeader(code)
}

func (w *writer) Header() http.Header { return w.w.Header() }

func (w *writer) Write(data []byte) (int, error) {
	return w.w.Write(data)
}

// Logger logs: METHOD /path STATUS DURATION for every HTTP request.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		writer := &writer{
			w:          w,
			statusCode: 200,
		}

		next.ServeHTTP(writer, r)

		log.Printf("%s %s %d %s", r.Method, r.URL.Path, writer.statusCode, time.Since(start))
	})
}
