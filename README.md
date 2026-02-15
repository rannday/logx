# Logx
Structured logging built on Go's `log/slog`.
-   Text or JSON output
-   Stack traces by level
-   Runtime level changes
-   Redaction support
-   HTTP server middleware
-   HTTP client transport
---
## Install
``` bash
go get github.com/rannday/logx
```
## Quick Start

``` go
package main

import (
    "log/slog"

    "github.com/rannday/logx"
)

func main() {
    logx.Init(logx.Config{
        Level:           slog.LevelInfo,
        Console:         true,
        AddSource:       false,
        StacktraceLevel: slog.LevelError,
    })

    logx.Info("starting", "version", "1.0.0")
    logx.Warn("cache miss", "key", "user:42")
    logx.Error("operation failed", "id", 123)
}
```
## Configuration
``` go
logx.Init(logx.Config{
    Level:           slog.LevelDebug,
    Console:         true,
    FilePath:        "app.log",
    JSONFile:        false,
    AddSource:       true,
    StacktraceLevel: slog.LevelError,
})
```
## Runtime Level Changes
``` go
logx.SetLevel(slog.LevelDebug)
```
## Structured Logging
``` go
logx.Info("user login",
    "user", "admin",
    "ip", "10.0.0.5",
)
```
## Error Helpers
``` go
err := doSomething()
logx.ErrorErr("operation failed", err, "device", "fw1")
```
Context version:
``` go
logx.ErrorErrContext(ctx, "commit failed", err)
```
## Custom Structured Errors
``` go
type APIError struct {
    Status int
    Code   string
}

func (e APIError) LogAttrs() []slog.Attr {
    return []slog.Attr{
        slog.Int("status", e.Status),
        slog.String("code", e.Code),
    }
}
```
Usage:

``` go
logx.ErrorErr("api failure", apiErr)
```
## Context Helpers
``` go
ctx := logx.WithRequestID(ctx, "abc123")
id, ok := logx.RequestID(ctx)
```
## Timing Helpers
``` go
done := logx.Timed(ctx, "panos commit", "device", "fw1")
defer done()
```
Custom level:
``` go
done := logx.TimedLevel(
    logx.Logger(),
    slog.LevelDebug,
    ctx,
    "panos commit",
    "device", "fw1",
)
defer done()
```
## Color Output
-   Enabled automatically for TTY
-   Disabled when piped
-   Disabled if `NO_COLOR` is set

## Fatal
``` go
logx.Fatal("unrecoverable error")
```
Logs at error level and exits with status code `1`.
## Examples
``` bash
go run ./examples/text
go run ./examples/json
```
## Testing
``` bash
go test -race ./...
```
# Middleware
## HTTP Integration
HTTP utilities live in the `httpx` subpackage.
``` go
import "github.com/rannday/logx/httpx"
```
## Server Middleware
``` go
handler := httpx.HTTPMiddleware(router)
```
## HTTP Client Transport
``` go
client := &http.Client{
    Transport: httpx.Transport(nil),
}
```
## Redaction
``` go
logx.SetRedactedKeys("password", "apikey", "token")
```
Example output:

    password=REDACTED

Query parameters like `apikey`, `password`, `token`, and `key` are
automatically redacted in URLs.
