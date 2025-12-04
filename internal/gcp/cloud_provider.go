package gcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	certificatemanagerpb "cloud.google.com/go/certificatemanager/apiv1/certificatemanagerpb"

	cloud_provider_t "github.com/hexiosec/asm-cloud-connector/internal/cloud_provider/types"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
	"github.com/hexiosec/asm-cloud-connector/internal/util"
)

type GCPProvider struct {
	cfg     *config.GCPCloudProvider
	wrapper IGCPWrapper
}

func NewGCPProvider(cfg *config.Config) (cloud_provider_t.CloudProvider, error) {
	wrapper, err := NewWrapper()
	if err != nil {
		return nil, err
	}

	return &GCPProvider{
		cfg:     cfg.GCP,
		wrapper: wrapper,
	}, nil
}

func (c *GCPProvider) GetName() string {
	return "GCP"
}

func (c *GCPProvider) Authenticate(ctx context.Context) error {
	if err := c.wrapper.CheckConnection(ctx); err != nil {
		return err
	}

	logger.GetLogger(ctx).Debug().Msg("authentication successful")
	return nil
}

// API Key will always be provided via the env variable
func (c *GCPProvider) GetAPIKey(ctx context.Context) (string, error) {
	return "", cloud_provider_t.ErrNoAPIKey
}

func (c *GCPProvider) GetResources(ctx context.Context) ([]string, error) {
	defs := map[string]struct {
		enabled bool
		getter  func(ctx context.Context, asset *assetpb.Asset, data map[string]any) ([]string, error)
	}{
		"dns.googleapis.com/ResourceRecordSet": {
			enabled: c.cfg.Services.CheckDNSResourceRecordSet,
			getter:  c.getResourcesFromResourceRecordSet,
		},
		"dns.googleapis.com/ManagedZone": {
			enabled: c.cfg.Services.CheckDNSManagedZone,
			getter:  c.getResourcesFromManagedZone,
		},
		"compute.googleapis.com/Instance": {
			enabled: c.cfg.Services.CheckComputeInstance,
			getter:  c.getResourcesFromInstance,
		},
		"compute.googleapis.com/Address": {
			enabled: c.cfg.Services.CheckComputeAddress,
			getter:  c.getResourcesFromAddress,
		},
		"storage.googleapis.com/Bucket": {
			enabled: c.cfg.Services.CheckStorageBucket,
			getter:  c.getResourcesFromBucket,
		},
		"cloudfunctions.googleapis.com/Function": {
			enabled: c.cfg.Services.CheckCloudFunction,
			getter:  c.getResourcesFromFunction,
		},
		"run.googleapis.com/Service": {
			enabled: c.cfg.Services.CheckRunService,
			getter:  c.getResourcesFromRunService,
		},
		"run.googleapis.com/DomainMapping": {
			enabled: c.cfg.Services.CheckRunDomainMapping,
			getter:  c.getResourcesFromDomainMapping,
		},
		"apigateway.googleapis.com/Gateway": {
			enabled: c.cfg.Services.CheckAPIGateway,
			getter:  c.getResourcesFromAPIGateway,
		},
		"sqladmin.googleapis.com/Instance": {
			enabled: c.cfg.Services.CheckSQLInstance,
			getter:  c.getResourcesFromSQLInstance,
		},
		"compute.googleapis.com/ForwardingRule": {
			enabled: c.cfg.Services.CheckComputeForwardingRule,
			getter:  c.getResourcesFromForwardingRule,
		},
		"compute.googleapis.com/GlobalForwardingRule": {
			enabled: c.cfg.Services.CheckComputeGlobalForwarding,
			getter:  c.getResourcesFromForwardingRule,
		},
		"compute.googleapis.com/UrlMap": {
			enabled: c.cfg.Services.CheckComputeURLMap,
			getter:  c.getResourcesFromURLMap,
		},
		"appengine.googleapis.com/Service": {
			enabled: c.cfg.Services.CheckAppEngineService,
			getter:  c.getResourcesFromAppEngineService,
		},
		"container.googleapis.com/Cluster": {
			enabled: c.cfg.Services.CheckGKECluster,
			getter:  c.getResourcesFromCluster,
		},
	}

	enabledAssetTypes := make([]string, 0, len(defs))
	for k, v := range defs {
		if v.enabled {
			enabledAssetTypes = append(enabledAssetTypes, k)
		}
	}
	logger.GetLogger(ctx).Debug().Strs("asset_types", enabledAssetTypes).Msg("enabled asset types")

	var resources []string
	for _, project := range c.cfg.Projects {
		logger.GetLogger(ctx).Debug().Msgf("searching project %s", project)
		if len(enabledAssetTypes) > 0 {
			assets, err := c.wrapper.GetAssets(ctx, project, enabledAssetTypes)
			if err != nil {
				return nil, err
			}

			for _, asset := range assets {
				logger.GetLogger(ctx).Trace().Str("asset_type", asset.AssetType).Msg("processing asset")
				def, ok := defs[asset.AssetType]
				if !ok {
					// Should not be possible
					logger.GetLogger(ctx).Warn().Str("asset_type", asset.AssetType).Msg("missing code to handle asset type")
					continue
				}

				data := asset.GetResource().GetData().AsMap()

				assetResources, err := def.getter(ctx, asset, data)
				if err != nil {
					if errType := (&ValidationErr{}); errors.As(err, &errType) {
						logger.GetLogger(ctx).Warn().Str("asset_type", asset.AssetType).Err(err).Msg("failed to decode asset, skipping")
						continue
					}
					return nil, err
				}

				resources = append(resources, assetResources...)
			}
		}

		// Certificates have to be retrieved separately because they are not available on the Assets API
		if c.cfg.Services.CheckCertificates {
			logger.GetLogger(ctx).Debug().Msg("fetching certificates")
			certs, err := c.wrapper.GetCertificates(ctx, project)
			if err != nil {
				return nil, err
			}
			logger.GetLogger(ctx).Trace().Int("certificate_count", len(certs)).Msg("certificates retrieved")

			resources = append(resources, extractDomainsFromCertificates(certs)...)
		}
	}

	logger.GetLogger(ctx).Info().Int("resource_count", len(resources)).Msg("resource discovery complete")
	return resources, nil
}

func extractDomainsFromCertificates(certificates []*certificatemanagerpb.Certificate) []string {
	var domains []string
	for _, cert := range certificates {
		for _, san := range cert.GetSanDnsnames() {
			domains = append(domains, strings.TrimSuffix(san, "."))
		}
	}

	return domains
}

func isIP(s string) bool {
	return net.ParseIP(s) != nil
}

func (c *GCPProvider) getResourcesFromResourceRecordSet(_ context.Context, _ *assetpb.Asset, data map[string]any) ([]string, error) {
	r := resourceRecordSet{}
	if err := util.MapStructDecodeAndValidate(data, &r); err != nil {
		return nil, &ValidationErr{err}
	}

	var resources []string

	// Only keep resource names when the DNS record is for a subdomain
	// Other records will be found & created by ASM
	if r.Name != nil && r.Type != nil && *r.Name != "" {
		t := strings.ToUpper(*r.Type)
		if t == "A" || t == "AAAA" || t == "CNAME" {
			resources = append(resources, *r.Name)
		}
	}

	// Always keep IP data from the record
	for _, v := range r.RRDatas {
		if v != nil && isIP(*v) {
			resources = append(resources, *v)
		}
	}

	return resources, nil
}

func (c *GCPProvider) getResourcesFromManagedZone(_ context.Context, _ *assetpb.Asset, data map[string]any) ([]string, error) {
	if dns, ok := data["dnsName"].(string); ok && dns != "" {
		return []string{dns}, nil
	}

	return nil, nil
}

func (c *GCPProvider) getResourcesFromInstance(_ context.Context, _ *assetpb.Asset, data map[string]any) ([]string, error) {
	i := instance{}
	if err := util.MapStructDecodeAndValidate(data, &i); err != nil {
		return nil, &ValidationErr{err}
	}

	var resources []string
	for _, n := range i.NetworkInterfaces {
		for _, ac := range n.AccessConfigs {
			if ac.NatIP != nil {
				resources = append(resources, *ac.NatIP)
			}
		}
	}

	return resources, nil
}

func (c *GCPProvider) getResourcesFromAddress(_ context.Context, _ *assetpb.Asset, data map[string]any) ([]string, error) {
	a := address{}
	if err := util.MapStructDecodeAndValidate(data, &a); err != nil {
		return nil, &ValidationErr{err}
	}

	if a.Address != nil && a.Type != nil && *a.Type == "EXTERNAL" {
		return []string{*a.Address}, nil
	}

	return nil, nil
}

func (c *GCPProvider) getResourcesFromBucket(ctx context.Context, asset *assetpb.Asset, data map[string]any) ([]string, error) {
	fullName, ok := data["name"].(string)
	if !ok {
		return nil, nil
	}

	parts := strings.Split(fullName, "/")
	bucketName := parts[len(parts)-1]
	if bucketName == "" {
		return nil, nil
	}

	if c.wrapper.IsBucketPublic(ctx, bucketName) {
		return []string{fmt.Sprintf("https://%s.storage.googleapis.com/", bucketName)}, nil
	}

	return nil, nil
}

func (c *GCPProvider) getResourcesFromFunction(_ context.Context, _ *assetpb.Asset, data map[string]any) ([]string, error) {
	f := function{}
	if err := util.MapStructDecodeAndValidate(data, &f); err != nil {
		return nil, &ValidationErr{err}
	}

	if f.HTTPSTrigger != nil && f.HTTPSTrigger.URL != nil {
		return []string{*f.HTTPSTrigger.URL}, nil
	}

	return nil, nil
}

func (c *GCPProvider) getResourcesFromRunService(_ context.Context, _ *assetpb.Asset, data map[string]any) ([]string, error) {
	s := service{}
	if err := util.MapStructDecodeAndValidate(data, &s); err != nil {
		return nil, &ValidationErr{err}
	}

	if s.Status != nil && s.Status.URL != nil {
		return []string{*s.Status.URL}, nil
	}

	return nil, nil
}

func (c *GCPProvider) getResourcesFromDomainMapping(_ context.Context, _ *assetpb.Asset, data map[string]any) ([]string, error) {
	dm := domainMapping{}
	if err := util.MapStructDecodeAndValidate(data, &dm); err != nil {
		return nil, &ValidationErr{err}
	}

	if dm.Metadata != nil && dm.Metadata.Name != nil {
		return []string{*dm.Metadata.Name}, nil
	}

	return nil, nil
}

func (c *GCPProvider) getResourcesFromAPIGateway(_ context.Context, _ *assetpb.Asset, data map[string]any) ([]string, error) {
	if hostname, ok := data["defaultHostname"].(string); ok && hostname != "" {
		return []string{"https://" + hostname}, nil
	}

	return nil, nil
}

func (c *GCPProvider) getResourcesFromSQLInstance(_ context.Context, _ *assetpb.Asset, data map[string]any) ([]string, error) {
	si := sqlInstance{}
	if err := util.MapStructDecodeAndValidate(data, &si); err != nil {
		return nil, &ValidationErr{err}
	}

	var resources []string
	for _, ip := range si.IPAddresses {
		if ip != nil && ip.IPAddress != nil && ip.Type != nil && *ip.Type == "PRIMARY" {
			resources = append(resources, *ip.IPAddress)
		}
	}

	return resources, nil
}

func (c *GCPProvider) getResourcesFromForwardingRule(_ context.Context, _ *assetpb.Asset, data map[string]any) ([]string, error) {
	f := forwardingRule{}
	if err := util.MapStructDecodeAndValidate(data, &f); err != nil {
		return nil, &ValidationErr{err}
	}

	if f.IPAddress != nil && f.LoadBalancingScheme != nil && strings.HasPrefix(*f.LoadBalancingScheme, "EXTERNAL") {
		return []string{*f.IPAddress}, nil
	}

	return nil, nil
}

func (c *GCPProvider) getResourcesFromURLMap(_ context.Context, _ *assetpb.Asset, data map[string]any) ([]string, error) {
	um := urlMap{}
	if err := util.MapStructDecodeAndValidate(data, &um); err != nil {
		return nil, &ValidationErr{err}
	}

	var resources []string
	for _, hr := range um.HostRules {
		if hr == nil {
			continue
		}
		for _, h := range hr.Hosts {
			if h == nil || *h == "" {
				continue
			}
			resources = append(resources, *h)
		}
	}

	return resources, nil
}

func (c *GCPProvider) getResourcesFromAppEngineService(_ context.Context, asset *assetpb.Asset, data map[string]any) ([]string, error) {
	serviceID, ok := data["id"].(string)
	if !ok || serviceID == "" {
		return nil, nil
	}

	parent := asset.GetResource().GetParent()
	if parent == "" {
		return nil, nil
	}

	// parent will be of the form `projects/<PROJECT_ID>`
	parts := strings.Split(parent, "/")
	if len(parts) == 0 {
		return nil, nil
	}

	projectID := parts[len(parts)-1]
	if projectID == "" {
		return nil, nil
	}

	var host string
	if serviceID == "default" {
		host = fmt.Sprintf("%s.appspot.com", projectID)
	} else {
		host = fmt.Sprintf("%s-dot-%s.appspot.com", serviceID, projectID)
	}

	return []string{host}, nil
}

func (c *GCPProvider) getResourcesFromCluster(_ context.Context, _ *assetpb.Asset, data map[string]any) ([]string, error) {
	cl := cluster{}
	if err := util.MapStructDecodeAndValidate(data, &cl); err != nil {
		return nil, &ValidationErr{err}
	}

	if cl.Endpoint == nil || *cl.Endpoint == "" {
		return nil, nil
	}

	if cl.PrivateClusterConfig != nil &&
		cl.PrivateClusterConfig.EnablePrivateEndpoint != nil &&
		*cl.PrivateClusterConfig.EnablePrivateEndpoint {
		return nil, nil
	}

	return []string{*cl.Endpoint}, nil
}
