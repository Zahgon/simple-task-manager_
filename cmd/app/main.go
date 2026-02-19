package main

import (
	"os"

	"github.com/rs/zerolog"

	"github.com/SilentPlaces/simple-task-manager/internal/app"
	"github.com/SilentPlaces/simple-task-manager/internal/config"
	"github.com/SilentPlaces/simple-task-manager/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		bootstrap := zerolog.New(os.Stderr).With().Timestamp().Logger()
		bootstrap.Fatal().Err(err).Msg("Failed to load config")
	}

	log := logger.New(cfg.Log.Level)

	application, err := app.New(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize application")
	}

	if err := application.Run(); err != nil {
		log.Fatal().Err(err).Msg("Application error")
	}
}
