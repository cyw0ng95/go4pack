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

	logger.Info().Msg("go4pack is running successfully!")
}
