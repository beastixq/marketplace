// Package logging builds an *slog.Logger from config.LoggingConfig.
//
// Dual sink (file + stdout) is supported via io.MultiWriter so the same
// records satisfy both the lab requirement (write all errors and user
// actions to a log file) and a developer-friendly stdout stream.
package logging

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/beastixq/marketplace/internal/config"
)

const (
	logDirPerm  = 0o755
	logFilePerm = 0o644
)

// nopCloser implements io.Closer with a no-op Close, used when no
// resource needs releasing (e.g., stdout-only sink).
type nopCloser struct{}

func (nopCloser) Close() error { return nil }

// New constructs a logger and returns an io.Closer that the caller
// must close on shutdown to flush/release the file sink.
func New(cfg config.LoggingConfig) (*slog.Logger, io.Closer, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, nil, err
	}

	writer, closer, err := openSink(cfg)
	if err != nil {
		return nil, nil, err
	}

	levelVar := new(slog.LevelVar)
	levelVar.Set(level)
	opts := &slog.HandlerOptions{Level: levelVar, AddSource: cfg.AddSource}

	var handler slog.Handler
	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		_ = closer.Close()
		return nil, nil, fmt.Errorf("unsupported logging.format %q", cfg.Format)
	}

	return slog.New(handler), closer, nil
}

func parseLevel(s string) (slog.Level, error) {
	switch s {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown logging.level %q", s)
	}
}

func openSink(cfg config.LoggingConfig) (io.Writer, io.Closer, error) {
	writers := make([]io.Writer, 0, 2)
	if cfg.Console {
		writers = append(writers, os.Stdout)
	}

	var closer io.Closer = nopCloser{}
	if cfg.File != "" {
		f, err := openLogFile(cfg.File)
		if err != nil {
			return nil, nil, err
		}
		writers = append(writers, f)
		closer = f
	}

	if len(writers) == 0 {
		return nil, nil, errors.New("logging: no sinks configured")
	}
	if len(writers) == 1 {
		return writers[0], closer, nil
	}
	return io.MultiWriter(writers...), closer, nil
}

func openLogFile(path string) (*os.File, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, logDirPerm); err != nil {
			return nil, fmt.Errorf("create log dir %q: %w", dir, err)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, logFilePerm)
	if err != nil {
		return nil, fmt.Errorf("open log file %q: %w", path, err)
	}
	return f, nil
}
