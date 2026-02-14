package log

import (
	"log/slog"
	"os"
)

func NewLogger() *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return slog.New(h).With(
		"app", hostname,
	)
}
