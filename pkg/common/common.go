package common

import (
	"go4pack/pkg/common/config"
	"go4pack/pkg/common/fs"
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

// InitWithConfig initializes both config and logger
func InitWithConfig(configPath string) error {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Initialize logger with debug level if debug is enabled
	loggerConfig := logger.DefaultConfig()
	if cfg.Debug {
		loggerConfig.Level = "debug"
	}

	return logger.Init(loggerConfig)
}

// Init initializes the application with default settings
func Init() error {
	return InitWithConfig("")
}

// GetLogger returns the global logger instance
func GetLogger() *zerolog.Logger {
	return logger.GetLogger()
}

// GetConfig returns the current configuration
func GetConfig() *config.Config {
	return config.Get()
}

// IsDebug returns whether debug mode is enabled
func IsDebug() bool {
	return config.IsDebug()
}

// GetFileSystem returns a new filesystem instance
func GetFileSystem() (*fs.FileSystem, error) {
	return fs.New()
}

// GetFileSystemWithPath returns a new filesystem instance with custom base path
func GetFileSystemWithPath(basePath string) (*fs.FileSystem, error) {
	return fs.NewWithBasePath(basePath)
}
