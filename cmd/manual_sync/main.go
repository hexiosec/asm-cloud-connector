package main

import (
	"context"
	"flag"
	"os"

	"github.com/hexiosec/asm-cloud-connector/internal/api"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
	"github.com/hexiosec/asm-cloud-connector/internal/connector"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	debugMode        = flag.Bool("debug", false, "Enable debug output")
	scanID           = flag.String("scan-id", "", "Scan ID")
	seedLabel        = flag.String("seed-label", "", "Seed Label")
	deleteStaleSeeds = flag.Bool("delete-stale-seeds", true, "Delete seeds not in resource list")
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
	logger.GetGlobalLogger().Info().Msg("Starting manual sync")
	ctx := context.Background()

	if *scanID == "" {
		log.Fatal().Msg("Scan ID not set, use --scan-id")
	}

	if *seedLabel == "" {
		log.Fatal().Msg("Seed Label not set, use --seed-label")
	}

	if flag.NArg() == 0 {
		log.Fatal().Msg("No resources have been provided")
	}

	resources := flag.Args()
	log.Info().Interface("resources", resources).Msgf("%d resources", len(resources))

	cfg := &config.Config{
		ScanID:           *scanID,
		SeedTag:          *seedLabel,
		DeleteStaleSeeds: *deleteStaleSeeds,
	}

	apiKey, ok := os.LookupEnv("API_KEY")
	if !ok {
		log.Fatal().Msg("API_KEY environment variable not set")
	}

	sdk, err := api.NewAPI(cfg, "hexiosec-cloud-connector", apiKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not init ASM SDK")
	}

	conn, err := connector.NewConnector(cfg, sdk)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not init Hexiosec ASM connector")
	}

	if err := conn.Authenticate(ctx); err != nil {
		log.Fatal().Err(err).Msg("Could not authenticate with Hexiosec ASM connector")
	}

	if err := conn.SyncResources(ctx, resources); err != nil {
		log.Fatal().Err(err).Msg("Could not sync resources with Hexiosec ASM connector")
	}

	logger.GetGlobalLogger().Info().Msg("Done")
}
