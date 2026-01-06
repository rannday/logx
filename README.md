# Golog
A simple Go logging library for my projects  
---
`go get github.com/rannday/golog@v0.1.0`
```go
import "github.com/rannday/golog/logx"

func main() {
  logx.Init(nil) // uses DEBUG=1 env
  logx.Info("starting up: pid=%d", os.Getpid())
  logx.Debug("only prints with DEBUG=1: x=%d", 123)
}
```