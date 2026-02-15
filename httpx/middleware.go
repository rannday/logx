package httpx

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/rannday/logx"
)

func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{
			ResponseWriter: w,
			status:         200,
		}

		defer func() {
			if rec := recover(); rec != nil {
				rw.status = http.StatusInternalServerError

				logx.Logger().ErrorContext(
					r.Context(),
					"http handler panic",
					"method", r.Method,
					"path", r.URL.Path,
					"panic", rec,
					"stack", string(debug.Stack()),
				)

				http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}

			duration := time.Since(start)

			fields := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration", duration,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"bytes", rw.bytes,
			}

			if id, ok := logx.RequestID(r.Context()); ok {
				fields = append(fields, "request_id", id)
			}

			level := slog.LevelInfo
			switch {
			case rw.status >= 500:
				level = slog.LevelError
			case rw.status >= 400:
				level = slog.LevelWarn
			}

			logx.Logger().Log(r.Context(), level,
				"http request completed",
				fields...,
			)
		}()

		next.ServeHTTP(rw, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

// --- Optional interfaces ---

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := rw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
    return rw.ResponseWriter
}