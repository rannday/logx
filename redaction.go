package logx

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
)

func SanitizeURL(u *url.URL) string {
	if u == nil {
		return ""
	}

	clone := *u
	q := clone.Query()

	for k := range q {
		lk := strings.ToLower(k)
		switch lk {
		case "apikey", "password", "token", "key":
			q.Set(k, "REDACTED")
		}
	}

	clone.RawQuery = q.Encode()
	return clone.String()
}

var redactedKeys = map[string]struct{}{}

func SetRedactedKeys(keys ...string) {
	for _, k := range keys {
		redactedKeys[strings.ToLower(k)] = struct{}{}
	}
}

type redactionHandler struct {
	next slog.Handler
}

func newRedactionHandler(next slog.Handler) slog.Handler {
	return &redactionHandler{next: next}
}

func (h *redactionHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *redactionHandler) Handle(ctx context.Context, r slog.Record) error {
	if len(redactedKeys) == 0 {
		return h.next.Handle(ctx, r)
	}

	nr := r.Clone()

	var attrs []slog.Attr

	nr.Attrs(func(a slog.Attr) bool {
		if _, ok := redactedKeys[strings.ToLower(a.Key)]; ok {
			a.Value = slog.StringValue("REDACTED")
		}
		attrs = append(attrs, a)
		return true
	})

	// Rebuild record while preserving metadata
	newRec := slog.NewRecord(
		nr.Time,
		nr.Level,
		nr.Message,
		nr.PC,
	)

	newRec.AddAttrs(attrs...)

	return h.next.Handle(ctx, newRec)
}

func (h *redactionHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return newRedactionHandler(h.next.WithAttrs(attrs))
}

func (h *redactionHandler) WithGroup(name string) slog.Handler {
	return newRedactionHandler(h.next.WithGroup(name))
}
