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
		FilePath:  "logs/json.log",
		JSONFile:  true,
		AddSource: true,
	}); err != nil {
		panic(err)
	}

	logx.Info("server started", "port", 8080)

	api := logx.With("component", "api")
	api.Info("request received",
		"method", "GET",
		"path", "/devices",
	)

	logx.Error("authentication failed", "user", "admin")
}
