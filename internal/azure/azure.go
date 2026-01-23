package azure

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/hexiosec/asm-cloud-connector/internal/util"
)

type IAzureWrapper interface {
	CheckConnection(ctx context.Context) error
	InitResourceGraph(ctx context.Context) error
	GetPublicIPs(ctx context.Context) ([]string, error)
	GetPublicIPDNSNames(ctx context.Context) ([]string, error)
	GetApplicationGatewayHostnames(ctx context.Context) ([]string, error)
	GetApplicationGatewayCertificateDomains(ctx context.Context) ([]string, error)
	GetFrontDoorClassicHostnames(ctx context.Context) ([]string, error)
	GetFrontDoorAfdHostnames(ctx context.Context) ([]string, error)
	GetTrafficManagerFQDNs(ctx context.Context) ([]string, error)
	GetDNSZones(ctx context.Context) ([]string, error)
	GetDNSRecordFQDNs(ctx context.Context) ([]string, error)
	GetStorageWebEndpoints(ctx context.Context) ([]string, error)
	GetCDNEndpointHostnames(ctx context.Context) ([]string, error)
	GetAppServiceHostnames(ctx context.Context) ([]string, error)
	GetSQLServerFQDNs(ctx context.Context) ([]string, error)
	GetCosmosDocumentEndpoints(ctx context.Context) ([]string, error)
	GetRedisHostnames(ctx context.Context) ([]string, error)
}

type AzureWrapper struct {
	cred      *azidentity.DefaultAzureCredential
	argClient *armresourcegraph.Client
}

func NewWrapper() (IAzureWrapper, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("azure: failed to get default credentials, %w", err)
	}
	return &AzureWrapper{cred: cred}, nil
}

const azureScopeARM = "https://management.azure.com/.default"

// Return nil if able to get a token and therefore can authenticate
// doesn't check that the required permissions are set
func (w *AzureWrapper) CheckConnection(ctx context.Context) error {
	// Try to get a token for ARM (Azure Resource Manager)
	_, err := w.cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{azureScopeARM},
	})
	if err != nil {
		return fmt.Errorf("azure: failed to get token, %w", err)
	}

	return nil
}

func (w *AzureWrapper) InitResourceGraph(ctx context.Context) error {
	client, err := armresourcegraph.NewClient(w.cred, nil)
	if err != nil {
		return fmt.Errorf("azure: failed to create resource graph client, %w", err)
	}

	w.argClient = client
	return nil
}

func (w *AzureWrapper) GetPublicIPs(ctx context.Context) ([]string, error) {
	query := `
	 	Resources
		| where type =~ 'microsoft.network/publicipaddresses'
		| extend resource = tostring(properties.ipAddress)
		| where isnotempty(resource)
		| distinct resource
	`

	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) GetPublicIPDNSNames(ctx context.Context) ([]string, error) {
	query := `
	 	Resources
		| where type =~ 'microsoft.network/publicipaddresses'
		| extend resource = tostring(properties.dnsSettings.fqdn)
		| where isnotempty(resource)
		| distinct resource
	`

	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) GetApplicationGatewayHostnames(ctx context.Context) ([]string, error) {
	query := `
		Resources
		| where type =~ 'microsoft.network/applicationgateways'
		| mv-expand l = properties.httpListeners
		| extend resource = tostring(l.properties.hostName)
		| where isnotempty(resource)
		| distinct resource
	`
	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) GetApplicationGatewayCertificateDomains(ctx context.Context) ([]string, error) {
	query := `
		Resources
		| where type =~ 'microsoft.network/applicationgateways'
		| mv-expand c = properties.sslCertificates
		| extend resource = tostring(c.properties.publicCertData)
		| where isnotempty(resource)
		| distinct resource
	`

	certData, err := w.queryResourceGraph(ctx, query)
	if err != nil {
		return nil, err
	}

	results := []string{}
	seen := map[string]struct{}{}
	for _, data := range certData {
		domains, err := extractCertificateDomains(data)
		if err != nil {
			return nil, err
		}

		for _, domain := range domains {
			if _, ok := seen[domain]; ok {
				continue
			}
			seen[domain] = struct{}{}
			results = append(results, domain)
		}
	}

	return results, nil
}

func (w *AzureWrapper) GetFrontDoorClassicHostnames(ctx context.Context) ([]string, error) {
	query := `
		Resources
		| where type =~ 'microsoft.network/frontdoors'
		| mv-expand fe = properties.frontendEndpoints
		| extend resource = tostring(fe.properties.hostName)
		| where isnotempty(resource)
		| distinct resource
	`
	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) GetFrontDoorAfdHostnames(ctx context.Context) ([]string, error) {
	query := `
		Resources
		| where type =~ 'microsoft.cdn/profiles/afdendpoints'
		| extend resource = tostring(properties.hostName)
		| where isnotempty(resource)
		| distinct resource
	`
	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) GetTrafficManagerFQDNs(ctx context.Context) ([]string, error) {
	query := `
		Resources
		| where type =~ 'microsoft.network/trafficmanagerprofiles'
		| extend resource = tostring(properties.dnsConfig.fqdn)
		| where isnotempty(resource)
		| distinct resource
	`
	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) GetDNSZones(ctx context.Context) ([]string, error) {
	query := `
		Resources
		| where type =~ 'microsoft.network/dnszones'
		| extend resource = tostring(name)
		| where isnotempty(resource)
		| distinct resource
	`
	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) GetDNSRecordFQDNs(ctx context.Context) ([]string, error) {
	query := `
		Resources
		| where type =~ 'microsoft.network/dnszones/A' or type =~ 'microsoft.network/dnszones/CNAME'
		| extend resource = tostring(properties.fqdn)
		| where isnotempty(resource)
		| distinct resource
	`
	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) GetStorageWebEndpoints(ctx context.Context) ([]string, error) {
	query := `
		Resources
		| where type =~ 'microsoft.storage/storageaccounts'
		| extend resource = tostring(properties.primaryEndpoints.web)
		| where isnotempty(resource)
		| distinct resource
	`
	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) GetCDNEndpointHostnames(ctx context.Context) ([]string, error) {
	query := `
		Resources
		| where type =~ 'microsoft.cdn/profiles/endpoints'
		| extend resource = tostring(properties.hostName)
		| where isnotempty(resource)
		| distinct resource
	`
	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) GetAppServiceHostnames(ctx context.Context) ([]string, error) {
	query := `
		Resources
		| where type =~ 'microsoft.web/sites'
		| mv-expand h = properties.hostNames
		| extend resource = tostring(h)
		| where isnotempty(resource)
		| distinct resource
	`
	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) GetSQLServerFQDNs(ctx context.Context) ([]string, error) {
	query := `
		Resources
		| where type =~ 'microsoft.sql/servers'
		| extend resource = tostring(properties.fullyQualifiedDomainName)
		| where isnotempty(resource)
		| distinct resource
	`
	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) GetCosmosDocumentEndpoints(ctx context.Context) ([]string, error) {
	query := `
		Resources
		| where type =~ 'microsoft.documentdb/databaseaccounts'
		| extend resource = tostring(properties.documentEndpoint)
		| where isnotempty(resource)
		| distinct resource
	`
	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) GetRedisHostnames(ctx context.Context) ([]string, error) {
	query := `
		Resources
		| where type =~ 'microsoft.cache/redis'
		| extend resource = tostring(properties.hostName)
		| where isnotempty(resource)
		| distinct resource
	`
	return w.queryResourceGraph(ctx, query)
}

func (w *AzureWrapper) queryResourceGraph(ctx context.Context, query string) ([]string, error) {
	resources := []string{}
	req := armresourcegraph.QueryRequest{Query: &query, Options: &armresourcegraph.QueryRequestOptions{}}
	for {
		resp, err := w.argClient.Resources(ctx, req, nil)
		if err != nil {
			return nil, fmt.Errorf("azure: resource graph query failed, %w", err)
		}

		res, err := decodeResourceGraphData(resp.Data)
		if err != nil {
			return nil, fmt.Errorf("azure: failed to decode response, %w", err)
		}

		resources = append(resources, res...)

		if resp.SkipToken == nil {
			break
		}

		req.Options.SkipToken = resp.SkipToken
	}

	return resources, nil
}

func extractCertificateDomains(certData string) ([]string, error) {
	trimmed := strings.TrimSpace(certData)
	if trimmed == "" {
		return []string{}, nil
	}

	var derBytes []byte
	if strings.Contains(trimmed, "BEGIN CERTIFICATE") {
		block, _ := pem.Decode([]byte(trimmed))
		if block == nil {
			return nil, fmt.Errorf("azure: failed to decode PEM certificate data")
		}
		derBytes = block.Bytes
	} else {
		decoded, err := base64.StdEncoding.DecodeString(trimmed)
		if err != nil {
			decoded, err = base64.RawStdEncoding.DecodeString(trimmed)
			if err != nil {
				return nil, fmt.Errorf("azure: failed to decode base64 certificate data: %w", err)
			}
		}
		derBytes = decoded
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, fmt.Errorf("azure: failed to parse certificate data: %w", err)
	}

	results := []string{}
	seen := map[string]struct{}{}
	for _, name := range cert.DNSNames {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		results = append(results, name)
	}

	if cn := strings.TrimSpace(cert.Subject.CommonName); cn != "" {
		if _, ok := seen[cn]; !ok {
			results = append(results, cn)
		}
	}

	return results, nil
}

func decodeResourceGraphData(data any) ([]string, error) {
	result := []string{}

	records := []struct {
		Resource *string `mapstructure:"resource"`
	}{}

	if err := util.MapStructDecodeAndValidate(data, &records); err != nil {
		return nil, err
	}

	for _, r := range records {
		if r.Resource != nil {
			result = append(result, *r.Resource)
		}
	}

	return result, nil
}
