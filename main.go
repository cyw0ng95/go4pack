package main

import (
	"go4pack/pkg/common"
)

func main() {
	// Initialize config and logger
	if err := common.Init(); err != nil {
		panic(err)
	}

	// Get the logger
	logger := common.GetLogger()

	logger.Info().Msg("Starting go4pack application")
	logger.Debug().Str("version", "1.0.0").Msg("Application info")

	// Show config status
	if common.IsDebug() {
		logger.Debug().Msg("Debug mode is enabled")
	} else {
		logger.Info().Msg("Debug mode is disabled")
	}

	// Initialize and test filesystem
	fsys, err := common.GetFileSystem()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize filesystem")
		panic(err)
	}

	logger.Info().Str("runtime_path", fsys.GetRuntimePath()).Msg("Runtime directory initialized")
	logger.Info().Str("objects_path", fsys.GetObjectsPath()).Msg("Objects directory initialized")

	// Test writing and reading an object
	testData := []byte("Hello, filesystem!")
	if err := fsys.WriteObject("test.txt", testData); err != nil {
		logger.Error().Err(err).Msg("Failed to write test object")
	} else {
		logger.Info().Msg("Test object written successfully")

		// Read it back
		if data, err := fsys.ReadObject("test.txt"); err != nil {
			logger.Error().Err(err).Msg("Failed to read test object")
		} else {
			logger.Info().Str("content", string(data)).Msg("Test object read successfully")
		}
	}

	logger.Info().Msg("go4pack is running successfully!")
}
