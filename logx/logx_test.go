package logx

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func capture(t *testing.T, fn func()) string {
  t.Helper()

  // Make output deterministic.
  os.Setenv("NO_COLOR", "1")
  colors.Store(false)
  logger.SetFlags(0)

  var buf bytes.Buffer
  oldOut := logger.Writer()
  logger.SetOutput(&buf)
  defer logger.SetOutput(oldOut)

  fn()
  return buf.String()
}

func assertHasLevelAndMsg(t *testing.T, out, level, msg string) {
  t.Helper()
  if !strings.Contains(out, level) {
    t.Fatalf("expected output to contain %q, got: %q", level, out)
  }
  if !strings.Contains(out, msg) {
    t.Fatalf("expected output to contain %q, got: %q", msg, out)
  }
}

func assertHasCaller(t *testing.T, out string) {
  t.Helper()
  // We just want "something_test.go:<digits>:" to show up.
  // This keeps the test stable if line numbers change.
  re := regexp.MustCompile(`\w+_test\.go:\d+:`)
  if !re.MatchString(out) {
    t.Fatalf("expected caller info like *_test.go:<line>:, got: %q", out)
  }
}

func TestDebug_SuppressedWhenDisabled(t *testing.T) {
  off := false
  Init(&off)

  out := capture(t, func() {
    Debug("hello %d", 1)
  })

  if out != "" {
    t.Fatalf("expected no output, got: %q", out)
  }
}

func TestDebug_EmitsWhenEnabled(t *testing.T) {
  on := true
  Init(&on)

  out := capture(t, func() {
    Debug("hello %d", 1)
  })

  assertHasLevelAndMsg(t, out, "[DEBUG]", "hello 1")
  assertHasCaller(t, out)
}

func TestInfo_Emits(t *testing.T) {
  out := capture(t, func() {
    Info("hi %s", "there")
  })

  assertHasLevelAndMsg(t, out, "[INFO]", "hi there")
  assertHasCaller(t, out)
}

func TestWarn_Emits(t *testing.T) {
  out := capture(t, func() {
    Warn("careful %d", 2)
  })

  assertHasLevelAndMsg(t, out, "[WARN]", "careful 2")
  assertHasCaller(t, out)
}

func TestError_Emits(t *testing.T) {
  out := capture(t, func() {
    Error("bad %s", "thing")
  })

  assertHasLevelAndMsg(t, out, "[ERROR]", "bad thing")
  assertHasCaller(t, out)
}

func TestInitNil_DoesNotOverrideCurrentSetting(t *testing.T) {
  off := false
  Init(&off)

  os.Setenv("DEBUG", "1")
  Init(nil) // should NOT override the already-forced false

  out := capture(t, func() {
    Debug("should not print")
  })

  if out != "" {
    t.Fatalf("expected no output, got: %q", out)
  }
}

func TestFatal_ExitsWithCode1AndLogs(t *testing.T) {
  // Subprocess mode: actually call Fatal() which os.Exit(1)s.
  if os.Getenv("LOGX_FATAL_CHILD") == "1" {
    os.Setenv("NO_COLOR", "1")
    colors.Store(false)
    logger.SetFlags(0)

    Fatal("boom %d", 2)
    return
  }

  cmd := exec.Command(os.Args[0], "-test.run=TestFatal_ExitsWithCode1AndLogs")
  cmd.Env = append(os.Environ(),
    "LOGX_FATAL_CHILD=1",
    "NO_COLOR=1",
  )

  var stderr bytes.Buffer
  cmd.Stdout = &bytes.Buffer{}
  cmd.Stderr = &stderr

  err := cmd.Run()
  out := stderr.String()

  // Expect exit code 1.
  if err == nil {
    t.Fatalf("expected non-nil error (exit code 1), got nil; output: %q", out)
  }
  if ee, ok := err.(*exec.ExitError); ok {
    if ee.ExitCode() != 1 {
      t.Fatalf("expected exit code 1, got %d; output: %q", ee.ExitCode(), out)
    }
  } else {
    t.Fatalf("expected *exec.ExitError, got %T: %v; output: %q", err, err, out)
  }

  assertHasLevelAndMsg(t, out, "[FATAL]", "boom 2")
  assertHasCaller(t, out)
}
