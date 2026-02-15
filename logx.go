// Package logx provides a simple wrapper around slog with some additional features:
package logx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	logger   *slog.Logger
	levelVar = new(slog.LevelVar)
	once     sync.Once
	useColor bool
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorGray   = "\033[90m"
)

type Config struct {
	Level           slog.Level
	Console         bool
	FilePath        string
	JSONFile        bool
	AddSource       bool
	StacktraceLevel slog.Level
	// File rotation settings
	FileMaxSizeBytes int // rotate when file exceeds this many bytes (0 = disabled)
	FileMaxBackups   int // number of rotated files to keep
	// ConsoleJSON outputs console logs as JSON when true
	ConsoleJSON bool
	// FileWriter can be provided to control file output (overrides FilePath)
	FileWriter io.WriteCloser
}

// Init initializes the global logger according to the provided Config.
// It is safe to call multiple times; initialization is performed once.
func Init(cfg Config) error {
	var initErr error

	once.Do(func() {
		levelVar.Set(cfg.Level)

		opts := &slog.HandlerOptions{
			Level:     levelVar,
			AddSource: cfg.AddSource,
		}

		var handlers []slog.Handler

		if cfg.Console {
			useColor = detectColor()

			var writer io.Writer = os.Stderr
			if useColor {
				writer = &colorWriter{w: os.Stderr}
			}

			if cfg.ConsoleJSON {
				handlers = append(handlers,
					slog.NewJSONHandler(writer, opts),
				)
			} else {
				handlers = append(handlers,
					slog.NewTextHandler(writer, opts),
				)
			}
		}

		var fileWriter io.WriteCloser
		if cfg.FileWriter != nil {
			fileWriter = cfg.FileWriter
		} else if cfg.FilePath != "" {
			if cfg.FileMaxSizeBytes > 0 {
				r, err := newFileRotator(cfg.FilePath, cfg.FileMaxSizeBytes, cfg.FileMaxBackups)
				if err != nil {
					initErr = err
				} else {
					fileWriter = r
				}
			} else {
				f, err := os.OpenFile(
					cfg.FilePath,
					os.O_CREATE|os.O_APPEND|os.O_WRONLY,
					0o644,
				)
				if err != nil {
					initErr = err
				} else {
					fileWriter = f
				}
			}
		}

		if fileWriter != nil {
			if cfg.JSONFile {
				handlers = append(handlers,
					slog.NewJSONHandler(fileWriter, opts),
				)
			} else {
				handlers = append(handlers,
					slog.NewTextHandler(fileWriter, opts),
				)
			}
		}

		if len(handlers) == 0 {
			handlers = append(handlers,
				slog.NewTextHandler(os.Stderr, opts),
			)
		}

		var handler slog.Handler
		if len(handlers) == 1 {
			handler = handlers[0]
		} else {
			handler = newMultiHandler(handlers...)
		}

		handler = newStackHandler(handler, cfg.StacktraceLevel)
		handler = newRedactionHandler(handler)

		logger = slog.New(handler)
		slog.SetDefault(logger)
	})

	return initErr
}

// Reset clears logger state.
// Intended for testing only.
func Reset() {
	logger = nil
	useColor = false
	levelVar = new(slog.LevelVar)
	once = sync.Once{}
	redactedKeys = map[string]struct{}{}
}

// SetLogger replaces the global logger.
// Intended for testing only.
func SetLogger(l *slog.Logger) {
	logger = l
	slog.SetDefault(l)
}

func SetLevel(level slog.Level) {
	levelVar.Set(level)
}

func Logger() *slog.Logger {
	if logger == nil {
		_ = Init(Config{
			Level:   slog.LevelInfo,
			Console: true,
		})
	}
	return logger
}

func Debug(msg string, args ...any) {
	Logger().Debug(msg, args...)
}

func Info(msg string, args ...any) {
	Logger().Info(msg, args...)
}

func Warn(msg string, args ...any) {
	Logger().Warn(msg, args...)
}

func Error(msg string, args ...any) {
	Logger().Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	Logger().Error(msg, args...)
	os.Exit(1)
}

type Loggable interface {
	LogAttrs() []slog.Attr
}

func ErrorErr(msg string, err error, args ...any) {
	if err == nil {
		Logger().Error(msg, args...)
		return
	}

	fields := make([]any, 0, len(args)+4)
	fields = append(fields, args...)
	fields = append(fields,
		"error", err,
		"error_type", fmt.Sprintf("%T", err),
	)

	// Structured error support
	if le, ok := err.(Loggable); ok {
		for _, attr := range le.LogAttrs() {
			fields = append(fields, attr.Key, attr.Value.Any())
		}
	}

	Logger().Error(msg, fields...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	Logger().DebugContext(ctx, msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	Logger().InfoContext(ctx, msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	Logger().WarnContext(ctx, msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	Logger().ErrorContext(ctx, msg, args...)
}

func ErrorErrContext(ctx context.Context, msg string, err error, args ...any) {
	if err == nil {
		Logger().ErrorContext(ctx, msg, args...)
		return
	}

	fields := make([]any, 0, len(args)+4)
	fields = append(fields, args...)
	fields = append(fields,
		"error", err,
		"error_type", fmt.Sprintf("%T", err),
	)

	if le, ok := err.(Loggable); ok {
		for _, attr := range le.LogAttrs() {
			fields = append(fields, attr.Key, attr.Value.Any())
		}
	}

	Logger().ErrorContext(ctx, msg, fields...)
}

func With(args ...any) *slog.Logger {
	return Logger().With(args...)
}

func WithGroup(name string) *slog.Logger {
	return Logger().WithGroup(name)
}

type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(h ...slog.Handler) slog.Handler {
	return &multiHandler{handlers: h}
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range m.handlers {
		if err := h.Handle(ctx, r); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, 0, len(m.handlers))
	for _, h := range m.handlers {
		next = append(next, h.WithAttrs(attrs))
	}
	return newMultiHandler(next...)
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, 0, len(m.handlers))
	for _, h := range m.handlers {
		next = append(next, h.WithGroup(name))
	}
	return newMultiHandler(next...)
}

// Timed uses the default logger.
func Timed(ctx context.Context, msg string, args ...any) func(extra ...any) {
	return TimedLevel(Logger(), slog.LevelInfo, ctx, msg, args...)
}

// TimedWith uses a provided logger (supports With(), WithGroup(), etc.)
func TimedWith(l *slog.Logger, ctx context.Context, msg string, args ...any) func(extra ...any) {
	start := time.Now()
	startMsg := msg + " started"
	doneMsg := msg + " completed"

	l.InfoContext(ctx, startMsg, args...)

	return func(extra ...any) {
		duration := time.Since(start)

		fields := make([]any, 0, len(args)+len(extra)+2)
		fields = append(fields, args...)
		fields = append(fields, extra...)
		fields = append(fields, "duration", duration)

		l.InfoContext(ctx, doneMsg, fields...)
	}
}

func TimedLevel(
	l *slog.Logger,
	level slog.Level,
	ctx context.Context,
	msg string,
	args ...any,
) func(extra ...any) {
	start := time.Now()
	startMsg := msg + " started"
	doneMsg := msg + " completed"

	l.Log(ctx, level, startMsg, args...)

	return func(extra ...any) {
		duration := time.Since(start)

		fields := make([]any, 0, len(args)+len(extra)+2)
		fields = append(fields, args...)
		fields = append(fields, extra...)
		fields = append(fields, "duration", duration)

		l.Log(ctx, level, doneMsg, fields...)
	}
}

type colorWriter struct {
	w io.Writer
}

func (cw *colorWriter) Write(p []byte) (int, error) {
	var (
		levelTag []byte
		colored  []byte
	)

	switch {
	case bytes.Contains(p, []byte("level=ERROR")):
		levelTag = []byte("level=ERROR")
		colored = []byte(colorRed + "level=ERROR" + colorReset)
	case bytes.Contains(p, []byte("level=WARN")):
		levelTag = []byte("level=WARN")
		colored = []byte(colorYellow + "level=WARN" + colorReset)
	case bytes.Contains(p, []byte("level=INFO")):
		levelTag = []byte("level=INFO")
		colored = []byte(colorGreen + "level=INFO" + colorReset)
	case bytes.Contains(p, []byte("level=DEBUG")):
		levelTag = []byte("level=DEBUG")
		colored = []byte(colorGray + "level=DEBUG" + colorReset)
	default:
		return cw.w.Write(p)
	}

	i := bytes.Index(p, levelTag)
	if i < 0 {
		return cw.w.Write(p)
	}

	out := make([]byte, 0, len(p)+len(colored)-len(levelTag))
	out = append(out, p[:i]...)
	out = append(out, colored...)
	out = append(out, p[i+len(levelTag):]...)
	return cw.w.Write(out)
}

func detectColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}

	if (fi.Mode() & os.ModeCharDevice) == 0 {
		return false
	}

	if runtime.GOOS != "windows" {
		return true
	}

	if os.Getenv("WT_SESSION") != "" {
		return true
	}
	if os.Getenv("TERM_PROGRAM") != "" {
		return true
	}
	if os.Getenv("ANSICON") != "" {
		return true
	}

	return false
}
