package aws

import (
	"context"
	"fmt"

	cloud_provider_t "github.com/hexiosec/asm-cloud-connector/internal/cloud_provider/types"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
)

type AWSProvider struct {
	cfg     *config.AWSCloudProvider
	wrapper IAWSWrapper
}

func NewAWSProvider(cfg *config.Config) (cloud_provider_t.CloudProvider, error) {
	return &AWSProvider{
		cfg: cfg.AWS,
	}, nil
}

func (c *AWSProvider) GetName() string {
	return "AWS"
}

func (c *AWSProvider) Authenticate(ctx context.Context) error {
	wrapper, err := NewWrapper(ctx, c.cfg.DefaultRegion)
	if err != nil {
		return err
	}

	if err := wrapper.CheckConnection(ctx); err != nil {
		return err
	}

	c.wrapper = wrapper
	logger.GetLogger(ctx).Debug().Msg("authentication successful")
	return nil
}

func (c *AWSProvider) GetAPIKey(ctx context.Context) (string, error) {
	if c.cfg.APIKeySecret == nil {
		return "", cloud_provider_t.ErrNoAPIKey
	}

	return c.wrapper.GetSecretString(ctx, *c.cfg.APIKeySecret)
}

func (c *AWSProvider) GetResources(ctx context.Context) ([]string, error) {
	// Use the default config
	if !c.cfg.ListAllAccounts && len(c.cfg.Accounts) == 0 {
		return getResources(ctx, c.wrapper, c.cfg.Services, []string{})
	}

	var err error
	accounts := c.cfg.Accounts
	if c.cfg.ListAllAccounts {
		accounts, err = c.wrapper.ListAllAccounts(ctx)
		if err != nil {
			return nil, err
		}
	}

	resources := []string{}
	for _, account := range accounts {
		ctx = logger.WithLogger(ctx, logger.GetLogger(ctx).With().Str("account", account).Logger())
		role := fmt.Sprintf("arn:aws:iam::%s:role/%s", account, *c.cfg.AssumeRole)
		logger.GetLogger(ctx).Trace().Msgf("assuming role %s", role)

		assumeWrapper, err := c.wrapper.AssumeRole(ctx, role)
		if err != nil {
			logger.GetLogger(ctx).Warn().Err(err).Msgf("unable to load config with role %s, skipping account %s", role, account)
			continue
		}

		resources, err = getResources(ctx, assumeWrapper, c.cfg.Services, resources)
		if err != nil {
			return nil, fmt.Errorf("failed to get resources for account %s %w", account, err)
		}
	}

	return resources, nil
}

func getResources(ctx context.Context, wrapper IAWSWrapper, services *config.AWSServices, resources []string) ([]string, error) {
	var err error

	regions, err := wrapper.GetRegions(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not determine active regions, %w", err)
	}

	defs := []struct {
		name    string
		enabled bool
		f       func(ctx context.Context, resources []string) ([]string, error)
	}{
		{"EC2", services.CheckEC2, wrapper.GetEC2Resources},
		{"EIP", services.CheckEIP, wrapper.GetEIPResources},
		{"ELB", services.CheckELB, wrapper.GetELBResources},
		{"S3", services.CheckS3, wrapper.GetS3Resources},
		{"ACM", services.CheckACM, wrapper.GetACMResources},
		{"Route53", services.CheckRoute53, wrapper.GetRoute53Resources},
		{"CloudFront", services.CheckCloudFront, wrapper.GetCloudFrontResources},
		{"APIGateway", services.CheckAPIGateway, wrapper.GetAPIGatewayResources},
		{"APIGatewayV2", services.CheckAPIGatewayV2, wrapper.GetAPIGatewayV2Resources},
		{"EKS", services.CheckEKS, wrapper.GetEKSResources},
		{"RDS", services.CheckRDS, wrapper.GetRDSResources},
		{"OpenSearch", services.CheckOpenSearch, wrapper.GetOpenSearchResources},
		{"Lambda", services.CheckLambda, wrapper.GetLambdaResources},
	}

	for _, def := range defs {
		if !def.enabled {
			logger.GetLogger(ctx).Trace().Msgf("skipping %s discovery; check disabled", def.name)
			continue
		}

		for _, region := range regions {
			ctx = logger.WithLogger(ctx, logger.GetLogger(ctx).With().Str("region", region).Logger())
			logger.GetLogger(ctx).Trace().Msgf("checking region %s", region)
			wrapper.ChangeRegion(region)

			resources, err = def.f(ctx, resources)
			if err != nil {
				logger.GetLogger(ctx).Warn().Err(err).Msgf("failed to get %s resources", def.name)
			}
		}

		wrapper.ResetRegion()
	}
	return resources, nil
}
