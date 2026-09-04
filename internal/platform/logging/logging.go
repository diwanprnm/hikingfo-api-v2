// Package logging owns the process-wide logger. It is a thin wrapper over
// log/slog so every context logs through one configured handler; contexts and
// the platform never construct their own loggers (research.md §4 logging rule).
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Env selects the output format. "development" (default) uses the human-
// readable text handler; "production" emits JSON for structured ingestion.
type Env string

const (
	EnvDevelopment Env = "development"
	EnvProduction  Env = "production"
)

// Setup configures slog as the process default and returns the root logger.
// level accepts slog.Level or a name ("debug", "info", "warn", "error"); an
// unknown name is treated as info. Output goes to w, defaulting to stderr
// (slog's convention) so stdout stays clean for program output.
func Setup(env Env, level string, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stderr
	}
	lvl := parseLevel(level)
	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler
	if env == EnvProduction {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
