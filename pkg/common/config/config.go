package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Debug bool `json:"debug" mapstructure:"debug"`
	// Add more configuration fields here as needed
}

var appConfig *Config

// Load loads the configuration from config.json file
func Load(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("json")

	if configPath != "" {
		viper.AddConfigPath(configPath)
	} else {
		// Default paths to look for config file
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
	}

	// Set defaults
	viper.SetDefault("debug", false)

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		// If config file doesn't exist, create a default one
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return createDefaultConfig()
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	appConfig = &config
	return &config, nil
}

// createDefaultConfig creates a default config.json file if it doesn't exist
func createDefaultConfig() (*Config, error) {
	defaultConfig := &Config{
		Debug: false,
	}

	// Set the default values in viper
	viper.Set("debug", defaultConfig.Debug)

	// Write the default config file
	configFile := filepath.Join(".", "config.json")
	if err := viper.WriteConfigAs(configFile); err != nil {
		return nil, fmt.Errorf("error creating default config file: %w", err)
	}

	appConfig = defaultConfig
	return defaultConfig, nil
}

// Get returns the current configuration
func Get() *Config {
	if appConfig == nil {
		// Return default config if not loaded
		return &Config{
			Debug: false,
		}
	}
	return appConfig
}

// IsDebug returns whether debug mode is enabled
func IsDebug() bool {
	return Get().Debug
}

// Reload reloads the configuration from file
func Reload() error {
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reloading config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("error unmarshaling reloaded config: %w", err)
	}

	appConfig = &config
	return nil
}
