package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	EnvironmentVariableLogLevel  = "LOG_LEVEL"
	EnvironmentVariableLogFormat = "LOG_FORMAT"
	FormatText                   = "text"
	FormatJSON                   = "json"
	tokenVisiblePrefixLength     = 6
	tokenMaskFixedLength         = 16
)

func NewLoggerFromEnvironment() *slog.Logger {
	level := parseLogLevel(os.Getenv(EnvironmentVariableLogLevel))
	format := parseLogFormat(os.Getenv(EnvironmentVariableLogFormat))

	return newLogger(os.Stdout, level, format)
}

func NewTestLogger(writer io.Writer) *slog.Logger {
	return newLogger(writer, slog.LevelDebug, FormatText)
}

func WithComponent(parentLogger *slog.Logger, componentName string) *slog.Logger {
	return parentLogger.With("component", componentName)
}

func MaskToken(token string) string {
	if len(token) == 0 {
		return "<empty>"
	}

	visiblePrefix := token
	if len(token) > tokenVisiblePrefixLength {
		visiblePrefix = token[:tokenVisiblePrefixLength]
	}

	return visiblePrefix + strings.Repeat("*", tokenMaskFixedLength)
}

func parseLogLevel(rawLevel string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(rawLevel)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info", "":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

func parseLogFormat(rawFormat string) string {
	switch strings.ToLower(strings.TrimSpace(rawFormat)) {
	case FormatJSON:
		return FormatJSON
	default:
		return FormatText
	}
}

func newLogger(writer io.Writer, level slog.Level, format string) *slog.Logger {
	handlerOptions := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if format == FormatJSON {
		handler = slog.NewJSONHandler(writer, handlerOptions)
	} else {
		handler = slog.NewTextHandler(writer, handlerOptions)
	}

	return slog.New(handler)
}

func Fatal(logger *slog.Logger, message string, args ...any) {
	logger.Error(message, args...)
	os.Exit(1)
}
