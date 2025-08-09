package logger

import (
	"go-wiki-app/internal/config"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

// Logger defines a standard interface for logging.
type Logger interface {
	Info(msg string)
	Warn(msg string)
	Error(err error, msg string)
	Fatal(err error, msg string)
	With(fields map[string]interface{}) Logger
}

// zerologLogger is an implementation of the Logger interface using zerolog.
type zerologLogger struct {
	logger zerolog.Logger
}

// New creates a new Logger instance based on the provided configuration.
func New(cfg config.LogConfig) Logger {
	var output io.Writer = os.Stdout
	if strings.ToLower(cfg.Format) == "console" {
		output = zerolog.ConsoleWriter{Out: os.Stdout}
	}

	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil {
		level = zerolog.InfoLevel
		// Log the error using a temporary basic logger
		tmpLogger := zerolog.New(os.Stderr).With().Timestamp().Logger()
		tmpLogger.Warn().Msgf("Invalid log level '%s', defaulting to 'info'", cfg.Level)
	}

	logger := zerolog.New(output).Level(level).With().Timestamp().Logger()

	return &zerologLogger{logger: logger}
}

func (l *zerologLogger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

func (l *zerologLogger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

func (l *zerologLogger) Error(err error, msg string) {
	l.logger.Error().Err(err).Msg(msg)
}

func (l *zerologLogger) Fatal(err error, msg string) {
	l.logger.Fatal().Err(err).Msg(msg)
}

// With creates a sub-logger with additional fields.
func (l *zerologLogger) With(fields map[string]interface{}) Logger {
	subLogger := l.logger.With().Fields(fields).Logger()
	return &zerologLogger{logger: subLogger}
}
