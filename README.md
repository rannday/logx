# Golog
A simple Go logging library.
## Install

```bash
go get github.com/rannday/golog@v0.1.1
```
## Usage
```go
import (
  "os"

  "github.com/rannday/golog/logx"
)

func main() {
  // Optional: only needed if you want to explicitly override debug mode.
  // Passing nil uses DEBUG=1 from the environment.
  logx.Init(nil)

  logx.Info("starting up: pid=%d", os.Getpid())
  logx.Debug("only prints with DEBUG=1: x=%d", 123)
  logx.Warn("something looks odd")
  logx.Error("something failed")
}
```
Enable debug logging:

```bash
DEBUG=1 ./yourapp
```

Disable color output:

```bash
NO_COLOR=1 ./yourapp
```
## Log Levels

| Level | Enabled by default | Notes |
|------|-------------------|-------|
| Debug | ❌ | Enabled with `DEBUG=1` |
| Info  | ✅ | Normal operation |
| Warn  | ✅ | Non-fatal issues |
| Error | ✅ | Errors that do not exit |
| Fatal | ✅ | Logs and exits with code 1 |