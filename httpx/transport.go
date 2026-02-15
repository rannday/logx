package httpx

// httpx contains helpers for instrumenting HTTP servers and clients.
// This file implements a RoundTripper that logs outbound requests and
// optionally captures small request/response bodies (with redaction).

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rannday/logx"
)

// TransportLogger wraps an existing RoundTripper and logs outbound requests.
type TransportLogger struct {
	rt     http.RoundTripper
	logger *slog.Logger

	// LogBody controls whether request/response bodies are captured and logged.
	LogBody bool
	// MaxBodyLogBytes limits how many bytes are read from bodies for logging.
	// Only bodies with a known ContentLength <= MaxBodyLogBytes are captured.
	// If 0, default is 32*1024.
	MaxBodyLogBytes int
}

// NewTransportLogger constructs a TransportLogger. If rt is nil, http.DefaultTransport
// is used. If logger is nil, the transport will use the request context logger or
// the package global logger.
func NewTransportLogger(rt http.RoundTripper, logger *slog.Logger) *TransportLogger {
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &TransportLogger{rt: rt, logger: logger}
}

// EnableBodyLogging enables body capture and sets a maximum capture size.
func (t *TransportLogger) EnableBodyLogging(maxBytes int) *TransportLogger {
	t.LogBody = true
	if maxBytes <= 0 {
		t.MaxBodyLogBytes = 32 * 1024
	} else {
		t.MaxBodyLogBytes = maxBytes
	}
	return t
}

func redactJSON(b []byte, redactedKeys []string) []byte {
	s := string(b)
	for _, k := range redactedKeys {
		re := regexp.MustCompile(`(?i)"` + regexp.QuoteMeta(k) + `"\s*:\s*"([^"]*)"`)
		s = re.ReplaceAllString(s, `"`+k+`":"REDACTED"`)
	}
	return []byte(s)
}

func redactForm(s string, redactedKeys []string) string {
	vals, _ := url.ParseQuery(s)
	for _, k := range redactedKeys {
		if _, ok := vals[k]; ok {
			vals.Set(k, "REDACTED")
		}
	}
	return vals.Encode()
}

func (t *TransportLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	// choose logger: explicit -> context -> global
	var l *slog.Logger
	if t.logger != nil {
		l = t.logger
	} else {
		l = logx.LoggerFromContext(req.Context())
	}

	// build fields
	fields := []any{
		"method", req.Method,
		"url", logx.SanitizeURL(req.URL),
	}

	// optionally capture request body (only for small, known-size bodies)
	if t.LogBody && req.Body != nil && req.ContentLength >= 0 {
		max := t.MaxBodyLogBytes
		if max == 0 {
			max = 32 * 1024
		}
		if req.ContentLength <= int64(max) {
			if bodyBytes, err := io.ReadAll(req.Body); err == nil {
				// restore request body for actual transport
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

				ct := req.Header.Get("Content-Type")
				redacted := ""
				if strings.Contains(ct, "application/json") {
					redacted = string(redactJSON(bodyBytes, logx.ListRedactedKeys()))
				} else if strings.Contains(ct, "application/x-www-form-urlencoded") {
					redacted = redactForm(string(bodyBytes), logx.ListRedactedKeys())
				} else {
					// default: include as string (truncated)
					if len(bodyBytes) > max {
						redacted = string(bodyBytes[:max])
					} else {
						redacted = string(bodyBytes)
					}
				}

				fields = append(fields, "req_body", redacted)
			}
		} else {
			fields = append(fields, "req_body_skipped", true)
		}
	}

	// propagate request id header from context if present
	if id, ok := logx.RequestID(req.Context()); ok {
		if req.Header.Get("X-Request-ID") == "" {
			req.Header.Set("X-Request-ID", id)
		}
	}

	start := time.Now()
	resp, err := t.rt.RoundTrip(req)
	duration := time.Since(start)

	// append duration
	fields = append(fields, "duration", duration)

	if err != nil {
		fields = append(fields, "error", err)
		l.Log(req.Context(), slog.LevelError, "http client request", fields...)
		return resp, err
	}

	// optionally capture small response bodies for logging
	if t.LogBody && resp != nil && resp.Body != nil {
		max := t.MaxBodyLogBytes
		if max == 0 {
			max = 32 * 1024
		}
		if resp.ContentLength >= 0 && resp.ContentLength <= int64(max) {
			if bodyBytes, err := io.ReadAll(resp.Body); err == nil {
				// restore response body for caller
				resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

				ct := resp.Header.Get("Content-Type")
				redacted := ""
				if strings.Contains(ct, "application/json") {
					redacted = string(redactJSON(bodyBytes, logx.ListRedactedKeys()))
				} else if strings.Contains(ct, "application/x-www-form-urlencoded") {
					redacted = redactForm(string(bodyBytes), logx.ListRedactedKeys())
				} else {
					if len(bodyBytes) > max {
						redacted = string(bodyBytes[:max])
					} else {
						redacted = string(bodyBytes)
					}
				}

				fields = append(fields, "resp_body", redacted)
			}
		} else {
			fields = append(fields, "resp_body_skipped", true)
		}
	}

	fields = append(fields, "status", resp.StatusCode)

	level := slog.LevelInfo
	switch {
	case resp.StatusCode >= 500:
		level = slog.LevelError
	case resp.StatusCode >= 400:
		level = slog.LevelWarn
	}

	l.Log(req.Context(), level, "http client request completed", fields...)
	return resp, nil
}
