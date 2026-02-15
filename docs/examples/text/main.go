package main

import (
	"log/slog"
	"os"

	"github.com/rannday/logx"
)

func main() {
	if err := os.MkdirAll("logs", 0755); err != nil {
		panic(err)
	}

	if err := logx.Init(logx.Config{
		Level:     slog.LevelDebug,
		Console:   true,
		FilePath:  "logs/text.log",
		JSONFile:  false,
		AddSource: true,
	}); err != nil {
		panic(err)
	}

	logx.Debug("debug example", "step", 1)
	logx.Info("server started", "port", 8080)
	logx.Warn("high memory usage", "percent", 87)
	logx.Error("connection failed", "host", "10.0.0.1")

	api := logx.With("component", "api")
	api.Info("request received", "method", "GET", "path", "/devices")

	logx.Info("done")
}
