package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hexiosec/asm-cloud-connector/internal/api"
	"github.com/hexiosec/asm-cloud-connector/internal/cloud_provider"
	cloud_provider_t "github.com/hexiosec/asm-cloud-connector/internal/cloud_provider/types"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
	"github.com/hexiosec/asm-cloud-connector/internal/connector"
	"github.com/hexiosec/asm-cloud-connector/internal/http"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
	"github.com/hexiosec/asm-cloud-connector/internal/version"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	cfgFilePath string = "./config.yml"
	debugMode   bool   = false
)

func SetCfgFilePath(v string) {
	cfgFilePath = v
}

func SetDebugMode(v bool) {
	debugMode = v
}

// Will load the .env file if available and setup
func Setup() error {
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		logger.GetGlobalLogger().Warn().Err(err).Msg("Could not load .env file")
	}

	logEnv, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		logEnv = "info"
	}

	logLevel, err := zerolog.ParseLevel(logEnv)
	if err != nil {
		logger.GetGlobalLogger().Warn().Err(err).Msg("Could not parse log level")
		return fmt.Errorf("core: could not parse log level, %w", err)
	}

	zerolog.SetGlobalLevel(logLevel)

	// Add file and line number to log output
	log.Logger = log.With().Caller().Logger()

	if debugMode {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	} else {
		log.Logger = log.Output(os.Stdout)
	}

	return nil
}

func Run(ctx context.Context) error {
	// Load config
	cfg := config.Provider(cfgFilePath)

	// Check for a new version
	http := http.NewHttpService(cfg, "hexiosec-cloud-connector")
	checker, err := version.NewChecker(http)
	if err != nil {
		logger.GetLogger(ctx).Warn().Err(err).Msg("Could not init version checker")
		return fmt.Errorf("core: could not init version checker, %w", err)
	}
	checker.LogVersion(ctx)

	logger.GetLogger(ctx).Info().Str("scan_id", cfg.ScanID).Msg("Getting cloud resources")

	// Setup Cloud Provider
	cp, err := cloud_provider.NewCloudProvider(cfg)
	if err != nil {
		logger.GetLogger(ctx).Warn().Err(err).Msg("Could not init cloud provider")
		return fmt.Errorf("core: could not init cloud provider, %w", err)
	}
	ctx = logger.WithLogger(ctx, logger.GetLogger(ctx).With().Str("cloud_provider", cp.GetName()).Logger())

	if err := cp.Authenticate(ctx); err != nil {
		logger.GetLogger(ctx).Warn().Err(err).Msg("Could not authenticate with cloud provider")
		return fmt.Errorf("core: could not authenticate with cloud provider, %w", err)
	}
	logger.GetLogger(ctx).Debug().Msg("Cloud provider authentication successful")

	// Try get the API key from the cloud provider first
	apiKey, err := cp.GetAPIKey(ctx)
	var ok bool
	if err != nil {
		if !errors.Is(err, cloud_provider_t.ErrNoAPIKey) {
			logger.GetLogger(ctx).Warn().Err(err).Msg("Failed to get api key")
			return fmt.Errorf("core: failed to get api key, %w", err)

		}

		// Default to getting API key via ENV if cloud provider doesn't have it
		apiKey, ok = os.LookupEnv("API_KEY")
		if !ok || strings.TrimSpace(apiKey) == "" {
			logger.GetLogger(ctx).Warn().Msg("API key not provided by cloud provider or en")
			return fmt.Errorf("core: API key not provided by cloud provider or env API_KEY")
		}
	}

	// Setup SDK and connector
	sdk, err := api.NewAPI(cfg, "hexiosec-cloud-connector", apiKey)
	if err != nil {
		logger.GetLogger(ctx).Warn().Err(err).Msg("Could not init ASM SDK")
		return fmt.Errorf("core: could not init ASM SDK, %w", err)

	}

	conn, err := connector.NewConnector(cfg, sdk)
	if err != nil {
		logger.GetLogger(ctx).Warn().Err(err).Msg("Could not init Hexiosec ASM connecto")
		return fmt.Errorf("core: could not init Hexiosec ASM connector %w", err)
	}

	if err := conn.Authenticate(ctx); err != nil {
		logger.GetLogger(ctx).Warn().Err(err).Msg("Could not authenticate with Hexiosec ASM connector")
		return fmt.Errorf("core: could not authenticate with Hexiosec ASM connector, %w", err)
	}
	logger.GetLogger(ctx).Debug().Msg("Cloud connector authentication successful")

	// Get resources and sync
	resources, err := cp.GetResources(ctx)
	if err != nil {
		logger.GetLogger(ctx).Warn().Err(err).Msg("Could not get resources of cloud provider")
		return fmt.Errorf("core: could not get resources of cloud provider, %w", err)
	}
	logger.GetLogger(ctx).Debug().Interface("resources", resources).Msgf("Got %d resources", len(resources))

	if err := conn.SyncResources(ctx, resources); err != nil {
		logger.GetLogger(ctx).Warn().Err(err).Msg("Could not sync resources with Hexiosec ASM connector")
		return fmt.Errorf("core: could not sync resources with Hexiosec ASM connector, %w", err)
	}

	logger.GetLogger(ctx).Info().Msg("Cloud resource sync successful with Hexiosec ASM")
	return nil
}
