package server

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// LogConfig holds settings for logger
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// GetHandler returns a slog handler based on the input. Valid values are "json", otherwise default text is used
func (c LogConfig) getHandler(writer io.Writer) slog.Handler {
	opts := &slog.HandlerOptions{Level: c.GetLogLevel()}
	switch c.Format {
	case "json":
		return slog.NewJSONHandler(writer, opts)
	default:
		return slog.NewTextHandler(writer, opts)
	}
}

func (c LogConfig) NewLogger() *slog.Logger {
	return c.NewLoggerWithWriter(os.Stdout)
}

func (c LogConfig) NewLoggerWithWriter(writer io.Writer) *slog.Logger {
	return slog.New(c.getHandler(writer))
}

// GetLogLevel returns the Level based on parsed string. Defaults to Info instead of error
func (c LogConfig) GetLogLevel() slog.Level {
	switch strings.ToLower(c.Level) {
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
