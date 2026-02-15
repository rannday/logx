package logx

import "context"

type ctxKey string

const requestIDKey ctxKey = "logx_request_id"

// WithRequestID returns a new context containing a request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
  if ctx == nil {
    ctx = context.Background()
  }
  return context.WithValue(ctx, requestIDKey, id)
}

// RequestID returns the request ID from context, if present.
func RequestID(ctx context.Context) (string, bool) {
  if ctx == nil {
    return "", false
  }

  v := ctx.Value(requestIDKey)
  if v == nil {
    return "", false
  }

  id, ok := v.(string)
  if !ok || id == "" {
    return "", false
  }

  return id, true
}
