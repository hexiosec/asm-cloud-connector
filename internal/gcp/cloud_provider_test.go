package gcp

import (
	"context"
	"testing"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	certificatemanagerpb "cloud.google.com/go/certificatemanager/apiv1/certificatemanagerpb"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test_GetResources_UsesEnabledAssetTypes(t *testing.T) {
	provider, wrapper := newProviderWithWrapper(t, &config.GCPCloudProvider{
		CloudProvider: config.CloudProvider{Enabled: true},
		Projects:      []string{"PROJECT_ID"},
		Services: &config.GCPServices{
			CheckDNSResourceRecordSet: true,
		},
	})

	// Only ResourceRecordSet should be requested
	wrapper.On("GetAssets", "PROJECT_ID", []string{"dns.googleapis.com/ResourceRecordSet"}).Return([]*assetpb.Asset{}, nil)

	resources, err := provider.GetResources(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, resources)
}

func Test_GetResources_CertificatesCalledSeparately(t *testing.T) {
	provider, wrapper := newProviderWithWrapper(t, &config.GCPCloudProvider{
		CloudProvider: config.CloudProvider{Enabled: true},
		Projects:      []string{"PROJECT_ID"},
		Services: &config.GCPServices{
			CheckCertificates: true,
		},
	})

	wrapper.On("GetCertificates", "PROJECT_ID").Return([]*certificatemanagerpb.Certificate{}, nil)

	resources, err := provider.GetResources(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, resources)
}

func Test_GetResources_GetAssetsErrs_ReturnsErr(t *testing.T) {
	provider, wrapper := newProviderWithWrapper(t, &config.GCPCloudProvider{
		CloudProvider: config.CloudProvider{Enabled: true},
		Projects:      []string{"PROJECT_ID"},
		Services: &config.GCPServices{
			CheckDNSResourceRecordSet: true,
		},
	})

	// Only ResourceRecordSet should be requested
	wrapper.On("GetAssets", "PROJECT_ID", []string{"dns.googleapis.com/ResourceRecordSet"}).Return(nil, assert.AnError)

	resources, err := provider.GetResources(context.Background())
	assert.Error(t, err)
	assert.Empty(t, resources)
}

func Test_GetResources_GetCertificatesErrs_ReturnsErr(t *testing.T) {
	provider, wrapper := newProviderWithWrapper(t, &config.GCPCloudProvider{
		CloudProvider: config.CloudProvider{Enabled: true},
		Projects:      []string{"PROJECT_ID"},
		Services: &config.GCPServices{
			CheckCertificates: true,
		},
	})

	wrapper.On("GetCertificates", "PROJECT_ID").Return(nil, assert.AnError)

	resources, err := provider.GetResources(context.Background())
	assert.Error(t, err)
	assert.Empty(t, resources)
}

func Test_GetResources_Asset_ReturnsResource(t *testing.T) {
	provider, wrapper := newProviderWithWrapper(t, &config.GCPCloudProvider{
		CloudProvider: config.CloudProvider{Enabled: true},
		Projects:      []string{"PROJECT_ID"},
		Services: &config.GCPServices{
			CheckDNSManagedZone: true,
		},
	})

	data, err := structpb.NewStruct(map[string]any{"dnsName": "example.com"})
	if err != nil {
		panic(err)
	}

	// Only ResourceRecordSet should be requested
	wrapper.On("GetAssets", "PROJECT_ID", []string{"dns.googleapis.com/ManagedZone"}).Return([]*assetpb.Asset{
		{
			AssetType: "dns.googleapis.com/ManagedZone",
			Resource: &assetpb.Resource{
				Data: data,
			},
		},
	}, nil)

	resources, err := provider.GetResources(context.Background())
	assert.NoError(t, err)
	assert.Contains(t, resources, "example.com")
}

func Test_GetResources_AssetValidationErr_AssetSkipped(t *testing.T) {
	provider, wrapper := newProviderWithWrapper(t, &config.GCPCloudProvider{
		CloudProvider: config.CloudProvider{Enabled: true},
		Projects:      []string{"PROJECT_ID"},
		Services: &config.GCPServices{
			CheckDNSResourceRecordSet: true,
		},
	})

	data1, err := structpb.NewStruct(map[string]any{"rrdatas": []any{"192.168.0.1"}})
	if err != nil {
		panic(err)
	}

	// rrdatas should be a list, providing a number should mean mapstructure can't parse it triggering a validation error
	data2, err := structpb.NewStruct(map[string]any{"rrdatas": 123})
	if err != nil {
		panic(err)
	}

	// Only ResourceRecordSet should be requested
	wrapper.On("GetAssets", "PROJECT_ID", []string{"dns.googleapis.com/ResourceRecordSet"}).Return([]*assetpb.Asset{
		{
			AssetType: "dns.googleapis.com/ResourceRecordSet",
			Resource: &assetpb.Resource{
				Data: data1,
			},
		},
		{
			AssetType: "dns.googleapis.com/ResourceRecordSet",
			Resource: &assetpb.Resource{
				Data: data2,
			},
		},
	}, nil)

	resources, err := provider.GetResources(context.Background())
	assert.NoError(t, err)
	assert.Contains(t, resources, "192.168.0.1")
}

func newProviderWithWrapper(t *testing.T, cfg *config.GCPCloudProvider) (*GCPProvider, *MockWrapper) {
	t.Helper()
	wrapper := NewMockWrapper(t).(*MockWrapper)
	provider := &GCPProvider{
		cfg:     cfg,
		wrapper: wrapper,
	}
	return provider, wrapper
}
