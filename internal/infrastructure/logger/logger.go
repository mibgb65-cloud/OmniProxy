// Package logger provides OmniProxy's unified structured logging.
//
// All application logs go through log/slog, pass through a central Redactor
// and are written to three sinks: the console, a rotating file and an
// in-memory UI buffer streamed to the desktop UI (技术架构选择.md §17-§18).
package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"omniproxy/internal/platform/profile"
)

// Options configures the application logger.
type Options struct {
	// Profile selects sensible defaults (development logs at debug level).
	Profile profile.Kind
	// LogDir is the directory for the rotating log file. Empty disables file
	// and UI-buffer sinks (console only).
	LogDir string
	// Level overrides the log level: debug, info, warn or error.
	// Empty selects the profile default, then OMNIPROXY_LOG_LEVEL.
	Level string
	// Console receives human-readable logs; defaults to os.Stderr.
	Console io.Writer
	// UIBufferCapacity is the number of lines kept for the UI; 0 = default.
	UIBufferCapacity int
}

// Setup builds the application logger and installs it as the slog default.
// The returned Closer flushes and closes the log file and must be called on
// shutdown.
func Setup(opts Options) (*slog.Logger, io.Closer, error) {
	level := resolveLevel(opts.Level, opts.Profile)
	console := opts.Console
	if console == nil {
		console = os.Stderr
	}

	capacity := opts.UIBufferCapacity
	if capacity <= 0 {
		capacity = 512
	}
	uiBuffer := NewUIBuffer(capacity)
	redactor := NewRedactor()

	sinks := []slog.Handler{
		newRedactingHandler(slog.NewTextHandler(console, &slog.HandlerOptions{Level: level}), redactor),
	}

	var closer io.Closer = nopCloser{}
	if opts.LogDir != "" {
		rot, err := NewRotatingFile(filepath.Join(opts.LogDir, "omniproxy.log"))
		if err != nil {
			return nil, nil, err
		}
		sinks = append(sinks,
			newRedactingHandler(slog.NewJSONHandler(rot, &slog.HandlerOptions{Level: level}), redactor),
			newRedactingHandler(slog.NewTextHandler(&lineWriter{buf: uiBuffer}, &slog.HandlerOptions{Level: level}), redactor),
		)
		closer = rot
	}

	log := slog.New(fanoutHandler{handlers: sinks})
	slog.SetDefault(log)
	return log, closer, nil
}

func resolveLevel(level string, kind profile.Kind) slog.Level {
	if level == "" {
		level = os.Getenv("OMNIPROXY_LOG_LEVEL")
	}
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "":
		if kind == profile.Development {
			return slog.LevelDebug
		}
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

type nopCloser struct{}

func (nopCloser) Close() error { return nil }
