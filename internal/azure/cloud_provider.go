package azure

import (
	"context"

	cloud_provider_t "github.com/hexiosec/asm-cloud-connector/internal/cloud_provider/types"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
)

type AzureProvider struct {
	cfg     *config.AzureCloudProvider
	wrapper IAzureWrapper
}

func NewAzureProvider(cfg *config.Config) (cloud_provider_t.CloudProvider, error) {
	wrapper, err := NewWrapper()
	if err != nil {
		return nil, err
	}

	return &AzureProvider{
		cfg:     cfg.Azure,
		wrapper: wrapper,
	}, nil
}

func (c *AzureProvider) GetName() string {
	return "Azure"
}

func (c *AzureProvider) Authenticate(ctx context.Context) error {
	if err := c.wrapper.CheckConnection(ctx); err != nil {
		return err
	}

	logger.GetLogger(ctx).Debug().Msg("authentication successful")
	return nil
}

func (c *AzureProvider) GetAPIKey(ctx context.Context) (string, error) {
	return "", cloud_provider_t.ErrNoAPIKey
}

func (c *AzureProvider) GetResources(ctx context.Context) ([]string, error) {
	if err := c.wrapper.InitResourceGraph(ctx); err != nil {
		logger.GetLogger(ctx).Warn().Err(err).Msgf("failed to create azure resource graph client, unable to check for any resources")
		return []string{}, nil
	}

	defs := []struct {
		name    string
		enabled bool
		f       func(ctx context.Context) ([]string, error)
	}{
		{"Public IPs", c.cfg.Services.CheckPublicIPAddresses, c.wrapper.GetPublicIPs},
		{"Public IP DNS", c.cfg.Services.CheckPublicIPAddresses, c.wrapper.GetPublicIPDNSNames},
		{"Application Gateways", c.cfg.Services.CheckApplicationGateways, c.wrapper.GetApplicationGatewayHostnames},
		{"Application Gateway Certificates", c.cfg.Services.CheckApplicationGatewayCertificates, c.wrapper.GetApplicationGatewayCertificateDomains},
		{"Front Door (Classic)", c.cfg.Services.CheckFrontDoorClassic, c.wrapper.GetFrontDoorClassicHostnames},
		{"Front Door (AFD)", c.cfg.Services.CheckFrontDoorAfd, c.wrapper.GetFrontDoorAfdHostnames},
		{"Traffic Manager", c.cfg.Services.CheckTrafficManager, c.wrapper.GetTrafficManagerFQDNs},
		{"DNS Zones", c.cfg.Services.CheckDNSZones, c.wrapper.GetDNSZones},
		{"DNS Records", c.cfg.Services.CheckDNSRecords, c.wrapper.GetDNSRecordFQDNs},
		{"Storage (Web)", c.cfg.Services.CheckStorageStaticWebsites, c.wrapper.GetStorageWebEndpoints},
		{"CDN Endpoints", c.cfg.Services.CheckCDNEndpoints, c.wrapper.GetCDNEndpointHostnames},
		{"App Services", c.cfg.Services.CheckAppServices, c.wrapper.GetAppServiceHostnames},
		{"Azure SQL", c.cfg.Services.CheckSQLServers, c.wrapper.GetSQLServerFQDNs},
		{"Cosmos DB", c.cfg.Services.CheckCosmosDB, c.wrapper.GetCosmosDocumentEndpoints},
		{"Redis", c.cfg.Services.CheckRedisCache, c.wrapper.GetRedisHostnames},
	}

	resources := []string{}
	for _, def := range defs {
		if !def.enabled {
			logger.GetLogger(ctx).Trace().Msgf("skipping %s discovery; check disabled", def.name)
			continue
		}

		res, err := def.f(ctx)
		if err != nil {
			logger.GetLogger(ctx).Warn().Err(err).Msgf("failed to get %s resources", def.name)
			continue
		}

		resources = append(resources, res...)
	}

	logger.GetLogger(ctx).Info().Int("resource_count", len(resources)).Msg("resource discovery complete")
	return resources, nil
}
