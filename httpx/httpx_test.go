package httpx

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rannday/logx"
)

type mockRoundTripper struct {
	resp *http.Response
	err  error
}

func (m *mockRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return m.resp, m.err
}

func captureHTTP(t *testing.T, fn func()) string {
	t.Helper()

	logx.Reset()

	var buf bytes.Buffer

	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	})

	logx.SetLogger(slog.New(handler)) // or direct assignment if internal

	fn()

	return buf.String()
}

func TestTransport_Success(t *testing.T) {
	out := captureHTTP(t, func() {
		rt := &mockRoundTripper{
			resp: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("ok")),
			},
		}

		tr := Transport(rt)

		req := httptest.NewRequest("GET", "https://example.com", nil)
		_, _ = tr.RoundTrip(req)
	})

	if !strings.Contains(out, "status=200") {
		t.Fatalf("expected status log, got: %q", out)
	}
}

func TestTransport_Error(t *testing.T) {
	out := captureHTTP(t, func() {
		rt := &mockRoundTripper{
			err: fmt.Errorf("boom"),
		}

		tr := Transport(rt)

		req := httptest.NewRequest("GET", "https://example.com", nil)
		_, _ = tr.RoundTrip(req)
	})

	if !strings.Contains(out, "network_error=true") {
		t.Fatalf("expected network error log")
	}
}

func TestMiddleware_LogsStatus(t *testing.T) {
	out := captureHTTP(t, func() {
		handler := HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
	})

	if !strings.Contains(out, "status=404") {
		t.Fatalf("expected status 404 log")
	}
}

func TestMiddleware_RecoversPanic(t *testing.T) {
	out := captureHTTP(t, func() {
		handler := HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("boom")
		}))

		req := httptest.NewRequest("GET", "/panic", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
	})

	if !strings.Contains(out, "http handler panic") {
		t.Fatalf("expected panic log")
	}

	if !strings.Contains(out, "stack=") {
		t.Fatalf("expected stack trace")
	}
}
