// Package logger provides file-based structured logging for prompt-agent.
// Logs are written to ~/.promptly/promptly.log using log/slog.
package logger

import (
	"log/slog"
	"os"
	"path/filepath"
)

var l *slog.Logger

// Init opens (or creates) the log file at <dir>/promptly.log and
// configures the global logger. Safe to call multiple times; subsequent
// calls are no-ops. Returns an error only if the file cannot be opened.
func Init(dir string) error {
	if l != nil {
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	logPath := filepath.Join(dir, "promptly.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	l = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	return nil
}

// DefaultLogDir returns ~/.promptly so callers don't need to
// duplicate the path logic.
func DefaultLogDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".promptly")
}

func Info(msg string, args ...any) {
	if l != nil {
		l.Info(msg, args...)
	}
}

func Warn(msg string, args ...any) {
	if l != nil {
		l.Warn(msg, args...)
	}
}

func Error(msg string, args ...any) {
	if l != nil {
		l.Error(msg, args...)
	}
}

func Debug(msg string, args ...any) {
	if l != nil {
		l.Debug(msg, args...)
	}
}
