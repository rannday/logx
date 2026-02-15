package logx

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
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

var (
	redactedKeys         = map[string]struct{}{}
	redactedKeysMu       sync.RWMutex
	redactedKeysSnapshot atomic.Value // map[string]struct{}
)

func init() {
	redactedKeysSnapshot.Store(map[string]struct{}{})
}

func SetRedactedKeys(keys ...string) {
	redactedKeysMu.Lock()
	defer redactedKeysMu.Unlock()
	for _, k := range keys {
		redactedKeys[strings.ToLower(k)] = struct{}{}
	}
	redactedKeysSnapshot.Store(cloneKeySet(redactedKeys))
}

// AddRedactedKeys appends keys to the redaction set (concurrency-safe).
func AddRedactedKeys(keys ...string) {
	SetRedactedKeys(keys...)
}

// ClearRedactedKeys removes all configured redacted keys.
func ClearRedactedKeys() {
	redactedKeysMu.Lock()
	defer redactedKeysMu.Unlock()
	redactedKeys = map[string]struct{}{}
	redactedKeysSnapshot.Store(map[string]struct{}{})
}

// ListRedactedKeys returns a snapshot of configured redacted keys.
func ListRedactedKeys() []string {
	keys, _ := redactedKeysSnapshot.Load().(map[string]struct{})
	out := make([]string, 0, len(keys))
	for k := range keys {
		out = append(out, k)
	}
	return out
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
	keys, _ := redactedKeysSnapshot.Load().(map[string]struct{})
	if len(keys) == 0 {
		return h.next.Handle(ctx, r)
	}

	nr := r.Clone()

	var attrs []slog.Attr

	nr.Attrs(func(a slog.Attr) bool {
		_, ok := keys[strings.ToLower(a.Key)]
		if ok {
			a.Value = slog.StringValue("REDACTED")
		}
		attrs = append(attrs, a)
		return true
	})

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

func cloneKeySet(src map[string]struct{}) map[string]struct{} {
	dst := make(map[string]struct{}, len(src))
	for k := range src {
		dst[k] = struct{}{}
	}
	return dst
}
