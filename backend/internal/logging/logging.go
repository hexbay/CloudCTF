package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"cloudctf/backend/internal/config"
)

// NewLogger creates a logger that writes human-readable text logs to stdout
// and structured JSON logs to a file when LOG_FILE is configured.
func NewLogger(cfg config.Config) *slog.Logger {
	consoleHandler := newConsoleTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})

	var fileHandler slog.Handler
	if cfg.LogFile != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.LogFile), 0o755); err != nil {
			// Fall back to stdout-only logging if the log directory cannot be created.
			return slog.New(consoleHandler)
		}

		file, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return slog.New(consoleHandler)
		}

		fileHandler = slog.NewJSONHandler(file, &slog.HandlerOptions{
			Level: cfg.LogLevel,
		})
		return slog.New(multiHandler{console: consoleHandler, file: fileHandler})
	}

	return slog.New(consoleHandler)
}

type consoleTextHandler struct {
	writer io.Writer
	level  slog.Leveler
	attrs  []slog.Attr
}

func newConsoleTextHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	level := slog.LevelVar{}
	level.Set(slog.LevelInfo)
	if opts != nil && opts.Level != nil {
		level.Set(opts.Level.Level())
	}
	return consoleTextHandler{writer: w, level: &level}
}

func (h consoleTextHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h consoleTextHandler) Handle(_ context.Context, record slog.Record) error {
	var b strings.Builder
	b.WriteString(record.Time.Format("2006-01-02 15:04:05"))
	b.WriteString(" ")
	b.WriteString(colorizeLevel(record.Level))
	b.WriteString(" ")
	b.WriteString(record.Message)

	attrs := append([]slog.Attr{}, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	for _, attr := range attrs {
		b.WriteString(" ")
		b.WriteString(attr.Key)
		b.WriteString("=")
		b.WriteString(fmt.Sprint(attr.Value.Any()))
	}
	b.WriteString("\n")

	_, err := io.WriteString(h.writer, b.String())
	return err
}

func (h consoleTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	combined := append([]slog.Attr{}, h.attrs...)
	combined = append(combined, attrs...)
	return consoleTextHandler{writer: h.writer, level: h.level, attrs: combined}
}

func (h consoleTextHandler) WithGroup(name string) slog.Handler {
	return h
}

func colorizeLevel(level slog.Level) string {
	const (
		reset  = "\033[0m"
		red    = "\033[31m"
		green  = "\033[32m"
		yellow = "\033[33m"
		cyan   = "\033[36m"
	)

	switch {
	case level >= slog.LevelError:
		return red + strings.ToUpper(level.String()) + reset
	case level >= slog.LevelWarn:
		return yellow + strings.ToUpper(level.String()) + reset
	case level >= slog.LevelDebug:
		return cyan + strings.ToUpper(level.String()) + reset
	default:
		return green + strings.ToUpper(level.String()) + reset
	}
}

// multiHandler writes the same record to both a text console handler and a JSON file handler.
type multiHandler struct {
	console slog.Handler
	file    slog.Handler
}

func (h multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.console.Enabled(ctx, level) || h.file.Enabled(ctx, level)
}

func (h multiHandler) Handle(ctx context.Context, record slog.Record) error {
	if err := h.console.Handle(ctx, record); err != nil {
		return err
	}
	if h.file != nil {
		return h.file.Handle(ctx, record)
	}
	return nil
}

func (h multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return multiHandler{
		console: h.console.WithAttrs(attrs),
		file:    h.file.WithAttrs(attrs),
	}
}

func (h multiHandler) WithGroup(name string) slog.Handler {
	return multiHandler{
		console: h.console.WithGroup(name),
		file:    h.file.WithGroup(name),
	}
}
