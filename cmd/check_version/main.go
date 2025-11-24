package main

import (
	"context"
	"flag"
	"os"

	"github.com/hexiosec/asm-cloud-connector/internal/config"
	"github.com/hexiosec/asm-cloud-connector/internal/http"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
	"github.com/hexiosec/asm-cloud-connector/internal/version"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	debugMode = flag.Bool("debug", false, "Enable debug output")
)

func main() {
	// Setup
	flag.Parse()

	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		log.Warn().Err(err).Msg("Could not load .env file")
	}

	logEnv, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		logEnv = "info"
	}

	logLevel, err := zerolog.ParseLevel(logEnv)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse log level")
	}

	zerolog.SetGlobalLevel(logLevel)

	// Add file and line number to log output
	log.Logger = log.With().Caller().Logger()

	if *debugMode {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	} else {
		log.Logger = log.Output(os.Stdout)
	}

	// Main
	logger.GetGlobalLogger().Info().Msg("Starting version check")
	ctx := context.Background()

	cfg := &config.Config{}

	http := http.NewHttpService(cfg, "hexiosec-cloud-connector")
	checker, err := version.NewChecker(http)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not init version checker")
	}

	checker.LogVersion(ctx)
	logger.GetGlobalLogger().Info().Msg("Done")
}
