package azure

import (
	"context"
	"testing"

	cloud_provider_t "github.com/hexiosec/asm-cloud-connector/internal/cloud_provider/types"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestAzureProvider_GetAPIKey_NoSecret(t *testing.T) {
	provider := &AzureProvider{
		cfg: &config.AzureCloudProvider{},
	}

	_, err := provider.GetAPIKey(context.Background())

	assert.ErrorIs(t, err, cloud_provider_t.ErrNoAPIKey)
}

func TestAzureProvider_Authenticate_CheckConnectionError(t *testing.T) {
	provider, wrapper := newProviderWithWrapper(t, &config.AzureCloudProvider{})

	wrapper.On("CheckConnection").Return(assert.AnError)

	err := provider.Authenticate(context.Background())

	assert.ErrorIs(t, err, assert.AnError)
}

func TestAzureProvider_Authenticate_CheckConnectionOk(t *testing.T) {
	provider, wrapper := newProviderWithWrapper(t, &config.AzureCloudProvider{})

	wrapper.On("CheckConnection").Return(nil)

	err := provider.Authenticate(context.Background())

	assert.NoError(t, err)
}

func TestAzureProvider_GetResources_UsesEnabledServices(t *testing.T) {
	provider, wrapper := newProviderWithWrapper(t, &config.AzureCloudProvider{
		CloudProvider: config.CloudProvider{Enabled: true},
		Services: &config.AzureServices{
			CheckAppServices: true,
		},
	})

	wrapper.On("InitResourceGraph").Return(nil)
	wrapper.On("GetAppServiceHostnames").Return(assetSlice("app.azurewebsites.net"), nil)

	resources, err := provider.GetResources(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, assetSlice("app.azurewebsites.net"), resources)
	wrapper.AssertNotCalled(t, "GetSQLServerFQDNs")
}

func TestAzureProvider_GetResources_ContinuesOnError(t *testing.T) {
	provider, wrapper := newProviderWithWrapper(t, &config.AzureCloudProvider{
		CloudProvider: config.CloudProvider{Enabled: true},
		Services: &config.AzureServices{
			CheckPublicIPAddresses: true,
		},
	})

	wrapper.On("InitResourceGraph").Return(nil)
	wrapper.On("GetPublicIPs").Return(nil, assert.AnError)
	wrapper.On("GetPublicIPDNSNames").Return(assetSlice("example.azure.com"), nil)

	resources, err := provider.GetResources(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, assetSlice("example.azure.com"), resources)
}

func TestAzureProvider_GetResources_ApplicationGatewayCertificates(t *testing.T) {
	provider, wrapper := newProviderWithWrapper(t, &config.AzureCloudProvider{
		CloudProvider: config.CloudProvider{Enabled: true},
		Services: &config.AzureServices{
			CheckApplicationGatewayCertificates: true,
		},
	})

	wrapper.On("InitResourceGraph").Return(nil)
	wrapper.On("GetApplicationGatewayCertificateDomains").Return(assetSlice("cert.example.com"), nil)

	resources, err := provider.GetResources(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, assetSlice("cert.example.com"), resources)
}

func TestAzureProvider_GetResources_InitResourceGraphError(t *testing.T) {
	provider, wrapper := newProviderWithWrapper(t, &config.AzureCloudProvider{
		CloudProvider: config.CloudProvider{Enabled: true},
		Services: &config.AzureServices{
			CheckAppServices: true,
		},
	})

	wrapper.On("InitResourceGraph").Return(assert.AnError)

	resources, err := provider.GetResources(context.Background())

	assert.NoError(t, err)
	assert.Empty(t, resources)
	wrapper.AssertNotCalled(t, "GetAppServiceHostnames")
}

func newProviderWithWrapper(t *testing.T, cfg *config.AzureCloudProvider) (*AzureProvider, *MockWrapper) {
	t.Helper()
	wrapper := NewMockWrapper(t).(*MockWrapper)
	provider := &AzureProvider{
		cfg:     cfg,
		wrapper: wrapper,
	}
	return provider, wrapper
}

func assetSlice(names ...string) []*cloud_provider_t.Asset {
	assets := make([]*cloud_provider_t.Asset, 0, len(names))
	for _, name := range names {
		assets = append(assets, &cloud_provider_t.Asset{Name: name})
	}
	return assets
}
