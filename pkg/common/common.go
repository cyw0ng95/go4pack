package common

import (
	"go4pack/pkg/common/logger"

	"github.com/rs/zerolog"
)

// InitLogger initializes the logger with default configuration
func InitLogger() error {
	config := logger.DefaultConfig()
	return logger.Init(config)
}

// InitLoggerWithConfig initializes the logger with custom configuration
func InitLoggerWithConfig(config *logger.Config) error {
	return logger.Init(config)
}

// GetLogger returns the global logger instance
func GetLogger() *zerolog.Logger {
	return logger.GetLogger()
}
