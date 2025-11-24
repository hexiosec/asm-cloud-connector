package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
	"github.com/hexiosec/asm-cloud-connector/pkg/core"
)

func main() {
	if err := core.Setup(); err != nil {
		logger.GetGlobalLogger().Fatal().Err(err).Msg("failed to setup")
	}

	lambda.Start(core.Run)
}
