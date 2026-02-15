package logx

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func capture(t *testing.T, level slog.Level, fn func()) string {
	t.Helper()

	Reset()

	var buf bytes.Buffer

	levelVar.Set(level)
	useColor = false

	base := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level:     levelVar,
		AddSource: false,
	})

	// Match Init() decorator chain
	handler := newStackHandler(base, 0)
	handler = newRedactionHandler(handler)

	logger = slog.New(handler)

	fn()
	return buf.String()
}

func assertContains(t *testing.T, out, want string) {
	t.Helper()
	if !strings.Contains(out, want) {
		t.Fatalf("expected %q in output, got: %q", want, out)
	}
}

//
// ---- Basic Level Tests ----
//

func TestDebug_SuppressedWhenDisabled(t *testing.T) {
	out := capture(t, slog.LevelInfo, func() {
		Debug("hello", "n", 1)
	})

	if out != "" {
		t.Fatalf("expected no output, got: %q", out)
	}
}

func TestDebug_EmitsWhenEnabled(t *testing.T) {
	out := capture(t, slog.LevelDebug, func() {
		Debug("hello", "n", 1)
	})

	assertContains(t, out, "level=DEBUG")
	assertContains(t, out, "hello")
	assertContains(t, out, "n=1")
}

func TestInfo_Emits(t *testing.T) {
	out := capture(t, slog.LevelInfo, func() {
		Info("hi", "who", "there")
	})

	assertContains(t, out, "level=INFO")
	assertContains(t, out, "who=there")
}

func TestWarn_Emits(t *testing.T) {
	out := capture(t, slog.LevelInfo, func() {
		Warn("careful", "n", 2)
	})

	assertContains(t, out, "level=WARN")
}

func TestError_Emits(t *testing.T) {
	out := capture(t, slog.LevelInfo, func() {
		Error("bad", "thing", true)
	})

	assertContains(t, out, "level=ERROR")
	assertContains(t, out, "thing=true")
}

//
// ---- Structured Tests ----
//

func TestWith_AddsFields(t *testing.T) {
	out := capture(t, slog.LevelInfo, func() {
		log := With("component", "api")
		log.Info("request", "method", "GET")
	})

	assertContains(t, out, "component=api")
	assertContains(t, out, "method=GET")
}

func TestInfoContext_Emits(t *testing.T) {
	out := capture(t, slog.LevelInfo, func() {
		ctx := context.Background()
		InfoContext(ctx, "ctx message", "k", "v")
	})

	assertContains(t, out, "ctx message")
	assertContains(t, out, "k=v")
}

//
// ---- Runtime Level Test ----
//

func TestSetLevel_RuntimeChange(t *testing.T) {
	Reset()

	var buf bytes.Buffer
	useColor = false

	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level:     levelVar,
		AddSource: false,
	})

	logger = slog.New(handler)

	SetLevel(slog.LevelError)

	Info("should not print")
	if buf.String() != "" {
		t.Fatalf("expected no output, got: %q", buf.String())
	}

	SetLevel(slog.LevelInfo)

	Info("now prints")
	assertContains(t, buf.String(), "now prints")
}

//
// ---- Hardened Init Test ----
//

func TestInit_FileFailureFallback(t *testing.T) {
	Reset()

	// Force console off and invalid file path
	err := Init(Config{
		Level:    slog.LevelInfo,
		Console:  false,
		FilePath: "/invalid/path/should/fail.log",
	})

	if err == nil {
		t.Fatalf("expected error for invalid file path")
	}

	// Replace logger with buffer to avoid noisy stderr
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level:     levelVar,
		AddSource: false,
	})
	logger = slog.New(handler)

	// Should not panic
	Info("still works")

	if !strings.Contains(buf.String(), "still works") {
		t.Fatalf("expected fallback logger to work")
	}
}

//
// ---- Fatal Test ----
//

func TestFatal_ExitsWithCode1AndLogs(t *testing.T) {
	if os.Getenv("LOGX_FATAL_CHILD") == "1" {
		Reset()
		useColor = false

		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     levelVar,
			AddSource: false,
		})
		logger = slog.New(handler)

		Fatal("boom", "n", 2)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestFatal_ExitsWithCode1AndLogs")
	cmd.Env = append(os.Environ(), "LOGX_FATAL_CHILD=1")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := stderr.String()

	if err == nil {
		t.Fatalf("expected exit code 1")
	}

	assertContains(t, out, "level=ERROR")
	assertContains(t, out, "boom")
	assertContains(t, out, "n=2")
}

func TestErrorErr_AddsErrorFields(t *testing.T) {
	out := capture(t, slog.LevelInfo, func() {
		err := fmt.Errorf("boom")
		ErrorErr("failed", err, "device", "fw1")
	})

	assertContains(t, out, "level=ERROR")
	assertContains(t, out, "failed")
	assertContains(t, out, "error=boom")
	assertContains(t, out, "error_type=*errors.errorString")
	assertContains(t, out, "device=fw1")
}

func TestErrorErrContext_AddsErrorFields(t *testing.T) {
	out := capture(t, slog.LevelInfo, func() {
		ctx := context.Background()
		err := fmt.Errorf("ctxboom")
		ErrorErrContext(ctx, "ctx failed", err)
	})

	assertContains(t, out, "ctx failed")
	assertContains(t, out, "error=ctxboom")
}

func TestTimedLevel_EmitsStartAndComplete(t *testing.T) {
	out := capture(t, slog.LevelInfo, func() {
		ctx := context.Background()

		done := TimedLevel(Logger(), slog.LevelInfo, ctx,
			"operation",
			"id", 1,
		)

		done()
	})

	assertContains(t, out, "operation started")
	assertContains(t, out, "operation completed")
	assertContains(t, out, "duration=")
}

func TestStacktraceLevel_AddsStack(t *testing.T) {
	Reset()

	var buf bytes.Buffer
	useColor = false

	err := Init(Config{
		Level:           slog.LevelInfo,
		Console:         false,
		StacktraceLevel: slog.LevelError,
	})

	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	// Override logger to capture output
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level:     levelVar,
		AddSource: false,
	})
	logger = slog.New(newStackHandler(handler, slog.LevelError))

	Error("boom")

	out := buf.String()

	assertContains(t, out, "level=ERROR")
	assertContains(t, out, "stack=")
}

func TestRedactionHandler_RedactsKeys(t *testing.T) {
	out := capture(t, slog.LevelInfo, func() {
		SetRedactedKeys("password")
		Info("login", "password", "secret", "user", "admin")
	})

	assertContains(t, out, "password=REDACTED")
	assertContains(t, out, "user=admin")
}

func TestSanitizeURL_RedactsQueryParams(t *testing.T) {
	u, _ := url.Parse("https://fw/api?apikey=abc123&name=test")

	s := SanitizeURL(u)

	if strings.Contains(s, "abc123") {
		t.Fatalf("expected apikey to be redacted")
	}

	assertContains(t, s, "apikey=REDACTED")
}
