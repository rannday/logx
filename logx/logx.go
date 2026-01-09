package logx

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
)

// ANSI color escape sequences.
// These are only emitted when color output is enabled.
const (
  colorReset   = "\033[0m"
  colorRed     = "\033[31m" // Error/Fatal
  colorGreen   = "\033[32m" // Info
  colorYellow  = "\033[33m" // Warn
  colorCyan    = "\033[36m" // Debug
  colorMagenta = "\033[35m" // File names
)

// Package-level logger and state.
// - logger: single shared stderr logger
// - debug: enables/disables Debug() output
// - colors: enables/disables ANSI color emission
var (
  logger = log.New(os.Stderr, "", log.LstdFlags)
  debug  atomic.Bool
  colors atomic.Bool
)

// init runs automatically when the package is imported.
// It establishes sane defaults with zero configuration.
func init() {
  // Enable debug logging when DEBUG=1 is present.
  debug.Store(os.Getenv("DEBUG") == "1")

  // Decide once whether ANSI colors should be used.
  colors.Store(shouldUseColors())
}

// Init optionally overrides debug mode at runtime.
// Passing nil leaves the environment-based default untouched.
func Init(debugMode *bool) {
  if debugMode != nil {
    debug.Store(*debugMode)
  }
}

// shouldUseColors determines whether ANSI escape codes should be emitted.
// It avoids emitting colors when output is redirected or unsupported.
func shouldUseColors() bool {
  // NO_COLOR is a widely used convention to disable ANSI colors.
  if os.Getenv("NO_COLOR") != "" {
    return false
  }

  // If stderr is not a character device (piped or redirected),
  // do not emit escape codes.
  fi, err := os.Stderr.Stat()
  if err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
    return false
  }

  // Non-Windows systems almost always support ANSI in terminals.
  if runtime.GOOS != "windows" {
    return true
  }

  // Windows: enable colors only in known modern terminals.
  if os.Getenv("WT_SESSION") != "" { // Windows Terminal
    return true
  }
  if os.Getenv("TERM_PROGRAM") != "" { // VS Code, etc.
    return true
  }

  // Fallback heuristic for other environments.
  term := strings.ToLower(os.Getenv("TERM"))
  return term != "" && term != "dumb"
}

// c conditionally returns a color escape sequence.
// When colors are disabled, it returns an empty string.
func c(s string) string {
  if !colors.Load() {
    return ""
  }
  return s
}

// getCallerInfo returns "file:line" for the logging callsite.
// skip controls how many stack frames to skip.
func getCallerInfo(skip int) string {
  _, file, line, ok := runtime.Caller(skip)
  if !ok {
    return "unknown:0"
  }
  filename := filepath.Base(file)
  return c(colorMagenta) + filename + ":" + strconv.Itoa(line) + c(colorReset)
}

// Debug logs a debug-level message.
// Output is suppressed unless debug mode is enabled.
func Debug(format string, v ...any) {
  if !debug.Load() {
    return
  }
  prefix := c(colorCyan) + "[DEBUG] " + c(colorMagenta) + "%s: " + c(colorReset)
  logger.Printf(prefix+format, append([]any{getCallerInfo(2)}, v...)...)
}

// Info logs an informational message.
func Info(format string, v ...any) {
  prefix := c(colorGreen) + "[INFO] " + c(colorMagenta) + "%s: " + c(colorReset)
  logger.Printf(prefix+format, append([]any{getCallerInfo(2)}, v...)...)
}

// Warn logs a warning message.
func Warn(format string, v ...any) {
  prefix := c(colorYellow) + "[WARN] " + c(colorMagenta) + "%s: " + c(colorReset)
  logger.Printf(prefix+format, append([]any{getCallerInfo(2)}, v...)...)
}

// Error logs an error message.
func Error(format string, v ...any) {
  prefix := c(colorRed) + "[ERROR] " + c(colorMagenta) + "%s: " + c(colorReset)
  logger.Printf(prefix+format, append([]any{getCallerInfo(2)}, v...)...)
}

// Fatal logs a fatal error message and terminates the process.
func Fatal(format string, v ...any) {
  prefix := c(colorRed) + "[FATAL] " + c(colorMagenta) + "%s: " + c(colorReset)
  logger.Printf(prefix+format, append([]any{getCallerInfo(2)}, v...)...)
  os.Exit(1)
}
