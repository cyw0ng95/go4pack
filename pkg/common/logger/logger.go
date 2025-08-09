package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config holds the logger configuration
type Config struct {
	Level      string `json:"level" yaml:"level"`
	Format     string `json:"format" yaml:"format"` // "json" or "console"
	TimeFormat string `json:"time_format" yaml:"time_format"`
	Output     string `json:"output" yaml:"output"` // "stdout", "stderr", or file path
}

// DefaultConfig returns the default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		Format:     "console",
		TimeFormat: time.RFC3339,
		Output:     "stdout",
	}
}

// Init initializes the global logger with the provided configuration
func Init(config *Config) error {
	// Set log level
	level, err := zerolog.ParseLevel(config.Level)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(level)

	// Set time format
	zerolog.TimeFieldFormat = config.TimeFormat

	// Configure output
	var output io.Writer
	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// Assume it's a file path
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		output = file
	}

	// Configure format
	if config.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: config.TimeFormat,
		}
	}

	// Set global logger
	log.Logger = zerolog.New(output).With().Timestamp().Logger()

	return nil
}

// GetLogger returns a new logger instance
func GetLogger() *zerolog.Logger {
	return &log.Logger
}

// WithComponent returns a logger with a component field
func WithComponent(component string) *zerolog.Logger {
	logger := log.Logger.With().Str("component", component).Logger()
	return &logger
}

// WithFields returns a logger with additional fields
func WithFields(fields map[string]interface{}) *zerolog.Logger {
	ctx := log.Logger.With()
	for key, value := range fields {
		ctx = ctx.Interface(key, value)
	}
	logger := ctx.Logger()
	return &logger
}
