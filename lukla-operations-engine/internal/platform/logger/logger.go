// Package logger provides structured logging for the application.
// Fields logged: timestamp, level, service, request_id, user_id (when applicable),
// route/method, duration, error code.
// Passwords, tokens, secrets, and sensitive data must never be logged.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New creates a slog.Logger configured according to level and format.
// format: "json" (production) or "text" (development).
func New(level, format string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	if strings.ToLower(format) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
