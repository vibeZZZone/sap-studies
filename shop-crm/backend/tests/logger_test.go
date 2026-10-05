package tests

import (
	"log/slog"
	"os"
)

// Tests keep the server logs on stderr so a failing 500 shows its cause.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}