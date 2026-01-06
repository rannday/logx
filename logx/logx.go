package logx

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
)

const (
  colorReset       = "\033[0m"
  colorRed         = "\033[31m"       // Fatal
  colorDarkOrange  = "\033[38;5;208m" // Error
  colorGreen       = "\033[32m"       // Info
  colorYellow      = "\033[33m"       // Info message
  colorCyan        = "\033[36m"       // Debug
  colorMagenta     = "\033[35m"       // File names
  colorDarkYellow  = "\033[38;5;214m" // Warn
  colorBrightWhite = "\033[97m"       // Message text
)

var (
  logger = log.New(os.Stderr, "", log.LstdFlags)
  debug  atomic.Bool
)

// Init configures the logger.
// If debugMode is nil, it uses env DEBUG=1.
func Init(debugMode *bool) {
  if debugMode == nil {
    debug.Store(os.Getenv("DEBUG") == "1")
    return
  }
  debug.Store(*debugMode)
}

func SetOutput(w io.Writer) {
  logger.SetOutput(w)
}

func SetFlags(flags int) {
  logger.SetFlags(flags)
}

func getCallerInfo(skip int) string {
  _, file, line, ok := runtime.Caller(skip)
  if !ok {
    return "unknown:0"
  }
  filename := filepath.Base(file)
  return colorMagenta + filename + ":" + strconv.Itoa(line) + colorReset
}

func Debug(format string, v ...any) {
  if !debug.Load() {
    return
  }
  // skip=2: Debug -> your callsite
  prefix := colorCyan + "[DEBUG] " + colorMagenta + "%s: " + colorBrightWhite
  logger.Printf(prefix+format+colorReset, append([]any{getCallerInfo(2)}, v...)...)
}

func Info(format string, v ...any) {
  logger.Printf(colorGreen+"[INFO] "+colorYellow+format+colorReset, v...)
}

func Warn(format string, v ...any) {
  // skip=2: Warn -> your callsite
  prefix := colorDarkYellow + "[WARN] " + colorMagenta + "%s: " + colorBrightWhite
  logger.Printf(prefix+format+colorReset, append([]any{getCallerInfo(2)}, v...)...)
}

func Error(format string, v ...any) {
  // skip=2: Error -> your callsite
  prefix := colorDarkOrange + "[ERROR] " + colorMagenta + "%s: " + colorBrightWhite
  logger.Printf(prefix+format+colorReset, append([]any{getCallerInfo(2)}, v...)...)
}

func Fatal(format string, v ...any) {
  // skip=2: Fatal -> your callsite
  prefix := colorRed + "[FATAL] " + colorMagenta + "%s: " + colorBrightWhite
  logger.Printf(prefix+format+colorReset, append([]any{getCallerInfo(2)}, v...)...)
  os.Exit(1)
}

// Optional: non-format version if you ever want it.
func Printf(format string, v ...any) { logger.Printf(format, v...) }
func Println(v ...any)              { logger.Println(v...) }
func Sprintf(format string, v ...any) string {
  return fmt.Sprintf(format, v...)
}
