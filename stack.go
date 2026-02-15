package logx

import (
	"context"
	"log/slog"
	"runtime/debug"
)

type stackHandler struct {
	next  slog.Handler
	level slog.Level
}

func newStackHandler(next slog.Handler, level slog.Level) slog.Handler {
	// If no stack level configured, return original handler
	if level == 0 {
		return next
	}
	return &stackHandler{
		next:  next,
		level: level,
	}
}

func (h *stackHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *stackHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= h.level {
		nr := r.Clone()
		nr.AddAttrs(
			slog.String("stack", string(debug.Stack())),
		)
		return h.next.Handle(ctx, nr)
	}

	return h.next.Handle(ctx, r)
}

func (h *stackHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return newStackHandler(h.next.WithAttrs(attrs), h.level)
}

func (h *stackHandler) WithGroup(name string) slog.Handler {
	return newStackHandler(h.next.WithGroup(name), h.level)
}
