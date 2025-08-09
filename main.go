package main

import (
	"go4pack/pkg/common"
)

func main() {
	// Initialize logger with default config
	if err := common.InitLogger(); err != nil {
		panic(err)
	}

	// Get the logger
	logger := common.GetLogger()

	logger.Info().Msg("Starting go4pack application")
	logger.Debug().Str("version", "1.0.0").Msg("Application info")
	logger.Info().Msg("go4pack is running successfully!")
}
