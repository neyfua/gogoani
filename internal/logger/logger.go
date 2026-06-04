package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

var Log *slog.Logger

func init() {
	Log = slog.New(slog.NewTextHandler(io.Discard, nil))
}

func Init(debug bool) error {
	var writer io.Writer = io.Discard

	if debug {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		logDir := filepath.Join(homeDir, ".local", "state", "gogoani")
		if err := os.MkdirAll(logDir, 0700); err != nil {
			return err
		}

		logFile := filepath.Join(logDir, "gogoani.log")
		f, err := os.OpenFile(filepath.Clean(logFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return err
		}

		writer = f
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	Log = slog.New(slog.NewTextHandler(writer, opts))
	return nil
}
