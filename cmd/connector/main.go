package main

import (
	"context"
	"flag"

	"github.com/hexiosec/asm-cloud-connector/internal/logger"
	"github.com/hexiosec/asm-cloud-connector/pkg/core"
)

var (
	debugMode   = flag.Bool("debug", false, "Enable debug output")
	cfgFilePath = flag.String("config", "./config.yml", "Path to config YAML")
)

func main() {
	flag.Parse()
	core.SetCfgFilePath(*cfgFilePath)
	core.SetDebugMode(*debugMode)

	if err := core.Setup(); err != nil {
		logger.GetGlobalLogger().Fatal().Err(err).Msg("failed to setup")
	}

	if err := core.Run(context.Background()); err != nil {
		logger.GetGlobalLogger().Fatal().Err(err).Msg("failed to run")
	}
}
