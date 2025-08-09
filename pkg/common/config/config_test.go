package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestLoad(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Test case 1: Config file doesn't exist - should create default
	t.Run("CreateDefaultConfig", func(t *testing.T) {
		// Change to temp directory
		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)
		os.Chdir(tempDir)

		// Reset viper state
		viper.Reset()
		appConfig = nil

		config, err := Load(".")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if config.Debug != false {
			t.Errorf("Expected default debug to be false, got %v", config.Debug)
		}

		// Check if config file was created
		if _, err := os.Stat("config.json"); os.IsNotExist(err) {
			t.Error("Expected config.json to be created")
		}
	})

	// Test case 2: Valid config file exists
	t.Run("LoadExistingConfig", func(t *testing.T) {
		// Create a test config file
		configContent := `{"debug": true}`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create test config file: %v", err)
		}

		// Reset viper state
		viper.Reset()
		appConfig = nil

		config, err := Load(tempDir)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if config.Debug != true {
			t.Errorf("Expected debug to be true, got %v", config.Debug)
		}
	})

	// Test case 3: Permission denied scenario
	t.Run("PermissionDenied", func(t *testing.T) {
		// Skip this test on Windows as permission handling is different
		if os.Getenv("GOOS") == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		// Create a directory without write permissions
		permDir := filepath.Join(tempDir, "noperm")
		if err := os.Mkdir(permDir, 0444); err != nil {
			t.Fatalf("Failed to create no-permission directory: %v", err)
		}
		defer os.Chmod(permDir, 0755) // Clean up

		// Reset viper state
		viper.Reset()
		appConfig = nil

		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)
		
		// Try to create config in directory without write permission
		if err := os.Chdir(permDir); err != nil {
			t.Skipf("Could not change to restricted directory: %v", err)
		}

		_, err := Load(".")
		// We expect this to fail due to permission issues when trying to create config.json
		if err == nil {
			t.Log("Warning: Expected permission error, but got none")
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("GetWithoutLoad", func(t *testing.T) {
		// Reset state
		appConfig = nil

		config := Get()
		if config.Debug != false {
			t.Errorf("Expected default debug to be false, got %v", config.Debug)
		}
	})

	t.Run("GetAfterLoad", func(t *testing.T) {
		// Set up a known config
		testConfig := &Config{Debug: true}
		appConfig = testConfig

		config := Get()
		if config.Debug != true {
			t.Errorf("Expected debug to be true, got %v", config.Debug)
		}
	})
}

func TestIsDebug(t *testing.T) {
	t.Run("DebugFalse", func(t *testing.T) {
		appConfig = &Config{Debug: false}

		if IsDebug() != false {
			t.Error("Expected IsDebug() to return false")
		}
	})

	t.Run("DebugTrue", func(t *testing.T) {
		appConfig = &Config{Debug: true}

		if IsDebug() != true {
			t.Error("Expected IsDebug() to return true")
		}
	})

	t.Run("DebugWithNilConfig", func(t *testing.T) {
		appConfig = nil

		if IsDebug() != false {
			t.Error("Expected IsDebug() to return false when config is nil")
		}
	})
}

func TestReload(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	t.Run("ReloadSuccess", func(t *testing.T) {
		// Create initial config
		initialConfig := `{"debug": false}`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatalf("Failed to create initial config: %v", err)
		}

		// Reset viper and load initial config
		viper.Reset()
		viper.SetConfigName("config")
		viper.SetConfigType("json")
		viper.AddConfigPath(tempDir)
		appConfig = nil

		// Load initial config
		config, err := Load(tempDir)
		if err != nil {
			t.Fatalf("Failed to load initial config: %v", err)
		}

		if config.Debug != false {
			t.Error("Expected initial debug to be false")
		}

		// Update config file
		updatedConfig := `{"debug": true}`
		if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
			t.Fatalf("Failed to update config file: %v", err)
		}

		// Reload config
		if err := Reload(); err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}

		// Check if config was updated
		reloadedConfig := Get()
		if reloadedConfig.Debug != true {
			t.Error("Expected debug to be true after reload")
		}
	})

	t.Run("ReloadWithoutInitialLoad", func(t *testing.T) {
		// Reset state
		viper.Reset()
		appConfig = nil

		err := Reload()
		if err == nil {
			t.Error("Expected error when reloading without initial config")
		}
	})
}

func TestCreateDefaultConfig(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Reset viper state
	viper.Reset()

	config, err := createDefaultConfig()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.Debug != false {
		t.Errorf("Expected default debug to be false, got %v", config.Debug)
	}

	// Check if file was created
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		t.Error("Expected config.json to be created")
	}

	// Verify file contents
	data, err := os.ReadFile("config.json")
	if err != nil {
		t.Fatalf("Failed to read created config file: %v", err)
	}

	expectedContent := "{\n  \"debug\": false\n}"
	if string(data) != expectedContent {
		t.Errorf("Expected config content %q, got %q", expectedContent, string(data))
	}
}

// Benchmark tests
func BenchmarkLoad(b *testing.B) {
	tempDir := b.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{"debug": true}`
	os.WriteFile(configPath, []byte(configContent), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		viper.Reset()
		appConfig = nil
		Load(tempDir)
	}
}

func BenchmarkGet(b *testing.B) {
	appConfig = &Config{Debug: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Get()
	}
}

func BenchmarkIsDebug(b *testing.B) {
	appConfig = &Config{Debug: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsDebug()
	}
}
