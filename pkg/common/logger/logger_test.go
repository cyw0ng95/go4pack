package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Level != "info" {
		t.Errorf("Expected default level 'info', got %s", config.Level)
	}

	if config.Format != "console" {
		t.Errorf("Expected default format 'console', got %s", config.Format)
	}

	if config.TimeFormat != time.RFC3339 {
		t.Errorf("Expected default time format %s, got %s", time.RFC3339, config.TimeFormat)
	}

	if config.Output != "stdout" {
		t.Errorf("Expected default output 'stdout', got %s", config.Output)
	}
}

func TestInit(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		config := &Config{
			Level:      "debug",
			Format:     "json",
			TimeFormat: time.RFC3339,
			Output:     "stdout",
		}

		err := Init(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Test that the global level was set
		if zerolog.GlobalLevel() != zerolog.DebugLevel {
			t.Errorf("Expected global level to be debug, got %s", zerolog.GlobalLevel())
		}

		// Test that time format was set
		if zerolog.TimeFieldFormat != time.RFC3339 {
			t.Errorf("Expected time format %s, got %s", time.RFC3339, zerolog.TimeFieldFormat)
		}
	})

	t.Run("InvalidLevel", func(t *testing.T) {
		config := &Config{
			Level:      "invalid",
			Format:     "json",
			TimeFormat: time.RFC3339,
			Output:     "stdout",
		}

		err := Init(config)
		if err == nil {
			t.Error("Expected error for invalid log level")
		}
	})

	t.Run("FileOutput", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "test.log")

		config := &Config{
			Level:      "info",
			Format:     "json",
			TimeFormat: time.RFC3339,
			Output:     logFile,
		}

		err := Init(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Test that we can write to the logger
		logger := GetLogger()
		logger.Info().Msg("Test message")

		// Check if file was created and has content
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			t.Error("Expected log file to be created")
		}

		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		if !strings.Contains(string(content), "Test message") {
			t.Error("Expected log file to contain test message")
		}
	})

	t.Run("InvalidFileOutput", func(t *testing.T) {
		config := &Config{
			Level:      "info",
			Format:     "json",
			TimeFormat: time.RFC3339,
			Output:     "/invalid/path/that/does/not/exist/test.log",
		}

		err := Init(config)
		if err == nil {
			t.Error("Expected error for invalid file path")
		}
	})

	t.Run("ConsoleFormat", func(t *testing.T) {
		var buf bytes.Buffer

		config := &Config{
			Level:      "info",
			Format:     "console",
			TimeFormat: "15:04:05",
			Output:     "stdout",
		}

		// Save original logger
		originalLogger := log.Logger

		err := Init(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Create a test logger that writes to buffer
		testLogger := zerolog.New(&buf).With().Timestamp().Logger()
		log.Logger = testLogger

		// Restore original logger
		defer func() { log.Logger = originalLogger }()

		logger := GetLogger()
		logger.Info().Msg("Console test message")

		output := buf.String()
		if !strings.Contains(output, "Console test message") {
			t.Error("Expected output to contain test message")
		}
	})

	t.Run("StderrOutput", func(t *testing.T) {
		config := &Config{
			Level:      "info",
			Format:     "json",
			TimeFormat: time.RFC3339,
			Output:     "stderr",
		}

		err := Init(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Just verify no error occurred - testing stderr output is complex
	})
}

func TestGetLogger(t *testing.T) {
	// Initialize with a known config
	config := DefaultConfig()
	err := Init(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	logger := GetLogger()
	if logger == nil {
		t.Error("Expected logger instance, got nil")
	}

	// Test that it's the same instance
	logger2 := GetLogger()
	if logger != logger2 {
		t.Error("Expected same logger instance")
	}
}

func TestWithComponent(t *testing.T) {
	config := DefaultConfig()
	err := Init(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	component := "test-component"
	logger := WithComponent(component)

	if logger == nil {
		t.Error("Expected logger instance, got nil")
	}

	// Test that we can use the logger
	var buf bytes.Buffer
	testLogger := logger.Output(&buf)
	testLogger.Info().Msg("Test component message")

	output := buf.String()
	if !strings.Contains(output, component) {
		t.Errorf("Expected output to contain component %s, got %s", component, output)
	}

	if !strings.Contains(output, "Test component message") {
		t.Error("Expected output to contain test message")
	}
}

func TestWithFields(t *testing.T) {
	config := DefaultConfig()
	config.Format = "json" // Use JSON for easier testing
	err := Init(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	fields := map[string]interface{}{
		"user_id":    123,
		"request_id": "abc-def-ghi",
		"action":     "test",
	}

	logger := WithFields(fields)
	if logger == nil {
		t.Error("Expected logger instance, got nil")
	}

	// Test that we can use the logger and fields are included
	var buf bytes.Buffer
	testLogger := logger.Output(&buf)
	testLogger.Info().Msg("Test fields message")

	output := buf.String()

	// Parse JSON to verify fields
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}

	// Check that all fields are present
	for key, expectedValue := range fields {
		if actualValue, exists := logEntry[key]; !exists {
			t.Errorf("Expected field %s to be present in log output", key)
		} else {
			// Handle type conversion for numbers (JSON unmarshaling converts numbers to float64)
			switch key {
			case "user_id":
				if actualFloat, ok := actualValue.(float64); !ok {
					t.Errorf("Expected field %s to be a number, got %T", key, actualValue)
				} else if int(actualFloat) != expectedValue.(int) {
					t.Errorf("Expected field %s to have value %v, got %v", key, expectedValue, int(actualFloat))
				}
			default:
				if actualValue != expectedValue {
					t.Errorf("Expected field %s to have value %v, got %v", key, expectedValue, actualValue)
				}
			}
		}
	}

	if logEntry["message"] != "Test fields message" {
		t.Error("Expected message field to contain test message")
	}
}

func TestLogLevels(t *testing.T) {
	testCases := []struct {
		configLevel string
		logLevel    string
		shouldLog   bool
	}{
		{"debug", "debug", true},
		{"debug", "info", true},
		{"debug", "warn", true},
		{"debug", "error", true},
		{"info", "debug", false},
		{"info", "info", true},
		{"info", "warn", true},
		{"info", "error", true},
		{"warn", "debug", false},
		{"warn", "info", false},
		{"warn", "warn", true},
		{"warn", "error", true},
		{"error", "debug", false},
		{"error", "info", false},
		{"error", "warn", false},
		{"error", "error", true},
	}

	for _, tc := range testCases {
		t.Run(tc.configLevel+"_"+tc.logLevel, func(t *testing.T) {
			config := &Config{
				Level:      tc.configLevel,
				Format:     "json",
				TimeFormat: time.RFC3339,
				Output:     "stdout",
			}

			err := Init(config)
			if err != nil {
				t.Fatalf("Failed to initialize logger: %v", err)
			}

			var buf bytes.Buffer
			logger := GetLogger().Output(&buf)

			// Log message at the specified level
			switch tc.logLevel {
			case "debug":
				logger.Debug().Msg("test message")
			case "info":
				logger.Info().Msg("test message")
			case "warn":
				logger.Warn().Msg("test message")
			case "error":
				logger.Error().Msg("test message")
			}

			output := buf.String()
			hasOutput := len(strings.TrimSpace(output)) > 0

			if tc.shouldLog && !hasOutput {
				t.Errorf("Expected log output for config level %s and log level %s", tc.configLevel, tc.logLevel)
			}

			if !tc.shouldLog && hasOutput {
				t.Errorf("Expected no log output for config level %s and log level %s, got: %s", tc.configLevel, tc.logLevel, output)
			}
		})
	}
}

func TestConfigStructValidation(t *testing.T) {
	// Test that Config struct has expected fields
	config := &Config{}

	// Test that we can set all expected fields
	config.Level = "debug"
	config.Format = "json"
	config.TimeFormat = time.RFC3339
	config.Output = "stdout"

	// Test JSON tags are present (indirect test)
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config to JSON: %v", err)
	}

	var unmarshaled Config
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal config from JSON: %v", err)
	}

	if unmarshaled.Level != config.Level {
		t.Errorf("Expected level %s, got %s", config.Level, unmarshaled.Level)
	}
	if unmarshaled.Format != config.Format {
		t.Errorf("Expected format %s, got %s", config.Format, unmarshaled.Format)
	}
	if unmarshaled.TimeFormat != config.TimeFormat {
		t.Errorf("Expected time format %s, got %s", config.TimeFormat, unmarshaled.TimeFormat)
	}
	if unmarshaled.Output != config.Output {
		t.Errorf("Expected output %s, got %s", config.Output, unmarshaled.Output)
	}
}

// Benchmark tests
func BenchmarkInit(b *testing.B) {
	config := DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Init(config)
	}
}

func BenchmarkGetLogger(b *testing.B) {
	config := DefaultConfig()
	Init(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetLogger()
	}
}

func BenchmarkWithComponent(b *testing.B) {
	config := DefaultConfig()
	Init(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WithComponent("test-component")
	}
}

func BenchmarkWithFields(b *testing.B) {
	config := DefaultConfig()
	Init(config)

	fields := map[string]interface{}{
		"user_id": 123,
		"action":  "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WithFields(fields)
	}
}

func BenchmarkLogMessage(b *testing.B) {
	config := DefaultConfig()
	Init(config)

	logger := GetLogger()
	var buf bytes.Buffer
	testLogger := logger.Output(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testLogger.Info().Msg("Benchmark test message")
		buf.Reset()
	}
}
